package ui

import (
	"errors"
	"testing"
	"time"

	"github.com/raloonsoc/sonora-cli/internal/player"
	"github.com/raloonsoc/sonora-cli/internal/subsonic"
)

func TestQueue_addAndCurrent(t *testing.T) {
	var q queue
	if _, ok := q.current(); ok {
		t.Fatal("current() on empty queue should report false")
	}

	q.add(subsonic.Song{ID: "1", Title: "One"})
	got, ok := q.current()
	if !ok || got.ID != "1" {
		t.Fatalf("current() = %+v, %v, want track 1", got, ok)
	}
}

func TestQueue_nextPrev(t *testing.T) {
	var q queue
	q.add(subsonic.Song{ID: "1"})
	q.add(subsonic.Song{ID: "2"})
	q.add(subsonic.Song{ID: "3"})
	q.pos = 0

	tests := []struct {
		name   string
		action func() (subsonic.Song, bool)
		wantID string
		wantOK bool
	}{
		{"next to 2", q.next, "2", true},
		{"next to 3", q.next, "3", true},
		{"next past end", q.next, "", false},
		{"prev to 2", q.prev, "2", true},
		{"prev to 1", q.prev, "1", true},
		{"prev before start", q.prev, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.action()
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got.ID != tt.wantID {
				t.Errorf("song ID = %q, want %q", got.ID, tt.wantID)
			}
		})
	}
}

func TestQueue_addSetsPosOnFirstTrack(t *testing.T) {
	q := queue{pos: -1}
	q.add(subsonic.Song{ID: "1"})
	if q.pos != 0 {
		t.Errorf("pos = %d, want 0 after adding to an empty queue", q.pos)
	}
}

func TestNowPlayingModel_positionTick_propertyUnavailableIsSilent(t *testing.T) {
	m := nowPlayingModel{}

	updated, cmd := m.Update(positionTickMsg{Err: player.ErrPropertyUnavailable})
	if updated.err != nil {
		t.Errorf("err = %v, want nil — property unavailable means nothing is loaded yet, not a failure", updated.err)
	}
	if cmd != nil {
		t.Errorf("cmd = %v, want nil", cmd)
	}
}

func TestNowPlayingModel_positionTick_realErrorIsSurfaced(t *testing.T) {
	m := nowPlayingModel{}
	wantErr := errors.New("mpv crashed")

	updated, _ := m.Update(positionTickMsg{Err: wantErr})
	if !errors.Is(updated.err, wantErr) {
		t.Errorf("err = %v, want %v", updated.err, wantErr)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds int
		want    string
	}{
		{0, "0:00"},
		{5, "0:05"},
		{65, "1:05"},
		{407, "6:47"},
	}
	for _, tt := range tests {
		got := formatDuration(time.Duration(tt.seconds) * time.Second)
		if got != tt.want {
			t.Errorf("formatDuration(%ds) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}
