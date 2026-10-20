//go:build darwin

package main

import (
	"os"
	"os/exec"
)

func defaultWindscribeCLIPath() string {
	candidates := []string{
		"/Applications/Windscribe.app/Contents/MacOS/windscribe-cli",
		"/usr/local/bin/windscribe-cli",
		"/opt/homebrew/bin/windscribe-cli",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("windscribe-cli"); err == nil {
		return p
	}
	return "/Applications/Windscribe.app/Contents/MacOS/windscribe-cli"
}

func prepareCmd(cmd *exec.Cmd) {
	// No window-hiding needed on macOS
}

func openFileInOS(filePath string) error {
	cmd := exec.Command("open", filePath)
	return cmd.Start()
}
