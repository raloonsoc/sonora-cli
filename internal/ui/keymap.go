package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap centralizes every keybinding so help text and dispatch never drift
// apart (CODESTYLE §5: one responsibility, defined once).
type KeyMap struct {
	Quit          key.Binding
	Help          key.Binding
	Search        key.Binding
	Back          key.Binding
	Tab           key.Binding
	Up            key.Binding
	Down          key.Binding
	Enter         key.Binding
	AddToQueue    key.Binding
	Top           key.Binding
	Bottom        key.Binding
	PlayPause     key.Binding
	Next          key.Binding
	Prev          key.Binding
	SeekBack      key.Binding
	SeekFwd       key.Binding
	VolDown       key.Binding
	VolUp         key.Binding
	SwitchProfile key.Binding
	LyricsView    key.Binding
}

// DefaultKeyMap matches the bindings documented in README.md.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:          key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		Help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Search:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Back:          key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Tab:           key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "cycle panes")),
		Up:            key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:          key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open/play")),
		AddToQueue:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add to queue")),
		Top:           key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "top")),
		Bottom:        key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
		PlayPause:     key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "play/pause")),
		Next:          key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next track")),
		Prev:          key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "previous track")),
		SeekBack:      key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "seek -10s")),
		SeekFwd:       key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "seek +10s")),
		VolDown:       key.NewBinding(key.WithKeys("-"), key.WithHelp("-", "volume down")),
		VolUp:         key.NewBinding(key.WithKeys("+"), key.WithHelp("+", "volume up")),
		SwitchProfile: key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "switch profile")),
		LyricsView:    key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "lyrics view")),
	}
}

// ShortHelp implements help.KeyMap for the compact help line.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Search, k.Tab, k.Quit}
}

// FullHelp implements help.KeyMap for the expanded `?` help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.AddToQueue, k.Top, k.Bottom},
		{k.PlayPause, k.Next, k.Prev, k.SeekBack, k.SeekFwd, k.VolDown, k.VolUp},
		{k.Search, k.Tab, k.LyricsView, k.SwitchProfile, k.Back, k.Help, k.Quit},
	}
}
