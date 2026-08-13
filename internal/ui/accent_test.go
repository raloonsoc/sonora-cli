package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestResolveAccent(t *testing.T) {
	tests := []struct {
		name string
		hex  string
		want string // "" means expect defaultAccent
	}{
		{"vibrant color with good contrast both ways", "#c96a3f", "#c96a3f"},
		{"near-white fails contrast against white background", "#fefefe", ""},
		{"near-black fails contrast against black background", "#010101", ""},
		{"invalid hex falls back to default", "not-a-color", ""},
		{"empty string falls back to default", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveAccent(tt.hex)
			want := lipgloss.Color(tt.want)
			if tt.want == "" {
				want = defaultAccent
			}
			if got != want {
				t.Errorf("resolveAccent(%q) = %q, want %q", tt.hex, got, want)
			}
		})
	}
}

func TestContrastRatio(t *testing.T) {
	// Black vs white is the maximum possible ratio, 21:1.
	got := contrastRatio(0.0, 1.0)
	if got < 20.9 || got > 21.1 {
		t.Errorf("contrastRatio(black, white) = %.2f, want ~21", got)
	}

	// Identical luminance has no contrast: ratio 1.
	got = contrastRatio(0.5, 0.5)
	if got < 0.99 || got > 1.01 {
		t.Errorf("contrastRatio(equal) = %.2f, want 1", got)
	}
}

func TestLoadAccent_noOpWhenAlbumUnchanged(t *testing.T) {
	current := accentState{albumID: "42", color: defaultAccent}
	cmd := loadAccent(nil, current, "42")
	if cmd != nil {
		t.Error("expected nil Cmd when albumID matches current state")
	}
}

func TestLoadAccent_noOpWhenAlbumIDEmpty(t *testing.T) {
	cmd := loadAccent(nil, accentState{}, "")
	if cmd != nil {
		t.Error("expected nil Cmd for an empty albumID")
	}
}
