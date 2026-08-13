package artwork

import (
	"fmt"
	"image"
	"io"

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
// cols/2 rows, or ASCII art comes out stretched.
const cellAspect = 2.0

// Render encodes img as colored ASCII into out, scaled to fit within cols
// terminal columns. ModeOff callers should skip calling Render entirely; it
// is not handled here since there's nothing to render.
//
// Native graphics protocols (Kitty, iTerm2, Sixel) were tried here and
// pulled after they proved unreliable in exactly the setting sonora-cli
// needs them for: a TUI that repaints its whole frame on every ~500ms
// position tick. Those protocols place images as state that persists
// outside the normal text grid — separate from whatever Bubble Tea's
// diffing renderer believes is on screen — and repainting on that cadence
// desynced the two, observed as a fresh, undeleted image placement
// appearing on every tick instead of the existing one updating in place.
// ASCII has no such out-of-band state: it's plain text, so it repaints
// exactly as reliably as every other part of the UI. term is accepted for
// API stability (a future terminal-aware ASCII variant might want it) but
// is currently unused.
func Render(out io.Writer, img image.Image, term TermType, mode Mode, cols int) error {
	_ = term
	_ = mode // ASCII is the only implementation now; kept for call-site stability
	if cols <= 0 {
		cols = defaultCols
	}
	rows := targetRows(img, cols)
	return renderASCII(out, img, cols, rows)
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
