package artwork

import (
	"bytes"
	"image"
)

// RenderString is Render, but returns the encoded output as a string
// instead of writing to an io.Writer — the shape internal/ui needs to
// embed cover art inside a Bubble Tea View() return value. Native
// protocols (Kitty, iTerm2, Sixel) work here because their encoders emit
// terminal escape sequences as plain bytes, which pass through untouched
// as part of the rendered string.
func RenderString(img image.Image, term TermType, mode Mode) (string, error) {
	var buf bytes.Buffer
	if err := Render(&buf, img, term, mode); err != nil {
		return "", err
	}
	return buf.String(), nil
}
