package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raloonsoc/sonora-cli/internal/config"
)

// firstRunField is one prompt in the setup sequence.
type firstRunField int

const (
	fieldURL firstRunField = iota
	fieldUsername
	fieldPassword
	fieldDone
)

// FirstRunModel prompts for server URL, username, and password on first
// launch, per ROADMAP Phase 3. It has no dependency on the rest of the UI
// so it can run and exit before App is constructed.
type FirstRunModel struct {
	field  firstRunField
	inputs [3]textinput.Model
	Result *config.Profile // set once the flow completes
	Done   bool
	Err    error
}

// NewFirstRunModel builds the prompt sequence.
func NewFirstRunModel() FirstRunModel {
	url := textinput.New()
	url.Placeholder = "https://music.example.com"
	url.Focus()

	username := textinput.New()
	username.Placeholder = "username"

	password := textinput.New()
	password.Placeholder = "password"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'

	return FirstRunModel{inputs: [3]textinput.Model{url, username, password}}
}

func (m FirstRunModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m FirstRunModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.Err = fmt.Errorf("setup cancelled")
			m.Done = true
			return m, tea.Quit
		case "enter":
			return m.advance()
		}
	}

	var cmd tea.Cmd
	m.inputs[m.field], cmd = m.inputs[m.field].Update(msg)
	return m, cmd
}

func (m FirstRunModel) advance() (FirstRunModel, tea.Cmd) {
	if m.inputs[m.field].Value() == "" {
		return m, nil // required field: stay put until something is typed
	}

	m.inputs[m.field].Blur()
	m.field++

	if m.field == fieldDone {
		m.Result = &config.Profile{
			URL:      m.inputs[fieldURL].Value(),
			Username: m.inputs[fieldUsername].Value(),
			Password: m.inputs[fieldPassword].Value(),
			Auth:     "subsonic",
		}
		m.Done = true
		return m, tea.Quit
	}

	m.inputs[m.field].Focus()
	return m, textinput.Blink
}

func (m FirstRunModel) View() string {
	labels := [3]string{"Server URL", "Username", "Password"}
	body := titleStyle.Render("Welcome to sonora-cli — let's connect to your server.") + "\n\n"
	for i := fieldURL; i < firstRunField(len(m.inputs)); i++ {
		if i > m.field {
			continue // not reached yet
		}
		body += fmt.Sprintf("%s\n%s\n\n", labels[i], m.inputs[i].View())
	}
	return borderStyle.Render(body)
}
