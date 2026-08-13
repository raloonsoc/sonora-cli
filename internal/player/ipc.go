//go:build !windows

package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// waitForSocketTimeout bounds how long waitForSocket polls before giving
// up. A var, not a const, so tests can shrink it instead of eating the real
// budget.
var waitForSocketTimeout = 5 * time.Second

// waitForSocket polls until mpv's IPC socket accepts a connection or the
// deadline elapses. mpv creates the socket asynchronously after --idle
// startup, so a fixed sleep would be either too slow or flaky.
func waitForSocket(path string) error {
	deadline := time.Now().Add(waitForSocketTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.Dial("unix", path)
		if err == nil {
			return conn.Close()
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("socket not ready after %s: %w", waitForSocketTimeout, lastErr)
}

// ipcRequest is one mpv IPC command, correlated to its response by
// request_id.
type ipcRequest struct {
	Command   []any `json:"command"`
	RequestID int64 `json:"request_id"`
}

// ipcResponse is mpv's reply to an ipcRequest, or an unsolicited event.
type ipcResponse struct {
	RequestID int64           `json:"request_id"`
	Error     string          `json:"error"`
	Data      json.RawMessage `json:"data"`
	Event     string          `json:"event,omitempty"`
}

// ipcConn is a single connection to mpv's IPC socket, dispatching responses
// to the goroutine that issued the matching request and events to a
// separate channel.
type ipcConn struct {
	conn   net.Conn
	writeM sync.Mutex

	pendingM sync.Mutex
	pending  map[int64]chan ipcResponse
	nextID   int64

	events chan ipcResponse
	done   chan struct{}
}

func newIPCConn(path string) (*ipcConn, error) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, fmt.Errorf("player: dial mpv socket: %w", err)
	}

	c := &ipcConn{
		conn:    conn,
		pending: make(map[int64]chan ipcResponse),
		events:  make(chan ipcResponse, 16),
		done:    make(chan struct{}),
	}
	go c.readLoop()
	return c, nil
}

func (c *ipcConn) readLoop() {
	defer close(c.events)
	scanner := bufio.NewScanner(c.conn)
	for scanner.Scan() {
		var resp ipcResponse
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			continue // malformed line from mpv; skip rather than crash the reader
		}

		if resp.Event != "" {
			select {
			case c.events <- resp:
			case <-c.done:
				return
			default:
				// Event channel full: drop rather than block the reader.
				// Position polling re-derives state on the next tick, so a
				// dropped event is not fatal (SPECS §5.3).
			}
			continue
		}

		c.pendingM.Lock()
		ch, ok := c.pending[resp.RequestID]
		if ok {
			delete(c.pending, resp.RequestID)
		}
		c.pendingM.Unlock()

		if ok {
			ch <- resp
		}
	}
}

// call issues command and blocks for mpv's response.
func (c *ipcConn) call(command []any) (json.RawMessage, error) {
	c.pendingM.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan ipcResponse, 1)
	c.pending[id] = ch
	c.pendingM.Unlock()

	req := ipcRequest{Command: command, RequestID: id}
	line, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("player: encode ipc request: %w", err)
	}
	line = append(line, '\n')

	c.writeM.Lock()
	_, err = c.conn.Write(line)
	c.writeM.Unlock()
	if err != nil {
		return nil, fmt.Errorf("player: write ipc request: %w", err)
	}

	resp := <-ch
	if resp.Error != "" && resp.Error != "success" {
		return nil, fmt.Errorf("player: mpv error: %s", resp.Error)
	}
	return resp.Data, nil
}

func (c *ipcConn) Close() error {
	close(c.done)
	return c.conn.Close()
}
