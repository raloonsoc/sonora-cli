// Command sonora is a terminal music client for Sonora and any
// OpenSubsonic-compatible server.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/raloonsoc/sonora-cli/internal/artwork"
	"github.com/raloonsoc/sonora-cli/internal/config"
	"github.com/raloonsoc/sonora-cli/internal/nativeapi"
	"github.com/raloonsoc/sonora-cli/internal/player"
	"github.com/raloonsoc/sonora-cli/internal/subsonic"
	"github.com/raloonsoc/sonora-cli/internal/ui"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "sonora:", err)
		os.Exit(1)
	}
}

func run() error {
	showVersion := flag.Bool("version", false, "print version and exit")
	profile := flag.String("profile", "", "server profile to use")
	configPath := flag.String("config", "", "override the config file location")
	flag.Parse()

	if *showVersion {
		fmt.Println("sonora-cli", version)
		return nil
	}

	if err := checkRuntimeDeps(); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath, *profile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.IsFirstRun() {
		if err := runFirstRunSetup(cfg); err != nil {
			return err
		}
	}

	// Switching profiles restarts the session (new server, new mpv
	// process) rather than hot-swapping state inside one Bubble Tea
	// program: runSession returns the next profile to load, if the user
	// picked one from the in-TUI switcher, and this loop relaunches for it.
	nextProfile := cfg.DefaultProfile
	for {
		switchTo, err := runSession(cfg, nextProfile)
		if err != nil {
			return err
		}
		if switchTo == "" {
			return nil
		}
		nextProfile = switchTo
	}
}

// runSession runs one full TUI session against profileName and returns the
// profile the user switched to, or "" if they quit normally.
func runSession(cfg *config.Config, profileName string) (string, error) {
	cfg.DefaultProfile = profileName
	prof, ok := cfg.ActiveProfile()
	if !ok {
		return "", fmt.Errorf("no profile named %q in config", profileName)
	}

	auth, err := buildAuth(cfg, prof)
	if err != nil {
		return "", fmt.Errorf("build auth: %w", err)
	}
	client := subsonic.NewClient(prof.URL, auth, nil)

	ctrl, err := player.New()
	if err != nil {
		return "", fmt.Errorf("start player: %w", err)
	}
	defer func() { _ = ctrl.Close() }() // graceful shutdown: kill mpv and remove the socket

	if err := ctrl.SetVolume(cfg.Playback.Volume); err != nil {
		return "", fmt.Errorf("set initial volume: %w", err)
	}

	profileNames := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		profileNames = append(profileNames, name)
	}

	app := ui.New(client, ctrl, ui.Options{
		InitialVolume:  cfg.Playback.Volume,
		Term:           artwork.DetectTermType(),
		ArtMode:        artwork.Mode(cfg.UI.Art),
		LyricsEnabled:  cfg.UI.Lyrics,
		ProfileNames:   profileNames,
		CurrentProfile: profileName,
	})
	p := tea.NewProgram(app)

	// SIGINT/SIGTERM: quit the Bubble Tea program so the deferred
	// ctrl.Close() above runs and mpv/the socket are cleaned up, instead of
	// the process dying mid-session with a leaked mpv process.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		p.Quit()
	}()

	m, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("run tui: %w", err)
	}

	return m.(ui.App).SwitchProfile, nil
}

// buildAuth constructs the subsonic.AuthProvider for prof, dispatching on
// its Auth field (SPECS §4.3's AuthProvider swap is meant to be exactly
// this: a config flag, not a rewrite of call sites).
func buildAuth(cfg *config.Config, prof config.Profile) (subsonic.AuthProvider, error) {
	switch prof.Auth {
	case "native":
		return buildNativeAuth(cfg, prof)
	default: // "subsonic", or unset defaulting to it
		return subsonic.NewTokenAuth(prof.Username, prof.Password)
	}
}

// buildNativeAuth logs in with a fresh refresh token if none is stored yet,
// or refreshes an existing one — so the caller always starts the session
// with a live access token (SPECS §4.3: "refreshed transparently before
// each session"). Only the refresh token is persisted afterward; the
// access token stays in memory (nativeapi.BearerAuth).
func buildNativeAuth(cfg *config.Config, prof config.Profile) (subsonic.AuthProvider, error) {
	auth := &nativeapi.BearerAuth{}
	nc := nativeapi.NewClient(prof.URL, auth, nil)
	profileName := cfg.DefaultProfile

	refreshToken, fromKeychain, err := config.LoadRefreshToken(profileName)
	if err != nil {
		return nil, fmt.Errorf("load refresh token: %w", err)
	}
	if !fromKeychain {
		refreshToken = prof.RefreshToken // 0600 config-file fallback
	}

	ctx := context.Background()
	if refreshToken != "" {
		nc.SetRefreshToken(refreshToken)
		if err := nc.Refresh(ctx); err != nil {
			return nil, fmt.Errorf("refresh session: %w", err)
		}
	} else {
		if err := nc.Login(ctx, prof.Username, prof.Password); err != nil {
			return nil, fmt.Errorf("login: %w", err)
		}
	}

	if err := persistRefreshToken(cfg, profileName, prof, nc.RefreshToken()); err != nil {
		return nil, err
	}

	return auth, nil
}

// persistRefreshToken saves the rotated refresh token to the keychain,
// falling back to the 0600 config file — documented, not silent — when no
// keychain is available (SPECS §4.1's storage preference order, reused
// here for the native auth flow).
func persistRefreshToken(cfg *config.Config, profileName string, prof config.Profile, token string) error {
	if err := config.SaveRefreshToken(profileName, token); err == nil {
		return nil
	}

	prof.RefreshToken = token
	cfg.AddProfile(profileName, prof)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("persist refresh token: %w", err)
	}
	return nil
}

// runFirstRunSetup prompts for server URL / username / password and writes
// the resulting profile to cfg's config file (ROADMAP Phase 3).
func runFirstRunSetup(cfg *config.Config) error {
	p := tea.NewProgram(ui.NewFirstRunModel())
	m, err := p.Run()
	if err != nil {
		return fmt.Errorf("first-run setup: %w", err)
	}

	fr := m.(ui.FirstRunModel)
	if fr.Err != nil {
		return fmt.Errorf("first-run setup: %w", fr.Err)
	}
	if fr.Result == nil {
		return fmt.Errorf("first-run setup: incomplete")
	}

	cfg.AddProfile("default", *fr.Result)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}
