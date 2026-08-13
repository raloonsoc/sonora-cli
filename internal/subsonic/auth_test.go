package subsonic

import (
	"crypto/md5"
	"encoding/hex"
	"testing"
)

func TestTokenAuth_AuthParams(t *testing.T) {
	auth, err := NewTokenAuth("alice", "hunter2")
	if err != nil {
		t.Fatalf("NewTokenAuth: %v", err)
	}

	params := auth.AuthParams()
	salt := params.Get("s")
	if salt == "" {
		t.Fatal("salt is empty")
	}

	sum := md5.Sum([]byte("hunter2" + salt))
	want := hex.EncodeToString(sum[:])
	if got := params.Get("t"); got != want {
		t.Errorf("token = %q, want %q", got, want)
	}
	if got := params.Get("u"); got != "alice" {
		t.Errorf("username = %q, want alice", got)
	}
}

func TestTokenAuth_saltIsRandom(t *testing.T) {
	a1, err := NewTokenAuth("alice", "hunter2")
	if err != nil {
		t.Fatalf("NewTokenAuth: %v", err)
	}
	a2, err := NewTokenAuth("alice", "hunter2")
	if err != nil {
		t.Fatalf("NewTokenAuth: %v", err)
	}

	if a1.AuthParams().Get("s") == a2.AuthParams().Get("s") {
		t.Error("two sessions produced the same salt")
	}
}

func TestTokenAuth_AuthHeader_isNil(t *testing.T) {
	auth, err := NewTokenAuth("alice", "hunter2")
	if err != nil {
		t.Fatalf("NewTokenAuth: %v", err)
	}
	if h := auth.AuthHeader(); h != nil {
		t.Errorf("AuthHeader() = %v, want nil", h)
	}
}
