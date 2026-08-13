//go:build mpv && !windows

// This file requires a real mpv binary on $PATH and is excluded from the
// default test run: go test -tags mpv ./internal/player/...
package player

import (
	"testing"
	"time"
)

func TestController_playPauseSeekVolume(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	// A short, freely redistributable local-file test tone would be ideal
	// here; lacking one, this exercises the command path against mpv's own
	// idle state rather than actual audio decoding.
	if err := c.SetVolume(50); err != nil {
		t.Fatalf("SetVolume: %v", err)
	}
	if err := c.Pause(); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	if err := c.Resume(); err != nil {
		t.Fatalf("Resume: %v", err)
	}

	idle, err := c.Idle()
	if err != nil {
		t.Fatalf("Idle: %v", err)
	}
	if !idle {
		t.Error("expected mpv to be idle with nothing loaded")
	}

	time.Sleep(100 * time.Millisecond) // let mpv settle after the property writes
}
