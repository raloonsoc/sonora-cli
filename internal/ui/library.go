package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raloonsoc/sonora-cli/internal/subsonic"
)

// libraryLevel is the depth of the browse stack: the top-level browse menu,
// then artists (or an album-list variant), then albums, then songs.
type libraryLevel int

const (
	levelBrowse libraryLevel = iota
	levelArtists
	levelGenres
	levelPlaylists
	levelMusicFolders
	levelAlbumList
	levelAlbums
	levelSongs
)

// browseEntry is one root-menu choice, wired to the endpoint it opens.
type browseEntry struct {
	title     string
	nextLevel libraryLevel
	albumType subsonic.AlbumListType // only meaningful when nextLevel == levelAlbumList
}

var browseEntries = []browseEntry{
	{title: "Artists", nextLevel: levelArtists},
	{title: "Recently Played", nextLevel: levelAlbumList, albumType: subsonic.AlbumListRecent},
	{title: "Newest", nextLevel: levelAlbumList, albumType: subsonic.AlbumListNewest},
	{title: "Frequently Played", nextLevel: levelAlbumList, albumType: subsonic.AlbumListFrequent},
	{title: "Random", nextLevel: levelAlbumList, albumType: subsonic.AlbumListRandom},
	{title: "Genres", nextLevel: levelGenres},
	{title: "Playlists", nextLevel: levelPlaylists},
	{title: "Music Folders", nextLevel: levelMusicFolders},
}

// browseItem adapts a browseEntry to bubbles/list.Item.
type browseItem struct{ entry browseEntry }

func (i browseItem) Title() string       { return i.entry.title }
func (i browseItem) Description() string { return "" }
func (i browseItem) FilterValue() string { return i.entry.title }

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

	// artistID/albumID/albumListType remember the selection at each level
	// so Back can re-fetch the right parent list rather than losing it.
	artistID      string
	albumID       string
	albumListType subsonic.AlbumListType

	err error
}

func newLibraryModel(client *subsonic.Client) libraryModel {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Browse"
	l.SetShowHelp(false)
	return libraryModel{client: client, list: l, level: levelBrowse}
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

// listLoadedMsg carries the result of any of the flat, single-request
// browse-menu loads (genres, playlists, music folders, an album-list
// variant) — unlike artists/albums/songs, these share one shape, so one
// message type covers all of them.
type listLoadedMsg struct {
	title string // list.Model.Title to set on success
	items []list.Item
	err   error
}

func (m libraryModel) loadBrowseMenu() tea.Cmd {
	items := make([]list.Item, len(browseEntries))
	for i, e := range browseEntries {
		items[i] = browseItem{entry: e}
	}
	return func() tea.Msg {
		return listLoadedMsg{title: "Browse", items: items}
	}
}

func (m libraryModel) loadGenres() tea.Cmd {
	return func() tea.Msg {
		genres, err := m.client.GetGenres(context.Background())
		if err != nil {
			return listLoadedMsg{err: fmt.Errorf("load genres: %w", err)}
		}
		items := make([]list.Item, len(genres))
		for i, g := range genres {
			items[i] = libraryItem{
				id:       g.Value,
				title:    g.Value,
				subtitle: fmt.Sprintf("%d albums", g.AlbumCount),
			}
		}
		return listLoadedMsg{title: "Genres", items: items}
	}
}

func (m libraryModel) loadPlaylists() tea.Cmd {
	return func() tea.Msg {
		playlists, err := m.client.GetPlaylists(context.Background())
		if err != nil {
			return listLoadedMsg{err: fmt.Errorf("load playlists: %w", err)}
		}
		items := make([]list.Item, len(playlists))
		for i, p := range playlists {
			items[i] = libraryItem{
				id:       p.ID,
				title:    p.Name,
				subtitle: fmt.Sprintf("%d tracks", p.SongCount),
			}
		}
		return listLoadedMsg{title: "Playlists", items: items}
	}
}

func (m libraryModel) loadMusicFolders() tea.Cmd {
	return func() tea.Msg {
		folders, err := m.client.GetMusicFolders(context.Background())
		if err != nil {
			return listLoadedMsg{err: fmt.Errorf("load music folders: %w", err)}
		}
		items := make([]list.Item, len(folders))
		for i, f := range folders {
			items[i] = libraryItem{title: f.Name}
		}
		return listLoadedMsg{title: "Music Folders", items: items}
	}
}

func (m libraryModel) loadAlbumList(listType subsonic.AlbumListType) tea.Cmd {
	return func() tea.Msg {
		albums, err := m.client.GetAlbumList2(context.Background(), listType, 50, 0)
		if err != nil {
			return listLoadedMsg{err: fmt.Errorf("load album list %s: %w", listType, err)}
		}
		items := make([]list.Item, len(albums))
		for i, al := range albums {
			items[i] = libraryItem{
				id:       al.ID,
				title:    al.Name,
				subtitle: fmt.Sprintf("%s · %d tracks", al.Artist, al.SongCount),
			}
		}
		return listLoadedMsg{title: string(listType), items: items}
	}
}

func (m libraryModel) loadArtists() tea.Cmd {
	return func() tea.Msg {
		idx, err := m.client.GetArtists(context.Background())
		if err != nil {
			return listLoadedMsg{err: fmt.Errorf("load artists: %w", err)}
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
		return listLoadedMsg{title: "Artists", items: items}
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
	return m.loadBrowseMenu()
}

func (m libraryModel) Update(msg tea.Msg) (libraryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case listLoadedMsg:
		m.err = msg.err
		if msg.err == nil {
			m.list.SetItems(msg.items)
			m.list.Title = msg.title
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
	switch m.level {
	case levelBrowse:
		bi, ok := m.list.SelectedItem().(browseItem)
		if !ok {
			return m, nil
		}
		return m.openBrowseEntry(bi.entry)

	case levelArtists:
		item, ok := m.list.SelectedItem().(libraryItem)
		if !ok {
			return m, nil
		}
		m.level = levelAlbums
		m.artistID = item.id
		return m, m.loadAlbums(item.id)

	case levelAlbumList:
		item, ok := m.list.SelectedItem().(libraryItem)
		if !ok {
			return m, nil
		}
		m.level = levelSongs
		m.albumID = item.id
		return m, m.loadSongs(item.id)

	case levelAlbums:
		item, ok := m.list.SelectedItem().(libraryItem)
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

// openBrowseEntry dispatches a root-menu selection to the level and load it
// opens (ROADMAP Phase 6: getGenres/getPlaylists/getMusicFolders/
// getAlbumList2 as browse entry points alongside Artists).
func (m libraryModel) openBrowseEntry(e browseEntry) (libraryModel, tea.Cmd) {
	m.level = e.nextLevel
	m.artistID = "" // clear stale state from a previous artist->album->songs run
	switch e.nextLevel {
	case levelArtists:
		return m, m.loadArtists()
	case levelAlbumList:
		m.albumListType = e.albumType
		return m, m.loadAlbumList(e.albumType)
	case levelGenres:
		return m, m.loadGenres()
	case levelPlaylists:
		return m, m.loadPlaylists()
	case levelMusicFolders:
		return m, m.loadMusicFolders()
	}
	return m, nil
}

func (m libraryModel) goBack() (libraryModel, tea.Cmd) {
	switch m.level {
	case levelSongs:
		if m.artistID != "" {
			m.level = levelAlbums
			return m, m.loadAlbums(m.artistID)
		}
		m.level = levelAlbumList
		return m, m.loadAlbumList(m.albumListType)
	case levelAlbums:
		m.level = levelArtists
		return m, m.loadArtists()
	case levelArtists, levelGenres, levelPlaylists, levelMusicFolders, levelAlbumList:
		m.level = levelBrowse
		return m, m.loadBrowseMenu()
	}
	return m, nil
}

func (m libraryModel) View() string {
	if m.err != nil {
		return errorView(m.err)
	}
	return m.list.View()
}
