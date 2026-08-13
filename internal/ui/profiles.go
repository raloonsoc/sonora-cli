package ui

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// profileSwitchModel is a simple selectable list of configured profile
// names. Switching profiles means reconnecting to a different server and
// restarting mpv, so it's handled as a clean process restart rather than
// hot-swapping App's internals: selecting an entry quits the program and
// SwitchTo carries the chosen profile name back to main for relaunch
// (ROADMAP Phase 9).
type profileSwitchModel struct {
	names   []string
	current string
	cursor  int
	active  bool

	// SwitchTo is set when the user confirms a selection, for main to read
	// after tea.Program.Run returns.
	SwitchTo string
}

// newProfileSwitchModel takes profile names directly rather than a
// config.Profile map, so internal/ui doesn't need to import
// internal/config just to display a list of strings.
func newProfileSwitchModel(names []string, current string) profileSwitchModel {
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	return profileSwitchModel{names: sorted, current: current}
}

func (m profileSwitchModel) Open() profileSwitchModel {
	m.active = true
	for i, n := range m.names {
		if n == m.current {
			m.cursor = i
		}
	}
	return m
}

func (m profileSwitchModel) Close() profileSwitchModel {
	m.active = false
	return m
}

func (m profileSwitchModel) Update(msg tea.Msg) (profileSwitchModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	keys := DefaultKeyMap()
	switch {
	case key.Matches(keyMsg, keys.Back):
		return m.Close(), nil
	case key.Matches(keyMsg, keys.Down):
		if len(m.names) > 0 {
			m.cursor = min(m.cursor+1, len(m.names)-1)
		}
		return m, nil
	case key.Matches(keyMsg, keys.Up):
		m.cursor = max(m.cursor-1, 0)
		return m, nil
	case key.Matches(keyMsg, keys.Enter):
		if len(m.names) == 0 {
			return m, nil
		}
		m.SwitchTo = m.names[m.cursor]
		return m, tea.Quit
	}
	return m, nil
}

func (m profileSwitchModel) View() string {
	body := titleStyle.Render("Switch profile") + "\n\n"
	if len(m.names) == 0 {
		return body + "No other profiles configured."
	}
	for i, n := range m.names {
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}
		marker := ""
		if n == m.current {
			marker = " (current)"
		}
		body += fmt.Sprintf("%s%s%s\n", cursor, n, marker)
	}
	return body
}
