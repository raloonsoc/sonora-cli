// Package ui implements the Bubble Tea models that make up sonora-cli's
// terminal interface, per SPECS §9/§10.3.
package ui

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raloonsoc/sonora-cli/internal/player"
	"github.com/raloonsoc/sonora-cli/internal/subsonic"
)

// pane is which half of the split view has focus; tab cycles between them.
type pane int

const (
	paneLibrary pane = iota
	paneNowPlaying
)

// App is the root Bubble Tea model: view switching, window-resize handling,
// and global quit. Dependencies are constructed by the caller and passed
// in, never reached for globally (CODESTYLE §5).
type App struct {
	client *subsonic.Client
	ctrl   *player.Controller

	library    libraryModel
	nowPlaying nowPlayingModel
	help       help.Model
	keys       KeyMap

	focus     pane
	showHelp  bool
	width     int
	height    int
	posCh     <-chan player.PositionUpdate
	cancelPos context.CancelFunc
}

// New builds the root model. ctrl must already be running (see
// player.New); App does not own its lifecycle beyond the session.
//
// The position-poll channel is opened here, not in Init, because Init
// returns only a tea.Cmd (no updated model) under Bubble Tea's
// value-receiver convention — the channel and its cancel func need to live
// on the value New returns.
func New(client *subsonic.Client, ctrl *player.Controller, initialVolume int) App {
	ctx, cancel := context.WithCancel(context.Background())
	return App{
		client:     client,
		ctrl:       ctrl,
		library:    newLibraryModel(client),
		nowPlaying: newNowPlayingModel(client, ctrl, initialVolume),
		help:       help.New(),
		keys:       DefaultKeyMap(),
		focus:      paneLibrary,
		posCh:      ctrl.PositionStream(ctx),
		cancelPos:  cancel,
	}
}

func (m App) Init() tea.Cmd {
	return tea.Batch(
		m.library.Init(),
		waitForPosition(m.posCh),
	)
}

// waitForPosition receives exactly one update from ch and wraps it as a
// tea.Msg. The App re-issues this Cmd after each tick to keep listening on
// the same long-lived channel, rather than starting a new PositionStream
// per tick — one poller for the session, not one per message.
func waitForPosition(ch <-chan player.PositionUpdate) tea.Cmd {
	return func() tea.Msg {
		update, ok := <-ch
		if !ok {
			return nil
		}
		return positionTickMsg(update)
	}
}

func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.cancelPos != nil {
				m.cancelPos()
			}
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, m.keys.Tab):
			m.focus = togglePane(m.focus)
			return m, nil
		}
	}

	var cmds []tea.Cmd

	// Each tick delivers exactly one update from the channel opened in New;
	// re-issue waitForPosition on the same channel to keep polling, rather
	// than opening a new PositionStream per tick.
	if _, ok := msg.(positionTickMsg); ok {
		var cmd tea.Cmd
		m.nowPlaying, cmd = m.nowPlaying.Update(msg)
		cmds = append(cmds, cmd, waitForPosition(m.posCh))
		return m, tea.Batch(cmds...)
	}

	// songSelectedMsg and transport keys route to nowPlaying regardless of
	// focus, so playback controls work while browsing the library.
	if m.focus == paneLibrary {
		var cmd tea.Cmd
		m.library, cmd = m.library.Update(msg)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	m.nowPlaying, cmd = m.nowPlaying.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func togglePane(p pane) pane {
	if p == paneLibrary {
		return paneNowPlaying
	}
	return paneLibrary
}

func (m App) View() string {
	view := m.library.View() + "\n\n" + m.nowPlaying.View()
	if m.showHelp {
		view += "\n\n" + m.help.View(m.keys)
	}
	return view
}
