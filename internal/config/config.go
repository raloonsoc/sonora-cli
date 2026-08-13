// Package config loads and saves sonora-cli's TOML configuration file.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Profile is one server connection: its URL, credentials, and auth mode.
type Profile struct {
	URL      string `toml:"url"`
	Username string `toml:"username"`
	Password string `toml:"password,omitempty"`
	Auth     string `toml:"auth"` // "subsonic" | "native"
}

// UI holds display preferences.
type UI struct {
	Art    string `toml:"art"` // "auto" | "ascii" | "off"
	Lyrics bool   `toml:"lyrics"`
}

// Playback holds playback preferences.
type Playback struct {
	Volume int `toml:"volume"`
}

// Config is the root of ~/.config/sonora-cli/config.toml.
type Config struct {
	DefaultProfile string             `toml:"default_profile"`
	Profiles       map[string]Profile `toml:"profiles"`
	UI             UI                 `toml:"ui"`
	Playback       Playback           `toml:"playback"`

	// path is where this Config was loaded from, used by Save.
	path string
}

// DefaultPath returns ~/.config/sonora-cli/config.toml, respecting
// os.UserConfigDir() for cross-platform correctness.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "sonora-cli", "config.toml"), nil
}

// Load reads the config file at path (or the default path if empty) and
// selects profile (or the file's default_profile if empty). A missing file
// is not an error: it yields an empty Config ready for the first-run flow.
func Load(path, profile string) (*Config, error) {
	if path == "" {
		p, err := DefaultPath()
		if err != nil {
			return nil, err
		}
		path = p
	}

	cfg := &Config{
		Profiles: map[string]Profile{},
		path:     path,
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if err := decodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("config: decode %s: %w", path, err)
	}
	cfg.path = path

	if profile != "" {
		cfg.DefaultProfile = profile
	}

	return cfg, nil
}

// Save writes cfg back to its source path with 0600 permissions —
// profiles may carry a plaintext password (see SPECS §4.1).
func (c *Config) Save() error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return fmt.Errorf("config: create config dir: %w", err)
	}
	if err := encodeFile(c.path, c); err != nil {
		return fmt.Errorf("config: write %s: %w", c.path, err)
	}
	return nil
}

// ActiveProfile returns the profile selected by DefaultProfile.
func (c *Config) ActiveProfile() (Profile, bool) {
	p, ok := c.Profiles[c.DefaultProfile]
	return p, ok
}
