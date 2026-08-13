package config

import (
	"path/filepath"
	"testing"
)

func TestLoad_envOverrides(t *testing.T) {
	t.Setenv("SONORA_CLI_DEFAULT_PROFILE", "envprofile")
	t.Setenv("SONORA_CLI_UI_ART", "ascii")
	t.Setenv("SONORA_CLI_UI_LYRICS", "false")
	t.Setenv("SONORA_CLI_PLAYBACK_VOLUME", "42")

	cfg, err := Load(filepath.Join(t.TempDir(), "config.toml"), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.DefaultProfile != "envprofile" {
		t.Errorf("DefaultProfile = %q, want envprofile", cfg.DefaultProfile)
	}
	if cfg.UI.Art != "ascii" {
		t.Errorf("UI.Art = %q, want ascii", cfg.UI.Art)
	}
	if cfg.UI.Lyrics != false {
		t.Errorf("UI.Lyrics = %v, want false", cfg.UI.Lyrics)
	}
	if cfg.Playback.Volume != 42 {
		t.Errorf("Playback.Volume = %d, want 42", cfg.Playback.Volume)
	}
}

func TestLoad_envOverrides_profileFields(t *testing.T) {
	t.Setenv("SONORA_CLI_URL", "https://env.example.com")
	t.Setenv("SONORA_CLI_USERNAME", "envuser")
	t.Setenv("SONORA_CLI_AUTH", "native")

	path := filepath.Join(t.TempDir(), "config.toml")
	cfg, err := Load(path, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.AddProfile("home", Profile{URL: "https://file.example.com", Username: "fileuser", Auth: "subsonic"})
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload: env vars should override whatever profile ends up active,
	// here the one persisted to disk (env doesn't set DEFAULT_PROFILE this
	// time, so the file's default_profile wins the name, but its fields
	// get overridden).
	cfg2, err := Load(path, "")
	if err != nil {
		t.Fatalf("Load (reload): %v", err)
	}
	p, ok := cfg2.ActiveProfile()
	if !ok {
		t.Fatal("ActiveProfile() not found")
	}
	if p.URL != "https://env.example.com" {
		t.Errorf("URL = %q, want env override", p.URL)
	}
	if p.Username != "envuser" {
		t.Errorf("Username = %q, want env override", p.Username)
	}
	if p.Auth != "native" {
		t.Errorf("Auth = %q, want env override", p.Auth)
	}
}

func TestLoad_noEnvVarsLeavesConfigUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg, err := Load(path, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.UI.Art != "auto" {
		t.Errorf("UI.Art = %q, want default auto with no env vars set", cfg.UI.Art)
	}
}

func TestLoad_missingFileStillHonorsProfileFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg, err := Load(path, "vps")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultProfile != "vps" {
		t.Errorf("DefaultProfile = %q, want vps even though the config file doesn't exist", cfg.DefaultProfile)
	}
}
