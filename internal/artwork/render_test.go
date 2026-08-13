package artwork

import (
	"bytes"
	"image"
	"image/color"
	"strings"
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

func TestRender_halfBlockOutput(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, solidImage(color.RGBA{R: 200, G: 50, B: 50, A: 255}), TermUnknown, ModeAuto, 20)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(out, halfBlock) {
		t.Errorf("output missing half-block character %q", halfBlock)
	}
	if !strings.Contains(out, "\x1b[38;2;") {
		t.Error("output missing a 24-bit truecolor foreground escape (\\x1b[38;2;...)")
	}
	if !strings.Contains(out, "\x1b[48;2;") {
		t.Error("output missing a 24-bit truecolor background escape (\\x1b[48;2;...)")
	}
}

func TestRender_lineCountMatchesRows(t *testing.T) {
	var buf bytes.Buffer
	img := solidImage(color.White)
	if err := Render(&buf, img, TermUnknown, ModeAuto, 20); err != nil {
		t.Fatalf("Render: %v", err)
	}
	wantRows := targetRows(img, 20)
	gotLines := strings.Count(buf.String(), "\n") + 1
	if gotLines != wantRows {
		t.Errorf("output has %d lines, want %d (targetRows)", gotLines, wantRows)
	}
}

// TestRender_alwaysBlocksRegardlessOfTermType locks in that native graphics
// protocols are gone: every TermType must produce identical, plain-text
// half-block output. Native protocols (Kitty in particular) were removed
// after proving unreliable in a TUI that repaints its whole frame on every
// ~500ms position tick — see the comment on Render.
func TestRender_alwaysBlocksRegardlessOfTermType(t *testing.T) {
	img := solidImage(color.RGBA{B: 200, A: 255})

	var kittyOut, iterm2Out, sixelOut, unknownOut bytes.Buffer
	if err := Render(&kittyOut, img, TermKitty, ModeAuto, 20); err != nil {
		t.Fatalf("Render (Kitty): %v", err)
	}
	if err := Render(&iterm2Out, img, TermITerm2, ModeAuto, 20); err != nil {
		t.Fatalf("Render (iTerm2): %v", err)
	}
	if err := Render(&sixelOut, img, TermSixel, ModeAuto, 20); err != nil {
		t.Fatalf("Render (Sixel): %v", err)
	}
	if err := Render(&unknownOut, img, TermUnknown, ModeAuto, 20); err != nil {
		t.Fatalf("Render (Unknown): %v", err)
	}

	if kittyOut.String() != unknownOut.String() {
		t.Error("TermKitty produced different output than TermUnknown — a native protocol path is still active")
	}
	if iterm2Out.String() != unknownOut.String() {
		t.Error("TermITerm2 produced different output than TermUnknown — a native protocol path is still active")
	}
	if sixelOut.String() != unknownOut.String() {
		t.Error("TermSixel produced different output than TermUnknown — a native protocol path is still active")
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

func TestSampleRGB_clampsYToBounds(t *testing.T) {
	img := solidImage(color.RGBA{R: 10, G: 20, B: 30, A: 255})
	r, g, b := sampleRGB(img, 0, 999) // past the image's bottom edge
	if r != 10 || g != 20 || b != 30 {
		t.Errorf("sampleRGB out-of-bounds = (%d,%d,%d), want (10,20,30) from the clamped row", r, g, b)
	}
}
