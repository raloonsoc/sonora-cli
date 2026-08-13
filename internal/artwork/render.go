package artwork

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"io"

	"github.com/BourgeoisBear/rasterm"
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

// Render encodes img for the terminal into out, dispatching on term (or
// forcing ASCII when mode is ModeASCII). ModeOff callers should skip
// calling Render entirely; it is not handled here since there's nothing to
// render.
func Render(out io.Writer, img image.Image, term TermType, mode Mode) error {
	if mode == ModeASCII {
		return renderASCII(out, img)
	}

	switch term {
	case TermKitty:
		if err := rasterm.KittyWriteImage(out, img, rasterm.KittyImgOpts{}); err != nil {
			return fmt.Errorf("artwork: kitty render: %w", err)
		}
		return nil
	case TermITerm2:
		if err := rasterm.ItermWriteImage(out, img); err != nil {
			return fmt.Errorf("artwork: iterm2 render: %w", err)
		}
		return nil
	case TermSixel:
		pImg, ok := img.(*image.Paletted)
		if !ok {
			pImg = toPaletted(img)
		}
		if err := rasterm.SixelWriteImage(out, pImg); err != nil {
			return fmt.Errorf("artwork: sixel render: %w", err)
		}
		return nil
	default:
		return renderASCII(out, img)
	}
}

// asciiWidth/asciiHeight bound the ASCII fallback's cell footprint. Fixed
// rather than auto-detected: image2ascii's FitScreen/StretchedScreen modes
// (on by default in convert.DefaultOptions) call out to terminal size
// detection and log.Fatal — killing the whole process — if that probe
// fails, which is not a risk worth taking for a cosmetic fallback render.
const (
	asciiWidth  = 40
	asciiHeight = 20
)

func renderASCII(out io.Writer, img image.Image) error {
	converter := convert.NewImageConverter()
	opts := convert.DefaultOptions
	opts.Colored = true
	opts.FitScreen = false
	opts.FixedWidth = asciiWidth
	opts.FixedHeight = asciiHeight
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
