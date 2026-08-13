package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_missingFile_returnsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	cfg, err := Load(path, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.IsFirstRun() {
		t.Error("expected IsFirstRun() to be true for a missing config file")
	}
	if cfg.UI.Art != "auto" {
		t.Errorf("UI.Art = %q, want auto", cfg.UI.Art)
	}
	if cfg.Playback.Volume != 80 {
		t.Errorf("Playback.Volume = %d, want 80", cfg.Playback.Volume)
	}
}

func TestSaveLoad_roundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	cfg, err := Load(path, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.AddProfile("home", Profile{
		URL:      "https://music.example.com",
		Username: "raul",
		Password: "hunter2",
		Auth:     "subsonic",
	})

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat saved config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file mode = %o, want 0600", perm)
	}

	got, err := Load(path, "")
	if err != nil {
		t.Fatalf("Load (reload): %v", err)
	}
	if got.IsFirstRun() {
		t.Fatal("reloaded config reports first-run with a saved profile")
	}
	p, ok := got.ActiveProfile()
	if !ok {
		t.Fatal("ActiveProfile() not found after reload")
	}
	if p.URL != "https://music.example.com" || p.Username != "raul" {
		t.Errorf("ActiveProfile() = %+v, want home profile", p)
	}
	if got.DefaultProfile != "home" {
		t.Errorf("DefaultProfile = %q, want home", got.DefaultProfile)
	}
}

func TestLoad_profileOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	cfg, err := Load(path, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.AddProfile("home", Profile{URL: "https://a.example.com", Auth: "subsonic"})
	cfg.AddProfile("vps", Profile{URL: "https://b.example.com", Auth: "native"})
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path, "vps")
	if err != nil {
		t.Fatalf("Load with profile override: %v", err)
	}
	if got.DefaultProfile != "vps" {
		t.Errorf("DefaultProfile = %q, want vps", got.DefaultProfile)
	}
	p, ok := got.ActiveProfile()
	if !ok || p.URL != "https://b.example.com" {
		t.Errorf("ActiveProfile() = %+v, want vps profile", p)
	}
}

func TestAddProfile_firstProfileBecomesDefault(t *testing.T) {
	cfg := &Config{Profiles: map[string]Profile{}}
	cfg.AddProfile("first", Profile{URL: "https://a.example.com"})
	cfg.AddProfile("second", Profile{URL: "https://b.example.com"})

	if cfg.DefaultProfile != "first" {
		t.Errorf("DefaultProfile = %q, want first", cfg.DefaultProfile)
	}
}

func TestDefaultPath_underUserConfigDir(t *testing.T) {
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if filepath.Base(path) != "config.toml" {
		t.Errorf("DefaultPath() = %q, want to end in config.toml", path)
	}
	if filepath.Base(filepath.Dir(path)) != "sonora-cli" {
		t.Errorf("DefaultPath() = %q, want parent dir sonora-cli", path)
	}
}
