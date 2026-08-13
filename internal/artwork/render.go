package artwork

import (
	"fmt"
	"image"
	"io"
	"strings"

	"github.com/nfnt/resize"
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
// cols/2 terminal rows of block characters.
const cellAspect = 2.0

// halfBlock is U+2580 UPPER HALF BLOCK: with a foreground color it paints
// the cell's top half, and with a background color the bottom half — one
// character carries two vertically-stacked pixel samples, doubling the
// effective vertical resolution over one block-per-cell.
const halfBlock = "▀"

// Render encodes img as colored half-block art into out, scaled to fit
// within cols terminal columns. ModeOff callers should skip calling Render
// entirely; it is not handled here since there's nothing to render.
//
// Native graphics protocols (Kitty, iTerm2, Sixel) were tried here and
// pulled after they proved unreliable in exactly the setting sonora-cli
// needs them for: a TUI that repaints its whole frame on every ~500ms
// position tick. Those protocols place images as state that persists
// outside the normal text grid — separate from whatever Bubble Tea's
// diffing renderer believes is on screen — and repainting on that cadence
// desynced the two, observed as a fresh, undeleted image placement
// appearing on every tick instead of the existing one updating in place.
// Block characters have no such out-of-band state: it's plain text with
// per-cell truecolor escapes, so it repaints exactly as reliably as every
// other part of the UI.
//
// A density-mapped ASCII glyph set (image2ascii, tried first) technically
// carried the same truecolor per character, but glyphs chosen by
// brightness read as visual noise next to a low-contrast source image —
// solid half-blocks read as color regardless of the source image's
// contrast, closer to how sixel/kitty actually look. term is accepted for
// API stability (a future terminal-aware variant might want it) but is
// currently unused.
func Render(out io.Writer, img image.Image, term TermType, mode Mode, cols int) error {
	_ = term
	_ = mode // block rendering is the only implementation now; kept for call-site stability
	if cols <= 0 {
		cols = defaultCols
	}
	rows := targetRows(img, cols)
	return renderHalfBlocks(out, img, cols, rows)
}

// defaultCols is used when a caller doesn't have a layout-derived width yet
// (e.g. before the first WindowSizeMsg).
const defaultCols = 30

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

// renderHalfBlocks resamples img to cols x (rows*2) pixels — two vertical
// samples per terminal row — and writes one halfBlock character per cell,
// its foreground the top pixel's color and its background the bottom
// pixel's, both as 24-bit truecolor escapes.
func renderHalfBlocks(out io.Writer, img image.Image, cols, rows int) error {
	scaled := resize.Resize(uint(cols), uint(rows*2), img, resize.Lanczos3)
	b := scaled.Bounds()

	var sb strings.Builder
	for row := 0; row < rows; row++ {
		topY := b.Min.Y + row*2
		botY := topY + 1
		for x := b.Min.X; x < b.Max.X; x++ {
			tr, tg, tb := sampleRGB(scaled, x, topY)
			br, bg, bb := sampleRGB(scaled, x, botY)
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm%s", tr, tg, tb, br, bg, bb, halfBlock)
		}
		sb.WriteString("\x1b[0m")
		if row < rows-1 {
			sb.WriteByte('\n')
		}
	}

	_, err := io.WriteString(out, sb.String())
	if err != nil {
		return fmt.Errorf("artwork: block render: %w", err)
	}
	return nil
}

// sampleRGB reads img at (x, y), clamped to its bounds — the bottom sample
// row can land one past the image's own last row when rows*2 is odd.
func sampleRGB(img image.Image, x, y int) (r, g, b uint8) {
	bounds := img.Bounds()
	if y >= bounds.Max.Y {
		y = bounds.Max.Y - 1
	}
	rr, gg, bb, _ := img.At(x, y).RGBA()
	// image.Color.RGBA() returns 16-bit-per-channel values; downshift to 8.
	return uint8(rr >> 8), uint8(gg >> 8), uint8(bb >> 8)
}
