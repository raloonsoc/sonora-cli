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

// barHeight is the fixed height of the bottom now-playing bar, including
// its border — the library pane gets whatever's left above it.
const barHeight = 5

// App is the root Bubble Tea model: view switching, window-resize handling,
// and global quit. Dependencies are constructed by the caller and passed
// in, never reached for globally (CODESTYLE §5).
//
// Layout is a full-width library pane with a fixed-height now-playing bar
// docked at the bottom (desktop-Spotify style), or — while nowPlaying is in
// "Spotify mode" (see nowPlayingModel.fullscreen, toggled by the L key) —
// nowPlaying alone fills the screen. The library list always owns keyboard
// focus otherwise; transport controls (space, n/p, seek, volume) work
// globally regardless, so there's no pane-to-pane focus to cycle through.
type App struct {
	client *subsonic.Client
	ctrl   *player.Controller

	library       libraryModel
	nowPlaying    nowPlayingModel
	search        searchModel
	profileSwitch profileSwitchModel
	help          help.Model
	keys          KeyMap

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

		// The library pane gets full width and whatever height remains
		// above the fixed-height bottom bar; nowPlaying (both its bar and
		// fullscreen views) gets full width and either barHeight or the
		// full height, depending on which view is active.
		libH := m.height - barHeight
		if libH < 0 {
			libH = 0
		}
		npH := barHeight
		if m.nowPlaying.IsFullscreen() {
			npH = m.height
		}

		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.library, cmd = m.library.Update(tea.WindowSizeMsg{Width: m.width, Height: libH})
		cmds = append(cmds, cmd)
		m.nowPlaying, cmd = m.nowPlaying.Update(tea.WindowSizeMsg{Width: m.width, Height: npH})
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

		// In fullscreen lyrics mode, only nowPlaying's own keys (play/pause,
		// seek, volume, L/esc to leave) apply — library navigation keys
		// would silently do nothing useful while its pane isn't visible.
		if m.nowPlaying.IsFullscreen() {
			var cmd tea.Cmd
			m.nowPlaying, cmd = m.nowPlaying.Update(msg)
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
		wasFullscreen := m.nowPlaying.IsFullscreen()
		m.nowPlaying, cmd = m.nowPlaying.Update(msg)
		cmds = append(cmds, cmd, waitForPosition(m.posCh))
		if m.nowPlaying.IsFullscreen() != wasFullscreen {
			cmds = append(cmds, resizeCmd(m.width, m.height))
		}
		return m, tea.Batch(cmds...)
	}

	// songSelectedMsg and transport keys route to both: the library tree
	// needs it for navigation/selection, nowPlaying needs it to react to a
	// newly selected track or a transport key.
	var cmd tea.Cmd
	m.library, cmd = m.library.Update(msg)
	cmds = append(cmds, cmd)

	wasFullscreen := m.nowPlaying.IsFullscreen()
	m.nowPlaying, cmd = m.nowPlaying.Update(msg)
	cmds = append(cmds, cmd)
	if m.nowPlaying.IsFullscreen() != wasFullscreen {
		cmds = append(cmds, resizeCmd(m.width, m.height))
	}

	return m, tea.Batch(cmds...)
}

// resizeCmd re-delivers the current terminal size as a synthetic
// WindowSizeMsg, so switching in or out of fullscreen mode immediately
// re-splits pane heights instead of waiting for the user to actually
// resize the terminal.
func resizeCmd(width, height int) tea.Cmd {
	return func() tea.Msg {
		return tea.WindowSizeMsg{Width: width, Height: height}
	}
}

func (m App) View() string {
	if m.search.active {
		return m.search.View()
	}
	if m.profileSwitch.active {
		return m.profileSwitch.View()
	}
	if m.nowPlaying.IsFullscreen() {
		return m.nowPlaying.ViewFullscreen()
	}

	view := lipgloss.JoinVertical(lipgloss.Left,
		m.library.View(true),
		m.nowPlaying.ViewBar(false),
	)
	if m.showHelp {
		view = lipgloss.JoinVertical(lipgloss.Left, view, m.help.View(m.keys))
	}
	return view
}
