//go:build !windows

package player

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const positionPollInterval = 500 * time.Millisecond

// Controller is the high-level playback API used by the UI layer. It owns
// one mpv Process and its IPC connection for the session's duration.
type Controller struct {
	proc *Process
	ipc  *ipcConn
}

// New starts mpv and returns a ready Controller. Call Close to shut down
// mpv and release the socket.
func New() (*Controller, error) {
	proc, err := Start()
	if err != nil {
		return nil, err
	}

	ipc, err := newIPCConn(proc.SocketPath())
	if err != nil {
		_ = proc.Close()
		return nil, fmt.Errorf("player: connect to mpv: %w", err)
	}

	return &Controller{proc: proc, ipc: ipc}, nil
}

// Close shuts down the IPC connection and terminates mpv.
func (c *Controller) Close() error {
	_ = c.ipc.Close()
	return c.proc.Close()
}

// Play loads streamURL and starts playback. mpv performs the HTTP request
// itself, including Range requests for seeking (SPECS §5.2) — the CLI never
// proxies audio bytes.
func (c *Controller) Play(streamURL string) error {
	_, err := c.ipc.call([]any{"loadfile", streamURL})
	if err != nil {
		return fmt.Errorf("player: play: %w", err)
	}
	return nil
}

// Pause pauses playback.
func (c *Controller) Pause() error {
	return c.setProperty("pause", true)
}

// Resume resumes playback.
func (c *Controller) Resume() error {
	return c.setProperty("pause", false)
}

// Seek moves playback to an absolute position in seconds.
func (c *Controller) Seek(seconds float64) error {
	_, err := c.ipc.call([]any{"seek", seconds, "absolute"})
	if err != nil {
		return fmt.Errorf("player: seek to %.1fs: %w", seconds, err)
	}
	return nil
}

// SetVolume sets playback volume, 0-100.
func (c *Controller) SetVolume(n int) error {
	return c.setProperty("volume", n)
}

// Stop halts playback and unloads the current track.
func (c *Controller) Stop() error {
	_, err := c.ipc.call([]any{"stop"})
	if err != nil {
		return fmt.Errorf("player: stop: %w", err)
	}
	return nil
}

// Position returns the current playback position in seconds.
func (c *Controller) Position() (float64, error) {
	data, err := c.ipc.call([]any{"get_property", "time-pos"})
	if err != nil {
		return 0, fmt.Errorf("player: get position: %w", err)
	}
	var pos float64
	if err := json.Unmarshal(data, &pos); err != nil {
		return 0, fmt.Errorf("player: decode position: %w", err)
	}
	return pos, nil
}

// Idle reports whether mpv has no track loaded or has reached end-of-file,
// used to drive queue advancement.
func (c *Controller) Idle() (bool, error) {
	data, err := c.ipc.call([]any{"get_property", "idle-active"})
	if err != nil {
		return false, fmt.Errorf("player: get idle state: %w", err)
	}
	var idle bool
	if err := json.Unmarshal(data, &idle); err != nil {
		return false, fmt.Errorf("player: decode idle state: %w", err)
	}
	return idle, nil
}

func (c *Controller) setProperty(name string, value any) error {
	_, err := c.ipc.call([]any{"set_property", name, value})
	if err != nil {
		return fmt.Errorf("player: set %s: %w", name, err)
	}
	return nil
}

// PositionUpdate is one tick from PositionStream.
type PositionUpdate struct {
	Position float64
	Idle     bool
	Err      error
}

// PositionStream polls playback position every ~500ms (SPECS §5.3) and
// sends updates on the returned channel until ctx is cancelled. The
// goroutine it spawns exits when ctx is done, closing the channel — its
// lifecycle is fully owned by the caller's context (CODESTYLE §5).
func (c *Controller) PositionStream(ctx context.Context) <-chan PositionUpdate {
	ch := make(chan PositionUpdate)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(positionPollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pos, posErr := c.Position()
				idle, idleErr := c.Idle()
				err := posErr
				if err == nil {
					err = idleErr
				}

				update := PositionUpdate{Position: pos, Idle: idle, Err: err}
				select {
				case ch <- update:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch
}
