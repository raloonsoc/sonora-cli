//go:build !windows

// Package player wraps mpv as a subprocess controlled over its JSON IPC
// socket, per SPECS §5. sonora-cli never decodes or proxies audio itself.
//
// Windows named-pipe IPC is not implemented — this package targets Unix
// domain sockets only, and is explicitly unsupported on Windows for v1
// (SPECS §5.2, ROADMAP Phase 2).
package player

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Process supervises one mpv subprocess for the session's duration.
type Process struct {
	cmd        *exec.Cmd
	socketPath string
}

// socketPath returns a per-session Unix socket path in $XDG_RUNTIME_DIR (or
// os.TempDir() as a fallback), unique per process so concurrent sessions
// never collide.
func socketPath() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, fmt.Sprintf("sonora-cli-%d.sock", os.Getpid()))
}

// Start spawns mpv in idle, headless mode and returns once its IPC socket is
// ready to accept commands. Call Close to terminate it and clean up the
// socket, including on panic and signal — a leaked mpv process is a
// user-visible bug (CODESTYLE §5).
func Start() (*Process, error) {
	sp := socketPath()
	_ = os.Remove(sp) // stale socket from a crashed prior run, if any

	cmd := exec.Command("mpv",
		"--no-video",
		"--idle=yes",
		"--input-ipc-server="+sp,
		"--really-quiet",
	)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("player: start mpv: %w", err)
	}

	p := &Process{cmd: cmd, socketPath: sp}

	if err := waitForSocket(sp); err != nil {
		_ = p.Close()
		return nil, fmt.Errorf("player: wait for mpv socket: %w", err)
	}

	return p, nil
}

// SocketPath returns the Unix socket mpv is listening on.
func (p *Process) SocketPath() string {
	return p.socketPath
}

// Close terminates mpv and removes its socket. Safe to call more than once.
func (p *Process) Close() error {
	var err error
	if p.cmd.Process != nil {
		if killErr := p.cmd.Process.Kill(); killErr != nil && killErr != os.ErrProcessDone {
			err = fmt.Errorf("player: kill mpv: %w", killErr)
		}
		_, _ = p.cmd.Process.Wait() // reap; ignore exit status of a killed process
	}
	if rmErr := os.Remove(p.socketPath); rmErr != nil && !os.IsNotExist(rmErr) {
		if err == nil {
			err = fmt.Errorf("player: remove socket: %w", rmErr)
		}
	}
	return err
}
