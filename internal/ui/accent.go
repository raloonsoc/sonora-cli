package ui

import (
	"context"
	"math"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	colorful "github.com/lucasb-eyer/go-colorful"

	"github.com/raloonsoc/sonora-cli/internal/subsonic"
)

// defaultAccent is the neutral fallback applied against any non-Sonora
// OpenSubsonic server, or when a color fails the contrast check (SPECS §7).
var defaultAccent = lipgloss.Color("245") // mid-gray, readable on light and dark backgrounds

// minContrastRatio is the WCAG AA threshold for large/UI text (3:1). A
// server-supplied accent color is only used if it clears this bar against
// both a black and a white background — sonora-cli doesn't know the user's
// actual terminal background, so it must be legible on either.
const minContrastRatio = 3.0

// accentState holds the resolved accent color for the current track, kept
// separate from playback state so it's only recomputed on track change,
// like artworkState and lyricsState.
type accentState struct {
	albumID string
	color   lipgloss.Color
}

// accentLoadedMsg carries the resolved accent for albumID. Always sent —
// even on failure or a non-Sonora server — so the caller has a definite
// color to apply rather than an unset zero value.
type accentLoadedMsg struct {
	albumID string
	color   lipgloss.Color
}

// loadAccent fetches the album's Sonora-specific accent_color field
// (SPECS §7) and resolves it to a lipgloss.Color, falling back to
// defaultAccent when the field is absent (any non-Sonora server — treated
// as normal, not an error) or fails the contrast check. A no-op if albumID
// already matches what's resolved.
func loadAccent(client *subsonic.Client, current accentState, albumID string) tea.Cmd {
	if albumID == "" || albumID == current.albumID {
		return nil
	}
	return func() tea.Msg {
		album, err := client.GetAlbum(context.Background(), albumID)
		if err != nil || album.AccentColor == "" {
			return accentLoadedMsg{albumID: albumID, color: defaultAccent}
		}
		return accentLoadedMsg{albumID: albumID, color: resolveAccent(album.AccentColor)}
	}
}

// resolveAccent validates hex against minContrastRatio on both black and
// white backgrounds, falling back to defaultAccent if it fails either —
// an accent color that vanishes against the user's actual terminal
// background is worse than no accent at all.
func resolveAccent(hex string) lipgloss.Color {
	c, err := colorful.Hex(hex)
	if err != nil {
		return defaultAccent
	}

	lum := relativeLuminance(c)
	if contrastRatio(lum, 0.0) < minContrastRatio || contrastRatio(lum, 1.0) < minContrastRatio {
		return defaultAccent
	}
	return lipgloss.Color(hex)
}

// relativeLuminance computes WCAG relative luminance from sRGB components.
func relativeLuminance(c colorful.Color) float64 {
	lin := func(v float64) float64 {
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// contrastRatio implements the WCAG contrast formula between two relative
// luminance values.
func contrastRatio(l1, l2 float64) float64 {
	lighter, darker := l1, l2
	if darker > lighter {
		lighter, darker = darker, lighter
	}
	return (lighter + 0.05) / (darker + 0.05)
}
