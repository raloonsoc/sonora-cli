// Package ui implements the Bubble Tea models that make up sonora-cli's
// terminal interface, per SPECS §9/§10.3.
package ui

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raloonsoc/sonora-cli/internal/artwork"
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

	library       libraryModel
	nowPlaying    nowPlayingModel
	search        searchModel
	profileSwitch profileSwitchModel
	help          help.Model
	keys          KeyMap

	focus     pane
	showHelp  bool
	width     int
	height    int
	posCh     <-chan player.PositionUpdate
	cancelPos context.CancelFunc

	// SwitchProfile is set once App.Update quits in response to a
	// profileSwitchModel selection; main reads it after tea.Program.Run
	// returns to decide whether to relaunch with a different profile.
	SwitchProfile string
}

// Options bundles the config-derived display preferences New needs, kept
// as a group so adding a future preference doesn't grow New's parameter
// list further.
type Options struct {
	InitialVolume  int
	Term           artwork.TermType // detected once at startup, cached for the session (SPECS §6.1)
	ArtMode        artwork.Mode     // "auto" | "ascii" | "off"
	LyricsEnabled  bool
	ProfileNames   []string // all configured profiles, for the switch-profile view
	CurrentProfile string
}

// New builds the root model. ctrl must already be running (see
// player.New); App does not own its lifecycle beyond the session.
//
// The position-poll channel is opened here, not in Init, because Init
// returns only a tea.Cmd (no updated model) under Bubble Tea's
// value-receiver convention — the channel and its cancel func need to live
// on the value New returns.
func New(client *subsonic.Client, ctrl *player.Controller, opts Options) App {
	ctx, cancel := context.WithCancel(context.Background())
	return App{
		client:        client,
		ctrl:          ctrl,
		library:       newLibraryModel(client),
		nowPlaying:    newNowPlayingModel(client, ctrl, opts.InitialVolume, opts.Term, opts.ArtMode, opts.LyricsEnabled),
		search:        newSearchModel(client),
		profileSwitch: newProfileSwitchModel(opts.ProfileNames, opts.CurrentProfile),
		help:          help.New(),
		keys:          DefaultKeyMap(),
		focus:         paneLibrary,
		posCh:         ctrl.PositionStream(ctx),
		cancelPos:     cancel,
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

		// Split the terminal into a library column and a now-playing
		// column instead of forwarding the full-width message to both —
		// each submodel used to size itself to the whole terminal and
		// then get stacked vertically, which is what made the library
		// list collapse and the cover art render oversized.
		libW, npW := m.paneWidths()
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.library, cmd = m.library.Update(tea.WindowSizeMsg{Width: libW, Height: msg.Height})
		cmds = append(cmds, cmd)
		m.nowPlaying, cmd = m.nowPlaying.Update(tea.WindowSizeMsg{Width: npW, Height: msg.Height})
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// While search is active, every key goes to its text input first —
		// including letters that double as global bindings (q, ?, tab) —
		// so typing "quit" into the search box doesn't quit the app.
		if m.search.active {
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			return m, cmd
		}

		if m.profileSwitch.active {
			var cmd tea.Cmd
			m.profileSwitch, cmd = m.profileSwitch.Update(msg)
			m.SwitchProfile = m.profileSwitch.SwitchTo
			return m, cmd
		}

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
		case key.Matches(msg, m.keys.Search):
			m.search = m.search.Open()
			return m, nil
		case key.Matches(msg, m.keys.SwitchProfile):
			m.profileSwitch = m.profileSwitch.Open()
			return m, nil
		}

	case searchTickMsg, searchResultMsg:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
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

// libraryWidthFraction is the portion of the terminal given to the library
// column; the rest goes to now-playing. Matches the README's mockup, which
// puts a narrower browse list next to a wider playback view.
const libraryWidthFraction = 0.4

// paneWidths splits m.width between the library and now-playing columns.
func (m App) paneWidths() (libW, npW int) {
	if m.width <= 0 {
		return minContentWidth, minContentWidth
	}
	libW = int(float64(m.width) * libraryWidthFraction)
	npW = m.width - libW
	return libW, npW
}

func (m App) View() string {
	if m.search.active {
		return m.search.View()
	}
	if m.profileSwitch.active {
		return m.profileSwitch.View()
	}

	libraryPane := m.library.View(m.focus == paneLibrary)
	nowPlayingPane := m.nowPlaying.View(m.focus == paneNowPlaying)

	view := lipgloss.JoinHorizontal(lipgloss.Top, libraryPane, nowPlayingPane)
	if m.showHelp {
		view = lipgloss.JoinVertical(lipgloss.Left, view, m.help.View(m.keys))
	}
	return view
}
