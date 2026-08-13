package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raloonsoc/sonora-cli/internal/player"
	"github.com/raloonsoc/sonora-cli/internal/subsonic"
)

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

	err error
}

func newNowPlayingModel(client *subsonic.Client, ctrl *player.Controller, initialVolume int) nowPlayingModel {
	return nowPlayingModel{
		client:   client,
		ctrl:     ctrl,
		progress: progress.New(progress.WithDefaultGradient()),
		volume:   initialVolume,
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

func (m nowPlayingModel) Update(msg tea.Msg) (nowPlayingModel, tea.Cmd) {
	switch msg := msg.(type) {
	case songSelectedMsg:
		m.queue.add(msg.song)
		m.queue.pos = len(m.queue.tracks) - 1
		m.duration = time.Duration(msg.song.Duration) * time.Second
		m.paused = false
		return m, m.playTrack(msg.song)

	case trackStartedMsg:
		m.duration = time.Duration(msg.song.Duration) * time.Second
		m.err = nil
		return m, nil

	case playErrMsg:
		m.err = msg.err
		return m, nil

	case positionTickMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.position = time.Duration(msg.Position * float64(time.Second))
		if msg.Idle {
			return m.advanceQueue()
		}
		return m, nil

	case tea.KeyMsg:
		keys := DefaultKeyMap()
		switch {
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

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	metaStyle   = lipgloss.NewStyle().Faint(true)
	borderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
)

func (m nowPlayingModel) View() string {
	if m.err != nil {
		return errorView(m.err)
	}

	s, ok := m.queue.current()
	if !ok {
		return borderStyle.Render("Nothing playing — pick a track from the library.")
	}

	pct := 0.0
	if m.duration > 0 {
		pct = m.position.Seconds() / m.duration.Seconds()
	}

	status := "▶"
	if m.paused {
		status = "⏸"
	}

	body := fmt.Sprintf(
		"%s\n%s\n\n%s %s / %s  🔊 %d%%\n%s",
		titleStyle.Render(s.Title),
		metaStyle.Render(fmt.Sprintf("%s · %s", s.Artist, s.Album)),
		status,
		formatDuration(m.position),
		formatDuration(m.duration),
		m.volume,
		m.progress.ViewAs(pct),
	)

	return borderStyle.Render(body)
}

func formatDuration(d time.Duration) string {
	total := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}

func errorView(err error) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: " + err.Error())
}
