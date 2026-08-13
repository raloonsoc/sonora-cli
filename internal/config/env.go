package config

import (
	"os"
	"strconv"
)

// applyEnvOverrides applies SONORA_CLI_* environment variables on top of
// the loaded config, per README's "Every setting can also be overridden"
// note. Only variables that are actually set are applied — an unset env
// var never clobbers a value loaded from the config file.
//
// Profile-scoped variables (URL/USERNAME/PASSWORD/AUTH) apply to whichever
// profile ends up active (DefaultProfile after this function runs), since
// overriding "the profile in use" is the only case an env var can express
// without a way to name which profile it targets.
func (c *Config) applyEnvOverrides() {
	if v, ok := os.LookupEnv("SONORA_CLI_DEFAULT_PROFILE"); ok {
		c.DefaultProfile = v
	}
	if v, ok := os.LookupEnv("SONORA_CLI_UI_ART"); ok {
		c.UI.Art = v
	}
	if v, ok := os.LookupEnv("SONORA_CLI_UI_LYRICS"); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			c.UI.Lyrics = b
		}
	}
	if v, ok := os.LookupEnv("SONORA_CLI_PLAYBACK_VOLUME"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			c.Playback.Volume = n
		}
	}

	c.applyProfileEnvOverrides()
}

func (c *Config) applyProfileEnvOverrides() {
	url, hasURL := os.LookupEnv("SONORA_CLI_URL")
	username, hasUsername := os.LookupEnv("SONORA_CLI_USERNAME")
	password, hasPassword := os.LookupEnv("SONORA_CLI_PASSWORD")
	auth, hasAuth := os.LookupEnv("SONORA_CLI_AUTH")

	if !hasURL && !hasUsername && !hasPassword && !hasAuth {
		return
	}

	if c.DefaultProfile == "" {
		c.DefaultProfile = "env"
	}
	prof := c.Profiles[c.DefaultProfile]

	if hasURL {
		prof.URL = url
	}
	if hasUsername {
		prof.Username = username
	}
	if hasPassword {
		prof.Password = password
	}
	if hasAuth {
		prof.Auth = auth
	}

	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	c.Profiles[c.DefaultProfile] = prof
}
