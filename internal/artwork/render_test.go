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
	err := Render(&buf, solidImage(color.RGBA{R: 200, G: 50, B: 50, A: 255}), TermUnknown, ModeAuto, 20)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty ASCII output")
	}
}

func TestRender_forcedASCIIIgnoresTermType(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, solidImage(color.White), TermKitty, ModeASCII, 20)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty ASCII output even with a Kitty-capable term")
	}
}

func TestRender_zeroColsUsesDefault(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, solidImage(color.White), TermUnknown, ModeAuto, 0)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty output when cols<=0 falls back to defaultCols")
	}
}

func TestRenderString_matchesRender(t *testing.T) {
	img := solidImage(color.RGBA{G: 255, A: 255})

	var buf bytes.Buffer
	if err := Render(&buf, img, TermUnknown, ModeAuto, 20); err != nil {
		t.Fatalf("Render: %v", err)
	}

	s, err := RenderString(img, TermUnknown, ModeAuto, 20)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}

	if s != buf.String() {
		t.Error("RenderString output diverged from Render output")
	}
}

func TestTargetRows_preservesAspectRatio(t *testing.T) {
	square := image.NewRGBA(image.Rect(0, 0, 100, 100))
	got := targetRows(square, 20)
	want := 10 // 20 cols * (100/100 aspect) / cellAspect(2.0)
	if got != want {
		t.Errorf("targetRows(square, 20) = %d, want %d", got, want)
	}
}

func TestTargetRows_zeroWidthImageFallsBackToCols(t *testing.T) {
	empty := image.NewRGBA(image.Rect(0, 0, 0, 0))
	if got := targetRows(empty, 20); got != 20 {
		t.Errorf("targetRows(empty, 20) = %d, want 20", got)
	}
}

func TestToPaletted_preservesBounds(t *testing.T) {
	img := solidImage(color.RGBA{B: 255, A: 255})
	p := toPaletted(img)
	if p.Bounds() != img.Bounds() {
		t.Errorf("toPaletted bounds = %v, want %v", p.Bounds(), img.Bounds())
	}
}
