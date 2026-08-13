package config

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	keyring.MockInit()
	m.Run()
}

func TestSaveLoadDeletePassword(t *testing.T) {
	if err := SavePassword("home", "hunter2"); err != nil {
		t.Fatalf("SavePassword: %v", err)
	}

	got, ok, err := LoadPassword("home")
	if err != nil {
		t.Fatalf("LoadPassword: %v", err)
	}
	if !ok || got != "hunter2" {
		t.Fatalf("LoadPassword() = %q, %v, want hunter2, true", got, ok)
	}

	if err := DeletePassword("home"); err != nil {
		t.Fatalf("DeletePassword: %v", err)
	}
	if _, ok, err := LoadPassword("home"); err != nil || ok {
		t.Fatalf("LoadPassword() after delete = ok=%v err=%v, want ok=false", ok, err)
	}
}

func TestLoadPassword_missingEntryIsNotAnError(t *testing.T) {
	_, ok, err := LoadPassword("nonexistent")
	if err != nil {
		t.Fatalf("LoadPassword: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for a profile with no stored password")
	}
}

func TestSaveLoadDeleteRefreshToken(t *testing.T) {
	if err := SaveRefreshToken("vps", "rt-abc123"); err != nil {
		t.Fatalf("SaveRefreshToken: %v", err)
	}

	got, ok, err := LoadRefreshToken("vps")
	if err != nil {
		t.Fatalf("LoadRefreshToken: %v", err)
	}
	if !ok || got != "rt-abc123" {
		t.Fatalf("LoadRefreshToken() = %q, %v, want rt-abc123, true", got, ok)
	}

	if err := DeleteRefreshToken("vps"); err != nil {
		t.Fatalf("DeleteRefreshToken: %v", err)
	}
	if _, ok, err := LoadRefreshToken("vps"); err != nil || ok {
		t.Fatalf("LoadRefreshToken() after delete = ok=%v err=%v, want ok=false", ok, err)
	}
}

func TestPasswordAndRefreshTokenKeysDoNotCollide(t *testing.T) {
	if err := SavePassword("shared", "pw-value"); err != nil {
		t.Fatalf("SavePassword: %v", err)
	}
	if err := SaveRefreshToken("shared", "rt-value"); err != nil {
		t.Fatalf("SaveRefreshToken: %v", err)
	}

	pw, _, err := LoadPassword("shared")
	if err != nil {
		t.Fatalf("LoadPassword: %v", err)
	}
	rt, _, err := LoadRefreshToken("shared")
	if err != nil {
		t.Fatalf("LoadRefreshToken: %v", err)
	}

	if pw != "pw-value" || rt != "rt-value" {
		t.Errorf("password/refresh token collided: pw=%q rt=%q", pw, rt)
	}
}
