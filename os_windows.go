//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func defaultWindscribeCLIPath() string {
	candidates := []string{
		`C:\Program Files\Windscribe\windscribe-cli.exe`,
		`C:\Program Files (x86)\Windscribe\windscribe-cli.exe`,
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("windscribe-cli.exe"); err == nil {
		return p
	}
	return `C:\Program Files\Windscribe\windscribe-cli.exe`
}

func prepareCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func openFileInOS(filePath string) error {
	cmd := exec.Command("cmd.exe", "/c", "start", "", filePath)
	prepareCmd(cmd)
	return cmd.Start()
}
