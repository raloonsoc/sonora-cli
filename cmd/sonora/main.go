// Command sonora is a terminal music client for Sonora and any
// OpenSubsonic-compatible server.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/raloonsoc/sonora-cli/internal/config"
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

	_, err := config.Load(*configPath, *profile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	fmt.Println("sonora-cli", version, "— TUI not yet implemented")
	return nil
}
