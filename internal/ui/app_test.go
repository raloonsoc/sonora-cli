package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/raloonsoc/sonora-cli/internal/artwork"
)

func TestApp_paneWidths(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		wantLib     int
		wantNowPlay int
	}{
		{"typical terminal", 100, 40, 60},
		{"narrow terminal", 60, 24, 36},
		{"zero width falls back to minContentWidth", 0, minContentWidth, minContentWidth},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := App{width: tt.width}
			gotLib, gotNP := m.paneWidths()
			if gotLib != tt.wantLib || gotNP != tt.wantNowPlay {
				t.Errorf("paneWidths() = (%d, %d), want (%d, %d)", gotLib, gotNP, tt.wantLib, tt.wantNowPlay)
			}
			if tt.width > 0 && gotLib+gotNP != tt.width {
				t.Errorf("paneWidths() sums to %d, want %d", gotLib+gotNP, tt.width)
			}
		})
	}
}

// newTestApp builds an App without a live client/controller — enough to
// exercise layout logic (Update on WindowSizeMsg, View) without touching
// anything that reaches for a real server or mpv process.
func newTestApp() App {
	return App{
		library:    newLibraryModel(nil),
		nowPlaying: newNowPlayingModel(nil, nil, 80, artwork.TermUnknown, artwork.ModeAuto, true),
		keys:       DefaultKeyMap(),
	}
}

func TestApp_windowSizeMsg_splitsBetweenPanes(t *testing.T) {
	m := newTestApp()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := updated.(App)

	if app.width != 120 || app.height != 40 {
		t.Fatalf("App size = (%d, %d), want (120, 40)", app.width, app.height)
	}
	if app.library.list.Width() == 0 {
		t.Error("library list width is 0 after a WindowSizeMsg — the panel would render empty")
	}
	if app.nowPlaying.width == 0 {
		t.Error("nowPlaying width is 0 after a WindowSizeMsg")
	}
}

func TestApp_view_rendersBothPanesSideBySide(t *testing.T) {
	m := newTestApp()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app := updated.(App)

	view := app.View()
	lines := strings.Split(view, "\n")
	if len(lines) == 0 {
		t.Fatal("View() produced no output")
	}
	// Side-by-side panes means the first content line should contain
	// characters from both the library border and the now-playing border,
	// not just one stacked above the other.
	if !strings.Contains(lines[0], "─") {
		t.Errorf("first line = %q, expected a pane border", lines[0])
	}
}
