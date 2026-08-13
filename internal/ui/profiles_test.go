package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewProfileSwitchModel_sortsNames(t *testing.T) {
	m := newProfileSwitchModel([]string{"vps", "home", "backup"}, "home")
	want := []string{"backup", "home", "vps"}
	for i, n := range want {
		if m.names[i] != n {
			t.Errorf("names[%d] = %q, want %q", i, m.names[i], n)
		}
	}
}

func TestProfileSwitchModel_open_cursorStartsOnCurrent(t *testing.T) {
	m := newProfileSwitchModel([]string{"a", "b", "c"}, "b")
	m = m.Open()
	if m.names[m.cursor] != "b" {
		t.Errorf("cursor at %q, want b", m.names[m.cursor])
	}
}

func TestProfileSwitchModel_enterSetsSwitchToAndQuits(t *testing.T) {
	m := newProfileSwitchModel([]string{"a", "b"}, "a").Open()
	m.cursor = 1

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.SwitchTo != "b" {
		t.Errorf("SwitchTo = %q, want b", updated.SwitchTo)
	}
	if cmd == nil {
		t.Fatal("expected a tea.Quit Cmd")
	}
}

func TestProfileSwitchModel_backCloses(t *testing.T) {
	m := newProfileSwitchModel([]string{"a"}, "a").Open()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.active {
		t.Error("expected active=false after Back")
	}
}

func TestProfileSwitchModel_navigationClampsAtEdges(t *testing.T) {
	m := newProfileSwitchModel([]string{"a", "b"}, "a").Open()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp}) // already at 0
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (clamped)", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}) // past the end
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (clamped)", m.cursor)
	}
}
