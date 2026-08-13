package artwork

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"io"

	"github.com/BourgeoisBear/rasterm"
	"github.com/nfnt/resize"
	"github.com/qeesung/image2ascii/convert"
)

// Mode overrides automatic terminal detection, matching the `art` config
// field ("auto" | "ascii" | "off").
type Mode string

const (
	ModeAuto  Mode = "auto"
	ModeASCII Mode = "ascii"
	ModeOff   Mode = "off"
)

// cellAspect approximates a terminal cell's height-to-width ratio. Most
// monospace fonts render cells roughly twice as tall as they are wide, so
// covering `cols` terminal columns with a square-ish image needs about
// cols/2 rows — without this, art renders at native resolution mapped
// 1:1 to cells and comes out stretched (Sixel/ASCII) or simply huge
// (Kitty/iTerm2, which otherwise display at the image's native pixel size).
const cellAspect = 2.0

// Render encodes img for the terminal into out, dispatching on term (or
// forcing ASCII when mode is ModeASCII), scaled to fit within cols terminal
// columns. ModeOff callers should skip calling Render entirely; it is not
// handled here since there's nothing to render.
func Render(out io.Writer, img image.Image, term TermType, mode Mode, cols int) error {
	if cols <= 0 {
		cols = defaultCols
	}
	rows := targetRows(img, cols)

	if mode == ModeASCII {
		return renderASCII(out, img, cols, rows)
	}

	switch term {
	case TermKitty:
		// A stable ImageId/PlacementId, not just DstCols/DstRows, is what
		// makes re-sending the same cover art on every Bubble Tea repaint
		// (every ~500ms position tick, not just on track change) update
		// the existing placement in place. Without one, Kitty treats each
		// transmission as a brand new image and stacks them instead of
		// replacing — the "printing a new bar every tick" bug. The delete
		// command clears that placement first so a resized re-render (a
		// different cols/rows for the same id) doesn't leave a stale
		// larger/smaller ghost behind it.
		if err := kittyDeletePlacement(out); err != nil {
			return fmt.Errorf("artwork: kitty clear placement: %w", err)
		}
		opts := rasterm.KittyImgOpts{
			DstCols:     uint32(cols),
			DstRows:     uint32(rows),
			ImageId:     kittyImageID,
			PlacementId: kittyPlacementID,
		}
		if err := rasterm.KittyWriteImage(out, img, opts); err != nil {
			return fmt.Errorf("artwork: kitty render: %w", err)
		}
		return nil
	case TermITerm2:
		opts := rasterm.ItermImgOpts{
			Width:             fmt.Sprintf("%d", cols),
			Height:            fmt.Sprintf("%d", rows),
			IgnoreAspectRatio: true, // rows is already derived from the image's own aspect ratio
			DisplayInline:     true,
		}
		if err := rasterm.ItermWriteImageWithOptions(out, img, opts); err != nil {
			return fmt.Errorf("artwork: iterm2 render: %w", err)
		}
		return nil
	case TermSixel:
		scaled := resize.Thumbnail(uint(cols)*sixelPxPerCol, uint(rows)*sixelPxPerRow, img, resize.Lanczos3)
		pImg, ok := scaled.(*image.Paletted)
		if !ok {
			pImg = toPaletted(scaled)
		}
		if err := rasterm.SixelWriteImage(out, pImg); err != nil {
			return fmt.Errorf("artwork: sixel render: %w", err)
		}
		return nil
	default:
		return renderASCII(out, img, cols, rows)
	}
}

// defaultCols is used when a caller doesn't have a layout-derived width yet
// (e.g. before the first WindowSizeMsg).
const defaultCols = 30

// kittyImageID/kittyPlacementID are fixed, sonora-cli-specific Kitty
// graphics protocol identifiers. sonora-cli only ever shows one piece of
// cover art on screen at a time, so a single hardcoded pair (rather than
// generating fresh ones per track) is enough to keep every re-render
// targeting the same placement.
const (
	kittyImageID     = 0x736f6e72 // "sonr" packed into 4 bytes, arbitrary but stable
	kittyPlacementID = 1
)

// kittyDeletePlacement clears kittyPlacementID's prior contents (Kitty
// graphics protocol "delete" action, d=i deletes by image id) before a new
// image is transmitted to it, so a size change on the same id doesn't
// leave the old placement's pixels showing around the new one.
func kittyDeletePlacement(out io.Writer) error {
	_, err := fmt.Fprintf(out, "%sa=d,d=i,i=%d,p=%d%s",
		rasterm.KITTY_IMG_HDR, kittyImageID, kittyPlacementID, rasterm.KITTY_IMG_FTR)
	return err
}

// sixelPxPerCol/sixelPxPerRow are a rough cell-to-pixel conversion for
// Sixel, which has no notion of "display in N columns" the way Kitty/iTerm2
// do — the image is rasterized at a pixel size and the terminal maps that
// to however many cells it covers.
const (
	sixelPxPerCol = 10
	sixelPxPerRow = 20
)

// targetRows derives a row count from cols using the image's own aspect
// ratio and cellAspect, so cover art (usually square) doesn't get squashed
// or stretched.
func targetRows(img image.Image, cols int) int {
	b := img.Bounds()
	if b.Dx() == 0 {
		return cols
	}
	imgAspect := float64(b.Dy()) / float64(b.Dx())
	rows := int(float64(cols) * imgAspect / cellAspect)
	if rows < 1 {
		rows = 1
	}
	return rows
}

func renderASCII(out io.Writer, img image.Image, cols, rows int) error {
	converter := convert.NewImageConverter()
	opts := convert.DefaultOptions
	opts.Colored = true
	opts.FitScreen = false
	opts.FixedWidth = cols
	opts.FixedHeight = rows
	s := converter.Image2ASCIIString(img, &opts)
	_, err := io.WriteString(out, s)
	if err != nil {
		return fmt.Errorf("artwork: ascii render: %w", err)
	}
	return nil
}

// toPaletted converts an arbitrary image.Image to *image.Paletted for
// Sixel encoding, which requires a palette. Floyd-Steinberg dithering
// against the stdlib's Plan9 256-color palette keeps cover art
// recognizable without a dedicated quantization dependency.
func toPaletted(img image.Image) *image.Paletted {
	pImg := image.NewPaletted(img.Bounds(), palette.Plan9)
	draw.FloydSteinberg.Draw(pImg, img.Bounds(), img, image.Point{})
	return pImg
}
