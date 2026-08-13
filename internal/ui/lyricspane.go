package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raloonsoc/sonora-cli/internal/lyrics"
	"github.com/raloonsoc/sonora-cli/internal/subsonic"
)

// lyricsPaneVisibleLines caps how many lines render around the active line,
// keeping the pane a fixed height regardless of the track's lyrics length.
const lyricsPaneVisibleLines = 7

// lyricsState holds the current track's parsed lyrics, kept separate from
// playback state so fetching only happens on track change, matching how
// artworkState is scoped (SPECS §8).
type lyricsState struct {
	songID string
	l      lyrics.Lyrics
}

// lyricsLoadedMsg carries parsed lyrics for songID, or an empty Lyrics when
// the server has none — that's not an error, it just hides the pane.
type lyricsLoadedMsg struct {
	songID string
	l      lyrics.Lyrics
}

// loadLyrics fetches structured lyrics for song, falling back to plain
// text keyed by artist/title when the structured endpoint returns nothing
// (SPECS §8). A no-op if song.ID already matches what's loaded.
func loadLyrics(client *subsonic.Client, current lyricsState, song subsonic.Song) tea.Cmd {
	if song.ID == "" || song.ID == current.songID {
		return nil
	}
	return func() tea.Msg {
		list, err := client.GetLyricsBySongID(context.Background(), song.ID)
		if err != nil {
			list = nil // degrade silently: missing lyrics support is normal, not fatal
		}

		var plain *subsonic.LyricsPlain
		if list == nil || len(list.StructuredLyrics) == 0 {
			plain, err = client.GetLyrics(context.Background(), song.Artist, song.Title)
			if err != nil {
				plain = nil
			}
		}

		return lyricsLoadedMsg{songID: song.ID, l: lyrics.Parse(list, plain)}
	}
}

var (
	lyricLineStyle       = lipgloss.NewStyle().Faint(true)
	lyricActiveLineStyle = lipgloss.NewStyle().Bold(true)
)

// View renders a fixed-height window of lines centered on the active line.
// Unsynced lyrics render as a static pane with no highlight; no lyrics
// renders nothing at all, so the pane disappears rather than showing an
// empty box (SPECS §8).
func (s lyricsState) View(positionMS int) string {
	if len(s.l.Lines) == 0 {
		return ""
	}

	active := lyrics.ActiveLine(s.l, positionMS)
	start, end := windowAround(active, len(s.l.Lines), lyricsPaneVisibleLines)

	var out string
	for i := start; i < end; i++ {
		line := s.l.Lines[i].Text
		if line == "" {
			line = " "
		}
		if i == active {
			out += lyricActiveLineStyle.Render(line) + "\n"
		} else {
			out += lyricLineStyle.Render(line) + "\n"
		}
	}
	return out
}

// windowAround computes a [start, end) slice window of size around center,
// clamped to [0, total), so the active line stays vertically centered
// until it nears either edge of the lyrics.
func windowAround(center, total, size int) (start, end int) {
	if center < 0 {
		center = 0
	}
	start = center - size/2
	if start < 0 {
		start = 0
	}
	end = start + size
	if end > total {
		end = total
		start = end - size
		if start < 0 {
			start = 0
		}
	}
	return start, end
}
