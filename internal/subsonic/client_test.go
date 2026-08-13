package subsonic

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	auth, err := NewTokenAuth("alice", "hunter2")
	if err != nil {
		t.Fatalf("NewTokenAuth: %v", err)
	}
	return NewClient(srv.URL, auth, srv.Client())
}

func TestPing(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		status  int
		wantErr error
	}{
		{
			name:   "ok",
			body:   `{"subsonic-response":{"status":"ok","version":"1.16.1"}}`,
			status: http.StatusOK,
		},
		{
			name:    "bad credentials",
			body:    `{"subsonic-response":{"status":"failed","version":"1.16.1","error":{"code":40,"message":"Wrong username or password"}}}`,
			status:  http.StatusOK,
			wantErr: ErrUnauthorized,
		},
		{
			name:    "not found",
			body:    `{"subsonic-response":{"status":"failed","version":"1.16.1","error":{"code":70,"message":"Data not found"}}}`,
			status:  http.StatusOK,
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/rest/ping.view") {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				q := r.URL.Query()
				for _, want := range []string{"u", "t", "s", "v", "c", "f"} {
					if q.Get(want) == "" {
						t.Errorf("missing query param %q", want)
					}
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body)) // test server; write failure would fail the read side
			})

			err := c.Ping(context.Background())
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Ping() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Ping() unexpected err: %v", err)
			}
		})
	}
}

func TestGetArtists(t *testing.T) {
	body := `{"subsonic-response":{"status":"ok","version":"1.16.1","artists":{"index":[{"name":"K","artist":[{"id":"1","name":"Kikagaku Moyo","albumCount":3}]}]}}}`

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body)) // test server; write failure would fail the read side
	})

	got, err := c.GetArtists(context.Background())
	if err != nil {
		t.Fatalf("GetArtists() err: %v", err)
	}
	if len(got.Index) != 1 || len(got.Index[0].Artists) != 1 {
		t.Fatalf("GetArtists() = %+v, want one index entry with one artist", got)
	}
	if got.Index[0].Artists[0].Name != "Kikagaku Moyo" {
		t.Errorf("artist name = %q, want Kikagaku Moyo", got.Index[0].Artists[0].Name)
	}
}

func TestGetAlbum_accentColor(t *testing.T) {
	body := `{"subsonic-response":{"status":"ok","version":"1.16.1","album":{"id":"42","name":"Masana Temples","artist":"Kikagaku Moyo","artistId":"1","accentColor":"#c96a3f"}}}`

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body)) // test server; write failure would fail the read side
	})

	got, err := c.GetAlbum(context.Background(), "42")
	if err != nil {
		t.Fatalf("GetAlbum() err: %v", err)
	}
	if got.AccentColor != "#c96a3f" {
		t.Errorf("AccentColor = %q, want #c96a3f", got.AccentColor)
	}
}

func TestStreamURL(t *testing.T) {
	auth, err := NewTokenAuth("alice", "hunter2")
	if err != nil {
		t.Fatalf("NewTokenAuth: %v", err)
	}
	c := NewClient("https://music.example.com", auth, nil)

	got := c.StreamURL("123")
	if !strings.HasPrefix(got, "https://music.example.com/rest/stream.view?") {
		t.Errorf("StreamURL() = %q, unexpected prefix", got)
	}
	if !strings.Contains(got, "id=123") {
		t.Errorf("StreamURL() = %q, missing id param", got)
	}
	if !strings.Contains(got, "t=") || !strings.Contains(got, "s=") {
		t.Errorf("StreamURL() = %q, missing auth params", got)
	}
}

func TestScrobble(t *testing.T) {
	var gotSubmission string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotSubmission = r.URL.Query().Get("submission")
		_, _ = w.Write([]byte(`{"subsonic-response":{"status":"ok","version":"1.16.1"}}`)) // test server; write failure would fail the read side
	})

	if err := c.Scrobble(context.Background(), "1", true); err != nil {
		t.Fatalf("Scrobble() err: %v", err)
	}
	if gotSubmission != "true" {
		t.Errorf("submission param = %q, want true", gotSubmission)
	}
}
