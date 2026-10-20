//go:build linux

package main

import (
	"os"
	"os/exec"
)

func defaultWindscribeCLIPath() string {
	candidates := []string{
		"/usr/bin/windscribe-cli",
		"/usr/local/bin/windscribe-cli",
		"/bin/windscribe-cli",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("windscribe-cli"); err == nil {
		return p
	}
	return "windscribe-cli"
}

func prepareCmd(cmd *exec.Cmd) {
	// No window-hiding needed on Linux
}

func openFileInOS(filePath string) error {
	cmd := exec.Command("xdg-open", filePath)
	return cmd.Start()
}
