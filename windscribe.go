package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type WindscribeWorker struct {
	cliPath string
}

func NewWindscribeWorker() *WindscribeWorker {
	// Usually C:\Program Files\Windscribe\windscribe-cli.exe
	return &WindscribeWorker{
		cliPath: `C:\Program Files\Windscribe\windscribe-cli.exe`,
	}
}

func (w *WindscribeWorker) RunCLI(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, w.cliPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error: %v, stderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func (w *WindscribeWorker) EnsureDisconnected(ctx context.Context) error {
	out, err := w.RunCLI(ctx, "status")
	if err != nil {
		return err
	}
	if strings.Contains(strings.ToUpper(out), "DISCONNECTED") {
		return nil
	}
	_, err = w.RunCLI(ctx, "disconnect")
	time.Sleep(2 * time.Second)
	return err
}

func (w *WindscribeWorker) Connect(ctx context.Context, city string, protocol string, port int) error {
	targetArg := fmt.Sprintf("%s:%d", protocol, port)
	_, err := w.RunCLI(ctx, "connect", "-n", city, targetArg)
	return err
}

func (w *WindscribeWorker) WaitForConnection(ctx context.Context, timeout time.Duration) (bool, float64, error) {
	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			return false, 0, ctx.Err()
		default:
			if time.Since(start) > timeout {
				return false, 0, nil
			}
			out, err := w.RunCLI(ctx, "status")
			if err == nil {
				upperOut := strings.ToUpper(out)
				if strings.Contains(upperOut, "CONNECTED") && !strings.Contains(upperOut, "DISCONNECTED") && !strings.Contains(upperOut, "CONNECTING") {
					duration := time.Since(start).Seconds()
					return true, duration, nil
				}
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func (w *WindscribeWorker) Disconnect(ctx context.Context) error {
	_, err := w.RunCLI(ctx, "disconnect")
	return err
}

func (w *WindscribeWorker) GetLocations(ctx context.Context) ([]string, error) {
	out, err := w.RunCLI(ctx, "locations")
	var locs []string
	
	if err == nil {
		lines := strings.Split(out, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.Contains(line, "Pro ") && !strings.Contains(line, "Free ") && !strings.Contains(line, "Available Locations") && !strings.HasPrefix(line, "-") {
				if idx := strings.Index(line, "("); idx != -1 {
					locs = append(locs, strings.TrimSpace(line[:idx]))
				}
			}
		}
	}

	// Fallback to static list if CLI failed (e.g. log file permission error)
	if len(locs) == 0 {
		locs = []string{
			"Paris", "London", "Frankfurt", "Amsterdam", "New York",
			"Los Angeles", "Tokyo", "Singapore", "Sydney", "Toronto",
			"Montreal", "Zurich", "Stockholm", "Madrid", "Miami",
			"Chicago", "Dallas", "Seattle", "Atlanta", "Washington DC",
			"Bucharest", "Oslo", "Vienna", "Brussels", "Milan",
			"Warsaw", "Copenhagen", "Helsinki", "Dublin", "Istanbul",
		}
	}

	return locs, nil
}

