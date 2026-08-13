package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raloonsoc/sonora-cli/internal/subsonic"
)

// searchDebounce delays search3.view calls until typing pauses, so each
// keystroke doesn't fire its own request.
const searchDebounce = 300 * time.Millisecond

// searchModel drives the `/` search overlay: a text input plus grouped
// artist/album/song results from search3.view. Selecting a song jumps
// straight to playback (ROADMAP Phase 6); artists and albums are
// informational for now — browsing into them reuses the library pane.
type searchModel struct {
	client *subsonic.Client
	input  textinput.Model
	active bool
	result *subsonic.SearchResult
	cursor int // index into flattened songs; only songs are selectable
	err    error
	// generation increments on every keystroke; a debounced search only
	// fires if it's still current when its timer elapses, so a stale
	// in-flight request can't overwrite a newer query's results.
	generation int
}

func newSearchModel(client *subsonic.Client) searchModel {
	ti := textinput.New()
	ti.Placeholder = "search artists, albums, songs..."
	return searchModel{client: client, input: ti}
}

// searchTickMsg fires searchDebounce after a keystroke, carrying the
// generation it was scheduled for.
type searchTickMsg struct {
	generation int
	query      string
}

// searchResultMsg carries a completed search3.view response.
type searchResultMsg struct {
	generation int
	result     *subsonic.SearchResult
	err        error
}

func (m searchModel) Open() searchModel {
	m.active = true
	m.input.Focus()
	return m
}

func (m searchModel) Close() searchModel {
	m.active = false
	m.input.Blur()
	m.input.SetValue("")
	m.result = nil
	m.cursor = 0
	return m
}

func (m searchModel) Update(msg tea.Msg) (searchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		keys := DefaultKeyMap()
		switch {
		case key.Matches(msg, keys.Back):
			return m.Close(), nil
		case key.Matches(msg, keys.Down) && m.hasSongs():
			m.cursor = min(m.cursor+1, len(m.result.Song)-1)
			return m, nil
		case key.Matches(msg, keys.Up) && m.hasSongs():
			m.cursor = max(m.cursor-1, 0)
			return m, nil
		case key.Matches(msg, keys.Enter) && m.hasSongs():
			return m.Close(), func() tea.Msg { return songSelectedMsg{song: m.result.Song[m.cursor]} }
		}

		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.generation++
		gen := m.generation
		query := m.input.Value()
		return m, tea.Batch(cmd, debounceSearch(gen, query))

	case searchTickMsg:
		if msg.generation != m.generation || msg.query == "" {
			return m, nil
		}
		return m, m.runSearch(msg.generation, msg.query)

	case searchResultMsg:
		if msg.generation != m.generation {
			return m, nil // stale response from a superseded query
		}
		m.result = msg.result
		m.err = msg.err
		m.cursor = 0
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m searchModel) hasSongs() bool {
	return m.result != nil && len(m.result.Song) > 0
}

func debounceSearch(generation int, query string) tea.Cmd {
	return tea.Tick(searchDebounce, func(time.Time) tea.Msg {
		return searchTickMsg{generation: generation, query: query}
	})
}

func (m searchModel) runSearch(generation int, query string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Search3(context.Background(), query)
		if err != nil {
			return searchResultMsg{generation: generation, err: fmt.Errorf("search: %w", err)}
		}
		return searchResultMsg{generation: generation, result: result}
	}
}

func (m searchModel) View() string {
	body := m.input.View()

	if m.err != nil {
		return body + "\n\n" + errorView(m.err)
	}
	if m.result == nil {
		return body
	}

	if len(m.result.Artist) > 0 {
		body += "\n\n" + titleStyle.Render("Artists")
		for _, a := range m.result.Artist {
			body += fmt.Sprintf("\n  %s", a.Name)
		}
	}
	if len(m.result.Album) > 0 {
		body += "\n\n" + titleStyle.Render("Albums")
		for _, al := range m.result.Album {
			body += fmt.Sprintf("\n  %s — %s", al.Name, al.Artist)
		}
	}
	if len(m.result.Song) > 0 {
		body += "\n\n" + titleStyle.Render("Songs (enter to play)")
		for i, s := range m.result.Song {
			cursor := "  "
			if i == m.cursor {
				cursor = "▸ "
			}
			body += fmt.Sprintf("\n%s%s — %s", cursor, s.Title, s.Artist)
		}
	}

	return body
}
