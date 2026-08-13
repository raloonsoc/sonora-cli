package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/raloonsoc/sonora-cli/internal/artwork"
)

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

func TestApp_windowSizeMsg_splitsHeightAboveBar(t *testing.T) {
	m := newTestApp()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := updated.(App)

	if app.width != 120 || app.height != 40 {
		t.Fatalf("App size = (%d, %d), want (120, 40)", app.width, app.height)
	}
	if app.library.list.Width() == 0 {
		t.Error("library list width is 0 after a WindowSizeMsg — the pane would render empty")
	}
	_, frameH := paneStyle.GetFrameSize()
	wantLibH := 40 - barHeight - frameH
	if app.library.list.Height() != wantLibH {
		t.Errorf("library list height = %d, want %d (total - barHeight - pane frame)", app.library.list.Height(), wantLibH)
	}
	if app.nowPlaying.width != 120 {
		t.Errorf("nowPlaying width = %d, want 120 (full width, not split)", app.nowPlaying.width)
	}
}

func TestApp_windowSizeMsg_shortTerminalClampsLibraryHeight(t *testing.T) {
	m := newTestApp()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 2})
	app := updated.(App)

	if app.library.list.Height() < 0 {
		t.Errorf("library list height = %d, want >= 0 even when terminal is shorter than barHeight", app.library.list.Height())
	}
}

func TestApp_view_stacksLibraryAboveBar(t *testing.T) {
	m := newTestApp()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app := updated.(App)

	view := app.View()
	if view == "" {
		t.Fatal("View() produced no output")
	}
	lines := strings.Split(view, "\n")
	// A vertical stack of two bordered panes is taller than either pane
	// alone; a regression back to a single unbounded panel would produce
	// far fewer lines than the terminal height requested.
	if len(lines) < barHeight {
		t.Errorf("View() produced %d lines, want at least barHeight (%d)", len(lines), barHeight)
	}
}

func TestApp_toggleFullscreen_hidesLibraryPane(t *testing.T) {
	m := newTestApp()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app := updated.(App)

	app.nowPlaying.fullscreen = true

	view := app.View()
	if strings.Contains(view, "Browse") {
		t.Error("View() in fullscreen mode should not render the library pane's title")
	}
}
