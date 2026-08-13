package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // getCoverArt.view commonly serves JPEG
	_ "image/png"  // ...and occasionally PNG
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raloonsoc/sonora-cli/internal/artwork"
	"github.com/raloonsoc/sonora-cli/internal/player"
	"github.com/raloonsoc/sonora-cli/internal/subsonic"
)

// coverArtSize is the pixel hint passed to getCoverArt.view — matched
// loosely to how much of a terminal cell grid the art occupies (SPECS §6.2).
const coverArtSize = 300

// queue is the CLI-local, ephemeral session playlist. Server playlists are
// read-only (SPECS §11), so "add to queue" has no server counterpart in v1.
type queue struct {
	tracks []subsonic.Song
	pos    int
}

func (q *queue) current() (subsonic.Song, bool) {
	if q.pos < 0 || q.pos >= len(q.tracks) {
		return subsonic.Song{}, false
	}
	return q.tracks[q.pos], true
}

func (q *queue) add(s subsonic.Song) {
	q.tracks = append(q.tracks, s)
	if q.pos < 0 {
		q.pos = 0
	}
}

func (q *queue) next() (subsonic.Song, bool) {
	if q.pos+1 >= len(q.tracks) {
		return subsonic.Song{}, false
	}
	q.pos++
	return q.current()
}

func (q *queue) prev() (subsonic.Song, bool) {
	if q.pos-1 < 0 {
		return subsonic.Song{}, false
	}
	q.pos--
	return q.current()
}

// nowPlayingModel renders the current track, playback progress, and
// transport controls.
type nowPlayingModel struct {
	client   *subsonic.Client
	ctrl     *player.Controller
	progress progress.Model
	queue    queue

	position time.Duration
	duration time.Duration
	paused   bool
	volume   int

	art      artworkState
	artCache *artwork.Cache
	artTerm  artwork.TermType
	artMode  artwork.Mode

	lyrics      lyricsState
	lyricsShown bool

	scrobble scrobbleState
	accent   accentState

	width  int // content width in terminal columns, set by App on WindowSizeMsg
	height int // full-screen height, set by App on WindowSizeMsg; unused by the bar view

	fullscreen bool // "Spotify mode": centered art + large synced lyrics

	err error
}

// artworkState holds the rendered cover art for the current track, kept
// separate from playback state so a re-render only happens on track change
// (SPECS §6.3), not on every ~500ms position tick.
type artworkState struct {
	albumID  string
	rendered string
}

// minContentWidth is the floor nowPlayingModel renders at before the first
// WindowSizeMsg arrives, so the very first frame isn't drawn at width 0.
const minContentWidth = 30

// progressBarWidth is fixed in the bottom bar view: it shares one row with
// art, track info, and transport text, so it can't grow with the terminal
// the way a full-panel progress bar could.
const progressBarWidth = 20

func newNowPlayingModel(client *subsonic.Client, ctrl *player.Controller, initialVolume int, term artwork.TermType, mode artwork.Mode, lyricsEnabled bool) nowPlayingModel {
	return nowPlayingModel{
		client:      client,
		ctrl:        ctrl,
		progress:    progress.New(progress.WithDefaultGradient(), progress.WithWidth(progressBarWidth)),
		volume:      initialVolume,
		artCache:    artwork.NewCache(0),
		artTerm:     term,
		artMode:     mode,
		lyricsShown: lyricsEnabled,
		accent:      accentState{color: defaultAccent},
		width:       minContentWidth,
	}
}

// positionTickMsg drives the progress bar and lyrics highlight from the
// player's polling stream (SPECS §5.3). App owns the channel (see New) and
// re-issues the receiving Cmd after each tick.
type positionTickMsg player.PositionUpdate

func (m nowPlayingModel) playTrack(s subsonic.Song) tea.Cmd {
	return func() tea.Msg {
		url := m.client.StreamURL(s.ID)
		if err := m.ctrl.Play(url); err != nil {
			return playErrMsg{err: fmt.Errorf("play %s: %w", s.Title, err)}
		}
		return trackStartedMsg{song: s}
	}
}

type trackStartedMsg struct{ song subsonic.Song }
type playErrMsg struct{ err error }

// artLoadedMsg carries a rendered cover art string, or a decode/fetch
// failure that should degrade silently (missing art is not a playback
// error worth surfacing to errorView).
type artLoadedMsg struct {
	albumID  string
	rendered string
}

// coverArtKey picks the ID getCoverArt.view should be called with: the
// song's own cover (rare — usually set only when it differs from the
// album's), falling back to the album ID.
func coverArtKey(s subsonic.Song) string {
	if s.CoverArt != "" {
		return s.CoverArt
	}
	return s.AlbumID
}

// artBarCols/artFullscreenCols bound cover art width for the two views: a
// thumbnail in the bottom bar, a larger centered image in fullscreen
// lyrics mode.
const (
	artBarCols        = 8
	artFullscreenCols = 34
)

// artCols picks the target width for the currently active view, so a mode
// switch (or a resize) re-renders art at the right size instead of
// reusing whatever was cached for the other view.
func (m nowPlayingModel) artCols() int {
	if m.fullscreen {
		cols := m.width
		if cols > artFullscreenCols {
			cols = artFullscreenCols
		}
		if cols < 10 {
			cols = 10
		}
		return cols
	}
	return artBarCols
}

// loadArt fetches and renders cover art for albumID, using the cache when
// available. mode == ModeOff skips the fetch entirely — no network call for
// art the user asked not to render. Re-renders only on track change per
// SPECS §6.3: a no-op if albumID already matches what's on screen.
func (m nowPlayingModel) loadArt(albumID string) tea.Cmd {
	if albumID == "" || m.artMode == artwork.ModeOff || albumID == m.art.albumID {
		return nil
	}
	artCols := m.artCols()
	return func() tea.Msg {
		if cached, ok := m.artCache.Get(albumID); ok {
			rendered, err := artwork.RenderString(cached, m.artTerm, m.artMode, artCols)
			if err != nil {
				return nil // degrade silently: keep showing the previous frame
			}
			return artLoadedMsg{albumID: albumID, rendered: rendered}
		}

		data, err := m.client.GetCoverArt(context.Background(), albumID, coverArtSize)
		if err != nil {
			return nil
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil
		}
		m.artCache.Put(albumID, img)

		rendered, err := artwork.RenderString(img, m.artTerm, m.artMode, artCols)
		if err != nil {
			return nil
		}
		return artLoadedMsg{albumID: albumID, rendered: rendered}
	}
}

func (m nowPlayingModel) Update(msg tea.Msg) (nowPlayingModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.fullscreen {
			m.progress.Width = m.width / 3
		} else {
			m.progress.Width = progressBarWidth
		}
		// Force a re-render at the new width rather than waiting for the
		// next track change — a live resize should update the art size
		// immediately, not stay stuck at whatever it was. loadArt is a
		// no-op if nothing is queued (coverArtKey of a zero Song is "").
		m.art = artworkState{}
		s, _ := m.queue.current()
		return m, m.loadArt(coverArtKey(s))

	case songSelectedMsg:
		m.queue.add(msg.song)
		m.queue.pos = len(m.queue.tracks) - 1
		m.duration = time.Duration(msg.song.Duration) * time.Second
		m.paused = false
		return m, m.playTrack(msg.song)

	case trackStartedMsg:
		m.duration = time.Duration(msg.song.Duration) * time.Second
		m.err = nil
		m.scrobble = scrobbleState{songID: msg.song.ID}
		var cmds []tea.Cmd
		if cmd := m.loadArt(coverArtKey(msg.song)); cmd != nil {
			cmds = append(cmds, cmd)
		}
		if m.lyricsShown {
			if cmd := loadLyrics(m.client, m.lyrics, msg.song); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if cmd := loadAccent(m.client, m.accent, msg.song.AlbumID); cmd != nil {
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, scrobbleCmd(m.client, msg.song.ID, false)) // now-playing ping
		return m, tea.Batch(cmds...)

	case artLoadedMsg:
		m.art = artworkState(msg)
		return m, nil

	case accentLoadedMsg:
		m.accent = accentState(msg)
		m.progress = progress.New(progress.WithSolidFill(string(m.accent.color)))
		return m, nil

	case lyricsLoadedMsg:
		m.lyrics = lyricsState(msg)
		return m, nil

	case playErrMsg:
		m.err = msg.err
		return m, nil

	case positionTickMsg:
		if msg.Err != nil {
			if errors.Is(msg.Err, player.ErrPropertyUnavailable) {
				// mpv has nothing loaded yet (before the first track, or
				// briefly between tracks) — expected, not a failure.
				return m, nil
			}
			m.err = msg.Err
			return m, nil
		}
		m.position = time.Duration(msg.Position * float64(time.Second))

		var cmd tea.Cmd
		if !m.scrobble.submitted && m.scrobble.songID != "" && shouldSubmit(m.position, m.duration) {
			m.scrobble.submitted = true
			cmd = scrobbleCmd(m.client, m.scrobble.songID, true)
		}

		if msg.Idle {
			mNext, nextCmd := m.advanceQueue()
			return mNext, tea.Batch(cmd, nextCmd)
		}
		return m, cmd

	case tea.KeyMsg:
		keys := DefaultKeyMap()
		switch {
		case key.Matches(msg, keys.LyricsView):
			return m.toggleFullscreen()
		case key.Matches(msg, keys.Back) && m.fullscreen:
			return m.toggleFullscreen()
		case key.Matches(msg, keys.PlayPause):
			return m.togglePause()
		case key.Matches(msg, keys.Next):
			return m.playQueueNext()
		case key.Matches(msg, keys.Prev):
			return m.playQueuePrev()
		case key.Matches(msg, keys.SeekBack):
			return m, m.seekRelative(-10)
		case key.Matches(msg, keys.SeekFwd):
			return m, m.seekRelative(10)
		case key.Matches(msg, keys.VolDown):
			return m.adjustVolume(-5)
		case key.Matches(msg, keys.VolUp):
			return m.adjustVolume(5)
		}
	}
	return m, nil
}

// IsFullscreen reports whether "Spotify mode" is active, so App can decide
// whether to render the library pane at all this frame.
func (m nowPlayingModel) IsFullscreen() bool {
	return m.fullscreen
}

// toggleFullscreen switches between the bottom-bar view and the centered
// "Spotify mode" lyrics view, forcing an art re-render at the new target
// size (artBarCols vs artFullscreenCols) rather than reusing whatever
// string was rendered for the other view.
func (m nowPlayingModel) toggleFullscreen() (nowPlayingModel, tea.Cmd) {
	m.fullscreen = !m.fullscreen
	m.art = artworkState{}
	s, _ := m.queue.current()
	return m, m.loadArt(coverArtKey(s))
}

func (m nowPlayingModel) togglePause() (nowPlayingModel, tea.Cmd) {
	paused := !m.paused
	m.paused = paused
	return m, func() tea.Msg {
		var err error
		if paused {
			err = m.ctrl.Pause()
		} else {
			err = m.ctrl.Resume()
		}
		if err != nil {
			return playErrMsg{err: fmt.Errorf("toggle pause: %w", err)}
		}
		return nil
	}
}

func (m nowPlayingModel) seekRelative(deltaSeconds float64) tea.Cmd {
	target := m.position.Seconds() + deltaSeconds
	if target < 0 {
		target = 0
	}
	return func() tea.Msg {
		if err := m.ctrl.Seek(target); err != nil {
			return playErrMsg{err: fmt.Errorf("seek: %w", err)}
		}
		return nil
	}
}

func (m nowPlayingModel) adjustVolume(delta int) (nowPlayingModel, tea.Cmd) {
	v := m.volume + delta
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	m.volume = v
	return m, func() tea.Msg {
		if err := m.ctrl.SetVolume(v); err != nil {
			return playErrMsg{err: fmt.Errorf("set volume: %w", err)}
		}
		return nil
	}
}

func (m nowPlayingModel) playQueueNext() (nowPlayingModel, tea.Cmd) {
	s, ok := m.queue.next()
	if !ok {
		return m, nil
	}
	return m, m.playTrack(s)
}

func (m nowPlayingModel) playQueuePrev() (nowPlayingModel, tea.Cmd) {
	s, ok := m.queue.prev()
	if !ok {
		return m, nil
	}
	return m, m.playTrack(s)
}

// advanceQueue auto-advances on track end (mpv idle-active).
func (m nowPlayingModel) advanceQueue() (nowPlayingModel, tea.Cmd) {
	s, ok := m.queue.next()
	if !ok {
		return m, nil
	}
	return m, m.playTrack(s)
}

// ViewBar renders the now-playing state as a single horizontal strip:
// small cover art, track info, progress, and transport status — meant for
// a fixed-height bar docked at the bottom of the screen (like a desktop
// Spotify client), not a tall side panel.
func (m nowPlayingModel) ViewBar(focused bool) string {
	style := barStyle
	if focused {
		style = style.BorderForeground(focusedBorderColor)
	} else {
		style = style.BorderForeground(m.accent.color)
	}

	if m.err != nil {
		return style.Width(m.width).Render(errorView(m.err))
	}

	s, ok := m.queue.current()
	if !ok {
		return style.Width(m.width).Render(metaStyle.Render("Nothing playing — pick a track from the library."))
	}

	status := "▶"
	if m.paused {
		status = "⏸"
	}

	info := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(truncate(s.Title, barInfoWidth)),
		metaStyle.Render(truncate(fmt.Sprintf("%s · %s", s.Artist, s.Album), barInfoWidth)),
	)

	pct := 0.0
	if m.duration > 0 {
		pct = m.position.Seconds() / m.duration.Seconds()
	}
	transport := fmt.Sprintf(
		"%s %s %s / %s   🔊 %d%%   [L] lyrics",
		status, m.progress.ViewAs(pct), formatDuration(m.position), formatDuration(m.duration), m.volume,
	)

	row := lipgloss.JoinHorizontal(lipgloss.Center, m.artBarThumbnail(), "  ", info, "   ", transport)
	return style.Width(m.width).Render(row)
}

// barInfoWidth caps the title/artist column so a long track name doesn't
// push the progress bar and controls off the edge of a narrow terminal.
const barInfoWidth = 28

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}

// artBarThumbnail returns the cached art already sized for the bar (see
// artCols/barArtCols), or a blank placeholder of the same footprint so the
// bar's height doesn't jump around while art is loading.
func (m nowPlayingModel) artBarThumbnail() string {
	if m.art.rendered != "" {
		return m.art.rendered
	}
	return ""
}

// ViewFullscreen renders the "Spotify mode" view: small centered art, large
// centered synced lyrics, transport status below — toggled by the L key
// (ROADMAP-adjacent UX request, not tied to a SPECS section).
func (m nowPlayingModel) ViewFullscreen() string {
	if m.err != nil {
		return errorView(m.err)
	}

	s, ok := m.queue.current()
	if !ok {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			metaStyle.Render("Nothing playing — pick a track from the library."))
	}

	status := "▶"
	if m.paused {
		status = "⏸"
	}
	pct := 0.0
	if m.duration > 0 {
		pct = m.position.Seconds() / m.duration.Seconds()
	}

	header := lipgloss.JoinVertical(lipgloss.Center,
		m.art.rendered,
		titleStyle.Render(s.Title),
		metaStyle.Render(fmt.Sprintf("%s · %s", s.Artist, s.Album)),
	)

	lyricsView := m.lyrics.View(int(m.position.Milliseconds()), m.accent.color)
	if lyricsView == "" {
		lyricsView = metaStyle.Render("No lyrics available for this track.")
	}

	footer := fmt.Sprintf(
		"%s %s / %s   🔊 %d%%   [esc/L] back",
		status, formatDuration(m.position), formatDuration(m.duration), m.volume,
	)

	body := lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		m.progress.ViewAs(pct),
		"",
		centeredStyle.Width(m.width).Render(lyricsView),
		"",
		footer,
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}

func formatDuration(d time.Duration) string {
	total := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}

func errorView(err error) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: " + err.Error())
}
