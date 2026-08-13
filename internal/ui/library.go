package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raloonsoc/sonora-cli/internal/subsonic"
)

// libraryLevel is the depth of the browse stack: artists, then that
// artist's albums, then that album's songs.
type libraryLevel int

const (
	levelArtists libraryLevel = iota
	levelAlbums
	levelSongs
)

// libraryItem adapts a subsonic entity to bubbles/list.Item.
type libraryItem struct {
	id       string
	title    string
	subtitle string
}

func (i libraryItem) Title() string       { return i.title }
func (i libraryItem) Description() string { return i.subtitle }
func (i libraryItem) FilterValue() string { return i.title }

// libraryModel drives the artist → album → song browse pane.
type libraryModel struct {
	client *subsonic.Client
	list   list.Model
	level  libraryLevel

	// artistID/albumID remember the selection at each level so Back can
	// return without re-fetching from scratch.
	artistID string
	albumID  string

	err error
}

func newLibraryModel(client *subsonic.Client) libraryModel {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Artists"
	l.SetShowHelp(false)
	return libraryModel{client: client, list: l, level: levelArtists}
}

// artistsLoadedMsg carries the result of fetching the artist index.
type artistsLoadedMsg struct {
	items []list.Item
	err   error
}

// albumsLoadedMsg carries the result of fetching one artist's albums.
type albumsLoadedMsg struct {
	items []list.Item
	err   error
}

// songsLoadedMsg carries the result of fetching one album's songs.
type songsLoadedMsg struct {
	items []list.Item
	err   error
}

// songSelectedMsg signals the user chose a track to play.
type songSelectedMsg struct {
	song subsonic.Song
}

func (m libraryModel) loadArtists() tea.Cmd {
	return func() tea.Msg {
		idx, err := m.client.GetArtists(context.Background())
		if err != nil {
			return artistsLoadedMsg{err: fmt.Errorf("load artists: %w", err)}
		}
		var items []list.Item
		for _, group := range idx.Index {
			for _, a := range group.Artists {
				items = append(items, libraryItem{
					id:       a.ID,
					title:    a.Name,
					subtitle: fmt.Sprintf("%d albums", a.AlbumCount),
				})
			}
		}
		return artistsLoadedMsg{items: items}
	}
}

func (m libraryModel) loadAlbums(artistID string) tea.Cmd {
	return func() tea.Msg {
		artist, err := m.client.GetArtist(context.Background(), artistID)
		if err != nil {
			return albumsLoadedMsg{err: fmt.Errorf("load albums: %w", err)}
		}
		var items []list.Item
		for _, al := range artist.Album {
			items = append(items, libraryItem{
				id:       al.ID,
				title:    al.Name,
				subtitle: fmt.Sprintf("%d tracks", al.SongCount),
			})
		}
		return albumsLoadedMsg{items: items}
	}
}

func (m libraryModel) loadSongs(albumID string) tea.Cmd {
	return func() tea.Msg {
		album, err := m.client.GetAlbum(context.Background(), albumID)
		if err != nil {
			return songsLoadedMsg{err: fmt.Errorf("load songs: %w", err)}
		}
		var items []list.Item
		for _, s := range album.Song {
			items = append(items, songItem{song: s})
		}
		return songsLoadedMsg{items: items}
	}
}

// songItem is a libraryItem carrying the full Song, so selecting it can
// hand playback everything it needs without a second fetch.
type songItem struct {
	song subsonic.Song
}

func (i songItem) Title() string { return i.song.Title }
func (i songItem) Description() string {
	if i.song.Duration == 0 {
		return ""
	}
	return fmt.Sprintf("%d:%02d", i.song.Duration/60, i.song.Duration%60)
}
func (i songItem) FilterValue() string { return i.song.Title }

func (m libraryModel) Init() tea.Cmd {
	return m.loadArtists()
}

func (m libraryModel) Update(msg tea.Msg) (libraryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case artistsLoadedMsg:
		m.err = msg.err
		if msg.err == nil {
			m.list.SetItems(msg.items)
		}
		return m, nil

	case albumsLoadedMsg:
		m.err = msg.err
		if msg.err == nil {
			m.list.SetItems(msg.items)
			m.list.Title = "Albums"
		}
		return m, nil

	case songsLoadedMsg:
		m.err = msg.err
		if msg.err == nil {
			m.list.SetItems(msg.items)
			m.list.Title = "Songs"
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		keys := DefaultKeyMap()
		switch {
		case key.Matches(msg, keys.Enter):
			return m.selectCurrent()
		case key.Matches(msg, keys.Back):
			return m.goBack()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m libraryModel) selectCurrent() (libraryModel, tea.Cmd) {
	item, ok := m.list.SelectedItem().(libraryItem)
	switch m.level {
	case levelArtists:
		if !ok {
			return m, nil
		}
		m.level = levelAlbums
		m.artistID = item.id
		return m, m.loadAlbums(item.id)
	case levelAlbums:
		if !ok {
			return m, nil
		}
		m.level = levelSongs
		m.albumID = item.id
		return m, m.loadSongs(item.id)
	case levelSongs:
		si, ok := m.list.SelectedItem().(songItem)
		if !ok {
			return m, nil
		}
		return m, func() tea.Msg { return songSelectedMsg(si) }
	}
	return m, nil
}

func (m libraryModel) goBack() (libraryModel, tea.Cmd) {
	switch m.level {
	case levelSongs:
		m.level = levelAlbums
		return m, m.loadAlbums(m.artistID)
	case levelAlbums:
		m.level = levelArtists
		return m, m.loadArtists()
	}
	return m, nil
}

func (m libraryModel) View() string {
	if m.err != nil {
		return errorView(m.err)
	}
	return m.list.View()
}
