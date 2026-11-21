package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func writeFakeCLI(t *testing.T, dir string, scriptBody string) string {
	t.Helper()
	var path string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, "fake-windscribe-cli.bat")
		content := "@echo off\r\n" + scriptBody + "\r\n"
		if err := os.WriteFile(path, []byte(content), 0755); err != nil {
			t.Fatalf("write fake cli: %v", err)
		}
	} else {
		path = filepath.Join(dir, "fake-windscribe-cli.sh")
		content := "#!/bin/sh\n" + scriptBody + "\n"
		if err := os.WriteFile(path, []byte(content), 0755); err != nil {
			t.Fatalf("write fake cli: %v", err)
		}
	}
	return path
}

func TestCLIExistsAndCheckEnvironmentMissing(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "does-not-exist-cli.exe")
	w := &WindscribeWorker{cliPath: missingPath}
	if w.CLIExists() {
		t.Fatalf("expected CLIExists() to be false for %s", missingPath)
	}

	app := &App{worker: w}
	env := app.CheckEnvironment()
	if env.CLIFound {
		t.Fatalf("expected CheckEnvironment().CLIFound to be false")
	}
	if !strings.Contains(env.ErrorMessage, "not found") {
		t.Fatalf("expected informative not-found error message, got %q", env.ErrorMessage)
	}
}

func TestConnectErrorClassificationAndSessionStatus(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Not logged in case
	var notLoggedScript string
	if runtime.GOOS == "windows" {
		notLoggedScript = "echo Not logged in. Run \"windscribe-cli login\" first. 1>&2\r\nexit /b 1"
	} else {
		notLoggedScript = "echo 'Not logged in. Run \"windscribe-cli login\" first.' >&2\nexit 1"
	}
	cli1 := writeFakeCLI(t, dir, notLoggedScript)
	w1 := &WindscribeWorker{cliPath: cli1}
	if !w1.CLIExists() {
		t.Fatalf("expected fake CLI %s to exist", cli1)
	}
	err1 := w1.Connect(ctx, "Crumpets", "stealth", 443)
	if err1 == nil || !strings.Contains(err1.Error(), "NOT_LOGGED_IN") {
		t.Fatalf("expected NOT_LOGGED_IN error, got %v", err1)
	}

	app1 := &App{worker: w1}
	env1 := app1.CheckEnvironment()
	if !env1.CLIFound || env1.LoggedIn {
		t.Fatalf("expected CLIFound=true, LoggedIn=false, got %+v", env1)
	}

	// 2. Another user session locked case
	dir2 := t.TempDir()
	var lockedScript string
	if runtime.GOOS == "windows" {
		lockedScript = "echo Windscribe is already running in another user session on this computer. 1>&2\r\nexit /b 1"
	} else {
		lockedScript = "echo 'Windscribe is already running in another user session on this computer.' >&2\nexit 1"
	}
	cli2 := writeFakeCLI(t, dir2, lockedScript)
	w2 := &WindscribeWorker{cliPath: cli2}
	err2 := w2.Connect(ctx, "Crumpets", "stealth", 443)
	if err2 == nil || !strings.Contains(err2.Error(), "SESSION_LOCKED") {
		t.Fatalf("expected SESSION_LOCKED error, got %v", err2)
	}

	app2 := &App{worker: w2}
	env2 := app2.CheckEnvironment()
	if !env2.CLIFound || !env2.SessionLocked {
		t.Fatalf("expected CLIFound=true, SessionLocked=true, got %+v", env2)
	}
}

func TestGetBestLocationsScansCountryCode(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_best.db")
	dbm, err := NewDatabaseManager(dbPath)
	if err != nil {
		t.Fatalf("NewDatabaseManager: %v", err)
	}
	defer dbm.db.Close()
	dbm.RecordResult(TestResult{
		Location:    "Crumpets",
		City:        "London",
		Country:     "United Kingdom",
		CountryCode: "GB",
		Continent:   "Europe",
		Protocol:    "stealth",
		Port:        443,
		Status:      "SUCCESS",
		Download:    15.5,
		Ping:        120,
		Timestamp:   "2026-11-21 04:00:00",
	})
	best, err := dbm.GetBestLocations()
	if err != nil {
		t.Fatalf("GetBestLocations error: %v", err)
	}
	if len(best) != 1 || best[0].CountryCode != "GB" {
		t.Fatalf("expected CountryCode='GB', got %+v", best)
	}
}
