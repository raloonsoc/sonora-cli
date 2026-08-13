package artwork

import (
	"bytes"
	"image"
)

// RenderString is Render, but returns the encoded output as a string
// instead of writing to an io.Writer — the shape internal/ui needs to
// embed cover art inside a Bubble Tea View() return value.
func RenderString(img image.Image, term TermType, mode Mode, cols int) (string, error) {
	var buf bytes.Buffer
	if err := Render(&buf, img, term, mode, cols); err != nil {
		return "", err
	}
	return buf.String(), nil
}
