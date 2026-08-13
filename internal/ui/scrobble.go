package ui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/raloonsoc/sonora-cli/internal/subsonic"
)

// scrobbleThresholdRatio and scrobbleThresholdDuration implement the
// standard scrobble rule (ROADMAP Phase 6): a completed play is submitted
// once playback passes 50% of the track's duration, or 4 minutes,
// whichever comes first.
const (
	scrobbleThresholdRatio    = 0.5
	scrobbleThresholdDuration = 4 * time.Minute
)

// scrobbleState tracks whether the completed-play submission has already
// fired for the current track, so it's sent at most once (SPECS §4.2
// scrobble.view). The now-playing ping needs no such guard: it fires
// exactly once already, from trackStartedMsg.
type scrobbleState struct {
	songID    string
	submitted bool
}

// shouldSubmit reports whether position/duration have crossed the scrobble
// threshold.
func shouldSubmit(position, duration time.Duration) bool {
	if duration <= 0 {
		return false
	}
	if position >= scrobbleThresholdDuration {
		return true
	}
	return position.Seconds()/duration.Seconds() >= scrobbleThresholdRatio
}

// scrobbleCmd calls scrobble.view for songID, tagged with submission so the
// caller can tell a now-playing ping apart from a completed-play error.
func scrobbleCmd(client *subsonic.Client, songID string, submission bool) tea.Cmd {
	return func() tea.Msg {
		if err := client.Scrobble(context.Background(), songID, submission); err != nil {
			return playErrMsg{err: fmt.Errorf("scrobble: %w", err)}
		}
		return nil
	}
}
