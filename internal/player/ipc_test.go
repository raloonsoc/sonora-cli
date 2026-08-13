//go:build !windows

package player

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// shortSocketDir returns a temp directory outside t.TempDir(), whose nested
// per-test path routinely exceeds the ~104-byte sun_path limit on Unix
// domain sockets (macOS in particular).
func shortSocketDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "sonora-ipc")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// fakeMPV listens on a Unix socket and answers every request with a
// canned response, echoing the request_id back — enough to exercise the
// framing and correlation logic without a real mpv binary.
func fakeMPV(t *testing.T, handle func(req ipcRequest) ipcResponse) string {
	t.Helper()

	sock := filepath.Join(shortSocketDir(t), "mpv.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			var req ipcRequest
			if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
				continue
			}
			resp := handle(req)
			resp.RequestID = req.RequestID
			line, _ := json.Marshal(resp)
			line = append(line, '\n')
			_, _ = conn.Write(line)
		}
	}()

	return sock
}

func TestIPCConn_call_success(t *testing.T) {
	sock := fakeMPV(t, func(req ipcRequest) ipcResponse {
		return ipcResponse{Error: "success", Data: json.RawMessage(`42.5`)}
	})

	conn, err := newIPCConn(sock)
	if err != nil {
		t.Fatalf("newIPCConn: %v", err)
	}
	defer func() { _ = conn.Close() }()

	data, err := conn.call([]any{"get_property", "time-pos"})
	if err != nil {
		t.Fatalf("call() err: %v", err)
	}

	var pos float64
	if err := json.Unmarshal(data, &pos); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pos != 42.5 {
		t.Errorf("pos = %v, want 42.5", pos)
	}
}

func TestIPCConn_call_error(t *testing.T) {
	sock := fakeMPV(t, func(req ipcRequest) ipcResponse {
		return ipcResponse{Error: "some other mpv error"}
	})

	conn, err := newIPCConn(sock)
	if err != nil {
		t.Fatalf("newIPCConn: %v", err)
	}
	defer func() { _ = conn.Close() }()

	_, err = conn.call([]any{"get_property", "time-pos"})
	if err == nil {
		t.Fatal("call() expected error, got nil")
	}
	if errors.Is(err, ErrPropertyUnavailable) {
		t.Error("a generic mpv error should not match ErrPropertyUnavailable")
	}
}

func TestIPCConn_call_propertyUnavailable(t *testing.T) {
	sock := fakeMPV(t, func(req ipcRequest) ipcResponse {
		return ipcResponse{Error: "property unavailable"}
	})

	conn, err := newIPCConn(sock)
	if err != nil {
		t.Fatalf("newIPCConn: %v", err)
	}
	defer func() { _ = conn.Close() }()

	_, err = conn.call([]any{"get_property", "time-pos"})
	if !errors.Is(err, ErrPropertyUnavailable) {
		t.Fatalf("call() err = %v, want ErrPropertyUnavailable", err)
	}
}

func TestIPCConn_concurrentCalls_correlated(t *testing.T) {
	sock := fakeMPV(t, func(req ipcRequest) ipcResponse {
		// Echo the command name back as data so the test can verify each
		// caller received its own response, not another's.
		b, _ := json.Marshal(req.Command[0])
		return ipcResponse{Error: "success", Data: b}
	})

	conn, err := newIPCConn(sock)
	if err != nil {
		t.Fatalf("newIPCConn: %v", err)
	}
	defer func() { _ = conn.Close() }()

	type result struct {
		cmd string
		got string
	}
	results := make(chan result, 4)
	cmds := []string{"a", "b", "c", "d"}

	for _, cmd := range cmds {
		cmd := cmd
		go func() {
			data, err := conn.call([]any{cmd})
			if err != nil {
				t.Errorf("call(%s): %v", cmd, err)
				return
			}
			var got string
			_ = json.Unmarshal(data, &got)
			results <- result{cmd: cmd, got: got}
		}()
	}

	for range cmds {
		select {
		case r := <-results:
			if r.cmd != r.got {
				t.Errorf("got response %q for request %q", r.got, r.cmd)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for concurrent calls")
		}
	}
}

func TestWaitForSocket_timesOutOnMissingSocket(t *testing.T) {
	// Point at a path nothing ever listens on and shrink the wait so the
	// test doesn't eat the real 5s budget.
	orig := waitForSocketTimeout
	waitForSocketTimeout = 100 * time.Millisecond
	defer func() { waitForSocketTimeout = orig }()

	missing := filepath.Join(shortSocketDir(t), "nope.sock")
	if err := waitForSocket(missing); err == nil {
		t.Fatal("expected error for a socket that never appears")
	}
}

func TestWaitForSocket_succeedsOnceListening(t *testing.T) {
	sock := filepath.Join(shortSocketDir(t), "ready.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	if err := waitForSocket(sock); err != nil {
		t.Fatalf("waitForSocket: %v", err)
	}
}

func TestSocketPath_includesPID(t *testing.T) {
	p := socketPath()
	if !filepath.IsAbs(p) {
		t.Errorf("socketPath() = %q, want absolute path", p)
	}
	if _, err := os.Stat(filepath.Dir(p)); err != nil {
		t.Errorf("socket dir does not exist: %v", err)
	}
}
