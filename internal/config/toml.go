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
	defer func() { _ = f.Close() }() // encode error, if any, is returned below

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return err
	}
	return f.Close()
}
