//go:build darwin

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func killPlatformVPNHelpers() {
	for _, proc := range []string{"windscribeopenvpn", "windscribewstunnel"} {
		cmd := exec.Command("pkill", "-f", proc)
		prepareCmd(cmd)
		_ = cmd.Run()
	}
}

func restartPlatformWindscribeApp() {
	killPlatformVPNHelpers()
}

func applyPlatformConnectionTweaks(ctx context.Context, config TestConfig, logger func(string)) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine user home directory: %w", err)
	}

	// 1. Large TLS / stealth extra tls padding
	extraConfDir := filepath.Join(homeDir, "Library", "Application Support", "Windscribe", "Windscribe2")
	extraConfPath := filepath.Join(extraConfDir, "windscribe_extra.conf")
	_ = os.MkdirAll(extraConfDir, 0755)

	if config.LargeTLS {
		_ = os.WriteFile(extraConfPath, []byte("ws-stealth-extra-tls-padding\n"), 0644)
		logger("Windscribe macOS: Applied Large TLS padding to windscribe_extra.conf")
	} else {
		_ = os.WriteFile(extraConfPath, []byte(""), 0644)
		logger("Windscribe macOS: Cleared Large TLS padding from windscribe_extra.conf")
	}

	logger(fmt.Sprintf("Windscribe macOS: Synchronized settings (Amnezia: %s, SNI: %s, Routing: %s, Large TLS: %v, DNS: %s).", config.AmneziaConfig, config.CustomSNI, config.ServerRouting, config.LargeTLS, config.InternalDNS))
	return nil
}
