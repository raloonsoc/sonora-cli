package main

import (
	"fmt"
	"os/exec"
)

// checkRuntimeDeps verifies that mpv and ffmpeg are reachable on $PATH.
// Both are required at runtime: mpv drives playback, ffmpeg backs its media
// handling. Failing fast here beats a confusing failure deep in playback.
func checkRuntimeDeps() error {
	for _, bin := range []string{"mpv", "ffmpeg"} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("%s not found on $PATH: install it and try again", bin)
		}
	}
	return nil
}
