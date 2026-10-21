//go:build linux

package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	extraConfDir := filepath.Join(homeDir, ".config", "Windscribe", "Windscribe2")
	extraConfPath := filepath.Join(extraConfDir, "windscribe_extra.conf")
	_ = os.MkdirAll(extraConfDir, 0755)

	if config.LargeTLS {
		_ = os.WriteFile(extraConfPath, []byte("ws-stealth-extra-tls-padding\n"), 0644)
		logger("Windscribe Linux: Applied Large TLS padding to windscribe_extra.conf")
	} else {
		_ = os.WriteFile(extraConfPath, []byte(""), 0644)
		logger("Windscribe Linux: Cleared Large TLS padding from windscribe_extra.conf")
	}

	// 2. Update ~/.config/Windscribe/windscribe_cli.conf
	confDir := filepath.Join(homeDir, ".config", "Windscribe")
	confPath := filepath.Join(confDir, "windscribe_cli.conf")
	_ = os.MkdirAll(confDir, 0755)

	targetAmnezia := strings.TrimSpace(config.AmneziaConfig)
	targetTweaks := "1"
	targetCircumvent := "false"
	if config.LargeTLS || config.ProtocolTweaks {
		targetCircumvent = "true"
	}
	if !config.ProtocolTweaks || targetAmnezia == "None (Disabled)" || targetAmnezia == "Disabled" || targetAmnezia == "None" {
		targetTweaks = "2"
		targetAmnezia = ""
	}

	targetRouting := "0"
	switch strings.ToLower(strings.TrimSpace(config.ServerRouting)) {
	case "regular":
		targetRouting = "1"
	case "alternate", "alternative":
		targetRouting = "2"
	default:
		targetRouting = "0"
	}

	targetSni := strings.TrimSpace(config.CustomSNI)
	if targetSni == "-" {
		targetSni = ""
	}

	targetDnsPolicy := "Google"
	switch strings.ToLower(strings.TrimSpace(config.InternalDNS)) {
	case "os default", "osdefault":
		targetDnsPolicy = "OS Default"
	case "open dns", "opendns":
		targetDnsPolicy = "Open DNS"
	case "cloudflare":
		targetDnsPolicy = "Cloudflare"
	case "google":
		targetDnsPolicy = "Google"
	case "control d", "controld":
		targetDnsPolicy = "ControlD"
	}

	// Read existing conf if present
	var lines []string
	if data, err := os.ReadFile(confPath); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
	}

	// Sections to manage
	connSettings := map[string]string{
		"CircumventCensorship": targetCircumvent,
		"ProtocolTweaksMethod": targetTweaks,
		"AmneziawgPreset":      targetAmnezia,
		"CustomSniDomain":      targetSni,
		"ServerRoutingMethod":  targetRouting,
	}
	advSettings := map[string]string{
		"DNSPolicy": targetDnsPolicy,
	}

	appliedConn := make(map[string]bool)
	appliedAdv := make(map[string]bool)

	var newLines []string
	currentSection := ""

	flushSection := func(sec string) {
		if strings.EqualFold(sec, "[Connection]") {
			for k, v := range connSettings {
				if !appliedConn[k] {
					newLines = append(newLines, fmt.Sprintf("%s=%s", k, v))
					appliedConn[k] = true
				}
			}
		} else if strings.EqualFold(sec, "[Advanced]") {
			for k, v := range advSettings {
				if !appliedAdv[k] {
					newLines = append(newLines, fmt.Sprintf("%s=%s", k, v))
					appliedAdv[k] = true
				}
			}
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			flushSection(currentSection)
			currentSection = trimmed
			newLines = append(newLines, line)
			continue
		}

		if strings.EqualFold(currentSection, "[Connection]") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				for targetKey, val := range connSettings {
					if strings.EqualFold(key, targetKey) {
						line = fmt.Sprintf("%s=%s", targetKey, val)
						appliedConn[targetKey] = true
						break
					}
				}
			}
		} else if strings.EqualFold(currentSection, "[Advanced]") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				for targetKey, val := range advSettings {
					if strings.EqualFold(key, targetKey) {
						line = fmt.Sprintf("%s=%s", targetKey, val)
						appliedAdv[targetKey] = true
						break
					}
				}
			}
		}

		newLines = append(newLines, line)
	}

	flushSection(currentSection)

	// If sections did not exist, append them
	if len(appliedConn) < len(connSettings) {
		newLines = append(newLines, "", "[Connection]")
		for k, v := range connSettings {
			if !appliedConn[k] {
				newLines = append(newLines, fmt.Sprintf("%s=%s", k, v))
				appliedConn[k] = true
			}
		}
	}
	if len(appliedAdv) < len(advSettings) {
		newLines = append(newLines, "", "[Advanced]")
		for k, v := range advSettings {
			if !appliedAdv[k] {
				newLines = append(newLines, fmt.Sprintf("%s=%s", k, v))
				appliedAdv[k] = true
			}
		}
	}

	content := strings.Join(newLines, "\n") + "\n"
	if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
		logger(fmt.Sprintf("Warning: Failed to update windscribe_cli.conf: %v", err))
		return nil
	}

	logger(fmt.Sprintf("Windscribe Linux: Injected parameters into windscribe_cli.conf (Amnezia: %s, SNI: %s, Routing: %s, DNS: %s).", targetAmnezia, targetSni, targetRouting, targetDnsPolicy))

	// Reload preferences via CLI
	worker := NewWindscribeWorker()
	if out, err := worker.RunCLI(ctx, "preferences", "reload"); err == nil {
		logger(fmt.Sprintf("Windscribe Linux preferences reloaded: %s", strings.TrimSpace(out)))
	}

	return nil
}
