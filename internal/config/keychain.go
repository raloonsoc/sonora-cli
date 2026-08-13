package config

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// keychainService namespaces sonora-cli's entries in the OS keychain from
// every other application using it.
const keychainService = "sonora-cli"

// SavePassword stores password in the OS keychain under profile, when a
// keychain is available (SPECS §4.1 preference 1). Callers should fall
// back to storing the password in the 0600 config file when this returns a
// non-nil error — that fallback is documented, not silent, per README's
// "Password storage" section.
func SavePassword(profile, password string) error {
	if err := keyring.Set(keychainService, profile, password); err != nil {
		return fmt.Errorf("config: save password to keychain: %w", err)
	}
	return nil
}

// LoadPassword retrieves a previously saved password from the OS keychain
// for profile. ok is false when no keychain entry exists (not an error —
// the caller should fall back to Profile.Password from the config file).
func LoadPassword(profile string) (password string, ok bool, err error) {
	password, err = keyring.Get(keychainService, profile)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("config: load password from keychain: %w", err)
	}
	return password, true, nil
}

// DeletePassword removes profile's keychain entry, e.g. once a profile
// migrates to native auth and no longer needs a stored password.
func DeletePassword(profile string) error {
	err := keyring.Delete(keychainService, profile)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("config: delete password from keychain: %w", err)
	}
	return nil
}

// refreshTokenKey namespaces a profile's native-API refresh token
// separately from its Subsonic password within the same keychain service,
// so a profile can hold both during an auth-mode migration.
func refreshTokenKey(profile string) string {
	return profile + ":refresh_token"
}

// SaveRefreshToken stores a profile's native-API refresh token in the OS
// keychain (SPECS §4.3: only the refresh token is persisted, never the
// access token). Falls back to the 0600 config file on error, same as
// SavePassword.
func SaveRefreshToken(profile, token string) error {
	if err := keyring.Set(keychainService, refreshTokenKey(profile), token); err != nil {
		return fmt.Errorf("config: save refresh token to keychain: %w", err)
	}
	return nil
}

// LoadRefreshToken retrieves a previously saved refresh token for profile.
// ok is false when no keychain entry exists.
func LoadRefreshToken(profile string) (token string, ok bool, err error) {
	token, err = keyring.Get(keychainService, refreshTokenKey(profile))
	if errors.Is(err, keyring.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("config: load refresh token from keychain: %w", err)
	}
	return token, true, nil
}

// DeleteRefreshToken removes profile's stored refresh token, e.g. on
// logout or when a profile migrates back to Subsonic auth.
func DeleteRefreshToken(profile string) error {
	err := keyring.Delete(keychainService, refreshTokenKey(profile))
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("config: delete refresh token from keychain: %w", err)
	}
	return nil
}
