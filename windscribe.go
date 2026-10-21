package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type WindscribeWorker struct {
	cliPath string
}

func NewWindscribeWorker() *WindscribeWorker {
	return &WindscribeWorker{
		cliPath: defaultWindscribeCLIPath(),
	}
}

func (w *WindscribeWorker) RunCLI(ctx context.Context, args ...string) (string, error) {
	// Guard against hanging windscribe-cli calls by enforcing a 10s maximum ceiling
	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, w.cliPath, args...)
	prepareCmd(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("windscribe-cli timed out after 10s for command %v", args)
		}
		return "", fmt.Errorf("error: %v, stderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func (w *WindscribeWorker) EnsureDisconnected(ctx context.Context) error {
	// Step 1: Quick status check
	out, err := w.RunCLI(ctx, "status")
	upper := strings.ToUpper(out)
	isDisconnected := err == nil && (strings.Contains(upper, "CONNECT STATE: DISCONNECTED") || (strings.Contains(upper, "DISCONNECTED") && !strings.Contains(upper, "DISCONNECTING")))

	if isDisconnected {
		// Even if Windscribe reports Disconnected, ensure no leaked zombie helpers linger
		killPlatformVPNHelpers()
		return nil
	}

	// Step 2: Request disconnection once
	_, _ = w.RunCLI(ctx, "disconnect")

	// Step 3: Wait up to 6 seconds for state to gracefully become Disconnected
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}

		out, err := w.RunCLI(ctx, "status")
		if err == nil {
			upper := strings.ToUpper(out)
			if strings.Contains(upper, "CONNECT STATE: DISCONNECTED") || (strings.Contains(upper, "DISCONNECTED") && !strings.Contains(upper, "DISCONNECTING")) {
				killPlatformVPNHelpers()
				return nil
			}
		}
	}

	// Step 4 (Tier 1): If still not disconnected after 5s (e.g. hung wstunnel/openvpn helper),
	// terminate stuck helper processes on the platform so Windscribe unblocks.
	killPlatformVPNHelpers()
	time.Sleep(1 * time.Second)

	out, err = w.RunCLI(ctx, "status")
	if err == nil {
		upper = strings.ToUpper(out)
		if strings.Contains(upper, "CONNECT STATE: DISCONNECTED") || (strings.Contains(upper, "DISCONNECTED") && !strings.Contains(upper, "DISCONNECTING")) {
			return nil
		}
	}

	// Step 5 (Tier 2 Fail-Safe): If Windscribe is STILL stuck in Disconnecting or unresponsive,
	// perform an automatic soft restart of the Windscribe desktop app to guarantee clean state.
	restartPlatformWindscribeApp()
	time.Sleep(1 * time.Second)
	_, _ = w.RunCLI(ctx, "disconnect")
	return nil
}

func (w *WindscribeWorker) ApplyConnectionTweaks(ctx context.Context, config TestConfig, logger func(string)) error {
	return applyPlatformConnectionTweaks(ctx, config, logger)
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
			return false, time.Since(start).Seconds(), ctx.Err()
		default:
			if time.Since(start) > timeout {
				return false, time.Since(start).Seconds(), nil
			}
			out, err := w.RunCLI(ctx, "status")
			if err == nil {
				upperOut := strings.ToUpper(out)
				if strings.Contains(upperOut, "CONNECTED") && !strings.Contains(upperOut, "DISCONNECTED") && !strings.Contains(upperOut, "CONNECTING") {
					duration := time.Since(start).Seconds()
					return true, duration, nil
				}
			}
			select {
			case <-ctx.Done():
				return false, time.Since(start).Seconds(), ctx.Err()
			case <-time.After(1 * time.Second):
			}
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

