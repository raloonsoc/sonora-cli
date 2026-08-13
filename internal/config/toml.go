package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

func decodeFile(path string, cfg *Config) error {
	_, err := toml.DecodeFile(path, cfg)
	return err
}

func encodeFile(path string, cfg *Config) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(cfg)
}
