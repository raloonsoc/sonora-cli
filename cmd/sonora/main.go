// Command sonora is a terminal music client for Sonora and any
// OpenSubsonic-compatible server.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/raloonsoc/sonora-cli/internal/artwork"
	"github.com/raloonsoc/sonora-cli/internal/config"
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

	prof, ok := cfg.ActiveProfile()
	if !ok {
		return fmt.Errorf("no profile named %q in config", cfg.DefaultProfile)
	}

	auth, err := subsonic.NewTokenAuth(prof.Username, prof.Password)
	if err != nil {
		return fmt.Errorf("build auth: %w", err)
	}
	client := subsonic.NewClient(prof.URL, auth, nil)

	ctrl, err := player.New()
	if err != nil {
		return fmt.Errorf("start player: %w", err)
	}
	defer func() { _ = ctrl.Close() }() // graceful shutdown: kill mpv and remove the socket

	if err := ctrl.SetVolume(cfg.Playback.Volume); err != nil {
		return fmt.Errorf("set initial volume: %w", err)
	}

	app := ui.New(client, ctrl, ui.Options{
		InitialVolume: cfg.Playback.Volume,
		Term:          artwork.DetectTermType(),
		ArtMode:       artwork.Mode(cfg.UI.Art),
		LyricsEnabled: cfg.UI.Lyrics,
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

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
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
