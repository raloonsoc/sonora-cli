package artwork

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func solidImage(c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestRender_asciiFallback(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, solidImage(color.RGBA{R: 200, G: 50, B: 50, A: 255}), TermUnknown, ModeAuto)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty ASCII output")
	}
}

func TestRender_forcedASCIIIgnoresTermType(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, solidImage(color.White), TermKitty, ModeASCII)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty ASCII output even with a Kitty-capable term")
	}
}

func TestRenderString_matchesRender(t *testing.T) {
	img := solidImage(color.RGBA{G: 255, A: 255})

	var buf bytes.Buffer
	if err := Render(&buf, img, TermUnknown, ModeAuto); err != nil {
		t.Fatalf("Render: %v", err)
	}

	s, err := RenderString(img, TermUnknown, ModeAuto)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}

	if s != buf.String() {
		t.Error("RenderString output diverged from Render output")
	}
}

func TestToPaletted_preservesBounds(t *testing.T) {
	img := solidImage(color.RGBA{B: 255, A: 255})
	p := toPaletted(img)
	if p.Bounds() != img.Bounds() {
		t.Errorf("toPaletted bounds = %v, want %v", p.Bounds(), img.Bounds())
	}
}
