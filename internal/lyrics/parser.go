// Package lyrics parses OpenSubsonic lyrics responses into sorted lines and
// finds the active line for a given playback position (SPECS §8).
package lyrics

import (
	"sort"

	"github.com/raloonsoc/sonora-cli/internal/subsonic"
)

// Line is one lyrics line. StartMS is only meaningful when the parent
// Lyrics.Synced is true.
type Line struct {
	StartMS int
	Text    string
}

// Lyrics is a parsed, display-ready lyrics track: either synced
// (millisecond-timestamped) or plain unsynced text, one line per entry
// either way.
type Lyrics struct {
	Synced bool
	Lines  []Line
}

// Parse converts an OpenSubsonic getLyricsBySongId.view result into
// display-ready Lyrics, sorted by timestamp. If list has no structured
// lyrics, it falls back to plain, which may itself be empty — callers
// should treat a zero-length Lyrics.Lines as "no lyrics available", not an
// error.
func Parse(list *subsonic.LyricsList, plain *subsonic.LyricsPlain) Lyrics {
	if list != nil && len(list.StructuredLyrics) > 0 {
		return parseStructured(list.StructuredLyrics[0])
	}
	if plain != nil && plain.Value != "" {
		return parsePlain(plain.Value)
	}
	return Lyrics{}
}

func parseStructured(sl subsonic.StructuredLyrics) Lyrics {
	lines := make([]Line, len(sl.Line))
	for i, l := range sl.Line {
		lines[i] = Line{StartMS: l.Start, Text: l.Value}
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].StartMS < lines[j].StartMS })
	return Lyrics{Synced: sl.Synced, Lines: lines}
}

// parsePlain splits getLyrics.view's raw text into lines with no timestamp
// data — rendered as a static scrollable pane, never highlighted (SPECS §8).
func parsePlain(text string) Lyrics {
	var lines []Line
	start := 0
	for i := 0; i <= len(text); i++ {
		if i == len(text) || text[i] == '\n' {
			line := text[start:i]
			line = trimCR(line)
			lines = append(lines, Line{Text: line})
			start = i + 1
		}
	}
	return Lyrics{Synced: false, Lines: lines}
}

// trimCR strips a trailing \r left by CRLF line endings.
func trimCR(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\r' {
		return s[:len(s)-1]
	}
	return s
}
