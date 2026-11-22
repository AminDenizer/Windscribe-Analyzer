package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

// Orchestrator coordinates the testing queue
type Orchestrator struct {
	db               *DatabaseManager
	worker           *WindscribeWorker
	cancel           context.CancelFunc
	mu               sync.Mutex
	runMu            sync.Mutex
	remainingQueue   []TestResult
	completedResults []TestResult
	totalTasks       int
}

func NewOrchestrator(db *DatabaseManager, worker *WindscribeWorker) *Orchestrator {
	return &Orchestrator{
		db:     db,
		worker: worker,
	}
}

type TestConfig struct {
	SelectedLocations  []LocationData   `json:"selectedLocations"`
	ProtocolPorts      map[string][]int `json:"protocolPorts"`
	MasterInjection    bool             `json:"masterInjection"`
	ProtocolTweaksMode string           `json:"protocolTweaksMode"`
	ProtocolTweaks     bool             `json:"protocolTweaks"`
	AmneziaConfig      string           `json:"amneziaConfig"`
	ServerRouting      string           `json:"serverRouting"`
	CustomSNI          string           `json:"customSni"`
	LargeTLS           bool             `json:"largeTls"`
	InternalDNS        string           `json:"internalDns"`
	ISP                string           `json:"isp"`
	IsResume           bool             `json:"isResume"`
	SelectionMode      string           `json:"selectionMode"` // "manual", "smart", "discovery", "all"
	TestPing           bool             `json:"testPing"`
	TestDownload       bool             `json:"testDownload"`
	TestUpload         bool             `json:"testUpload"`
	SpeedWait          int              `json:"speedWait"`
	Timeout            int              `json:"timeout"`
	StabilizeEnabled   bool             `json:"stabilizeEnabled"`
	StabilizeMinBytes  int              `json:"stabilizeMinBytes"`
	StabilizeUnit      string           `json:"stabilizeUnit"` // "KB" or "MB"
	StabilizeTimeout   int              `json:"stabilizeTimeout"`
	SpeedEngine        string           `json:"speedEngine"`      // "ookla", "cloudflare", "both"
	ShuffleLocations   bool             `json:"shuffleLocations"`
	ShuffleProtocols   bool             `json:"shuffleProtocols"`
	ConnectAndHold     bool             `json:"connectAndHold"`
}

type EventUpdate struct {
	Message        string      `json:"message"`
	Ping           string      `json:"ping"`
	Down           string      `json:"down"`
	Up             string      `json:"up"`
	Status         string      `json:"status"`
	ResetCards     bool        `json:"resetCards"`
	ExcelPath      string      `json:"excelPath,omitempty"`
	CanResume      bool        `json:"canResume"`
	TestedCount    int         `json:"testedCount"`
	TotalCount     int         `json:"totalCount"`
	ConnectedItem  *TestResult `json:"connectedItem,omitempty"`
	ErrorTitle     string      `json:"errorTitle,omitempty"`
	ErrorMsg       string      `json:"errorMsg,omitempty"`
}

func (o *Orchestrator) StartTesting(parentCtx context.Context, config TestConfig, onConnected func(), sendUpdate func(EventUpdate)) {
	o.mu.Lock()
	if o.cancel != nil {
		o.cancel()
		o.cancel = nil
	}
	o.mu.Unlock()

	o.runMu.Lock()
	defer o.runMu.Unlock()

	o.mu.Lock()
	ctx, cancel := context.WithCancel(parentCtx)
	o.cancel = cancel
	keepConnectedOnExit := false
	defer func() {
		if !keepConnectedOnExit {
			stopCtx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
			defer cancel()
			_ = o.worker.EnsureDisconnected(stopCtx)
		}
	}()

	var queue []TestResult
	var completedResults []TestResult

	if config.IsResume && len(o.remainingQueue) > 0 {
		queue = o.remainingQueue
		completedResults = o.completedResults
		sendUpdate(EventUpdate{
			Message:     fmt.Sprintf("Resuming test session: %d remaining (previously completed %d of %d).", len(queue), len(completedResults), o.totalTasks),
			Status:      "Resuming...",
			CanResume:   false,
			TestedCount: len(completedResults),
			TotalCount:  o.totalTasks,
		})
	} else {
		// Fresh Start
		o.completedResults = nil
		o.remainingQueue = nil

		var locationsToTest []LocationData = config.SelectedLocations

		if config.SelectionMode == "smart" {
			sendUpdate(EventUpdate{Message: "Calculating Smart Locations from history...", Status: "Analyzing"})
			best, err := o.db.GetBestLocations()
			if err == nil && len(best) > 0 {
				locMap := make(map[string]bool)
				var smartLocs []LocationData
				for _, r := range best {
					if !locMap[r.Location] {
						locMap[r.Location] = true
						smartLocs = append(smartLocs, LocationData{
							Location:    r.Location,
							City:        r.City,
							Region:      r.Country,
							CountryCode: r.CountryCode,
							Continent:   r.Continent,
							Display:     fmt.Sprintf("%s - %s (%s)", r.Country, r.City, r.Location),
						})
					}
				}
				if len(smartLocs) > 0 {
					locationsToTest = smartLocs
				}
			} else {
				sendUpdate(EventUpdate{Message: "No history found for Smart mode. Using selected locations.", Status: "Fallback"})
			}
		} else if config.SelectionMode == "discovery" {
			rand.Seed(time.Now().UnixNano())
			rand.Shuffle(len(locationsToTest), func(i, j int) { locationsToTest[i], locationsToTest[j] = locationsToTest[j], locationsToTest[i] })
			if len(locationsToTest) > 5 {
				locationsToTest = locationsToTest[:5] // Test 5 random locations
			}
		}

		ispName := strings.TrimSpace(config.ISP)
		if ispName == "" {
			ispName = "Default / Direct"
		}

		// Build protocol-port pairs
		type protoPort struct {
			protocol string
			port     int
		}
		var protoPairs []protoPort
		for protocol, portsToTest := range config.ProtocolPorts {
			for _, port := range portsToTest {
				protoPairs = append(protoPairs, protoPort{protocol: protocol, port: port})
			}
		}

		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		if config.ShuffleLocations && len(locationsToTest) > 1 {
			shuffledLocs := make([]LocationData, len(locationsToTest))
			copy(shuffledLocs, locationsToTest)
			rng.Shuffle(len(shuffledLocs), func(i, j int) { shuffledLocs[i], shuffledLocs[j] = shuffledLocs[j], shuffledLocs[i] })
			locationsToTest = shuffledLocs
		}
		if config.ShuffleProtocols && len(protoPairs) > 1 {
			rng.Shuffle(len(protoPairs), func(i, j int) { protoPairs[i], protoPairs[j] = protoPairs[j], protoPairs[i] })
		}

		appendTask := func(loc LocationData, protocol string, port int) {
			p := strings.ToLower(protocol)
			isTweaksSupported := (p == "wireguard" || p == "udp" || p == "tcp" || p == "stealth")
			taskTweaks := config.ProtocolTweaks && isTweaksSupported

			taskAmnezia := "-"
			if isTweaksSupported && config.ProtocolTweaks {
				if config.AmneziaConfig != "" && config.AmneziaConfig != "None (Disabled)" && config.AmneziaConfig != "Disabled" && config.AmneziaConfig != "None" {
					taskAmnezia = config.AmneziaConfig
				} else {
					taskAmnezia = "None"
				}
			}

			sni := config.CustomSNI
			if p != "stealth" && p != "wstunnel" {
				sni = "-"
			} else if sni == "" {
				sni = "-"
			}

			serverRouting := config.ServerRouting
			if serverRouting == "" {
				serverRouting = "Auto"
			}

			queue = append(queue, TestResult{
				Timestamp:          time.Now().Format("2006-01-02 15:04:05"),
				Location:           loc.Location,
				City:               loc.City,
				Country:            loc.Region,
				CountryCode:        loc.CountryCode,
				Continent:          loc.Continent,
				Protocol:           protocol,
				Port:               port,
				ProtocolTweaks:     taskTweaks,
				AntiCensorshipMode: taskAmnezia,
				ServerRouting:      serverRouting,
				CustomSNI:          sni,
				LargeTLS:           config.LargeTLS && (p == "stealth" || p == "wstunnel"),
				InternalDNS:        config.InternalDNS,
				ISP:                ispName,
			})
		}

		// Build the queue matrix (interleave by protocol when ShuffleLocations is enabled to avoid hammering one IP)
		if config.ShuffleLocations && !config.ShuffleProtocols {
			for _, pp := range protoPairs {
				for _, loc := range locationsToTest {
					appendTask(loc, pp.protocol, pp.port)
				}
			}
		} else {
			for _, loc := range locationsToTest {
				for _, pp := range protoPairs {
					appendTask(loc, pp.protocol, pp.port)
				}
			}
			if config.ShuffleLocations && config.ShuffleProtocols && len(queue) > 1 {
				rng.Shuffle(len(queue), func(i, j int) { queue[i], queue[j] = queue[j], queue[i] })
			}
		}

		o.totalTasks = len(queue)
		o.remainingQueue = queue

		sendUpdate(EventUpdate{
			Message:     fmt.Sprintf("Queued %d tests across %d location(s) [ISP: %s].", len(queue), len(locationsToTest), ispName),
			Status:      "Ready",
			TotalCount:  o.totalTasks,
			TestedCount: 0,
		})
	}
	o.mu.Unlock()

	// Pre-flight CLI existence, authentication & session check before running the queue
	if !o.worker.CLIExists() {
		sendUpdate(EventUpdate{
			Message:    fmt.Sprintf("Windscribe CLI not found at %s. Aborting test queue.", o.worker.cliPath),
			Status:     "Done",
			ErrorTitle: "Windscribe Not Found",
			ErrorMsg:   fmt.Sprintf("windscribe-cli was not found at:\n%s\n\nPlease install Windscribe or enable Custom CLI Path in Settings.", o.worker.cliPath),
		})
		return
	}

	stOut, stErr := o.worker.RunCLI(ctx, "status")
	stLower := strings.ToLower(stOut)
	if stErr != nil {
		stLower += " " + strings.ToLower(stErr.Error())
	}
	if strings.Contains(stLower, "not logged in") {
		sendUpdate(EventUpdate{
			Message: "Windscribe CLI reported 'Not logged in'. Attempting automatic Windscribe restart recovery...",
			Status:  "Recovering...",
		})
		if !o.worker.RecoverFromNotLoggedIn(ctx) {
			sendUpdate(EventUpdate{
				Message:    "Windscribe CLI still reports 'Not logged in' after restart. Aborting test queue.",
				Status:     "Done",
				ErrorTitle: "Windscribe Not Logged In",
				ErrorMsg:   "Windscribe CLI reports 'Not logged in' even after restarting Windscribe. Please sign in to Windscribe and try again.",
			})
			return
		}
		sendUpdate(EventUpdate{
			Message: "Windscribe session recovered successfully after restart.",
			Status:  "Ready",
		})
	}
	if strings.Contains(stLower, "another user session") {
		sendUpdate(EventUpdate{
			Message:    "Windscribe is running in another user session. Aborting test queue.",
			Status:     "Done",
			ErrorTitle: "Windscribe Session Mismatch",
			ErrorMsg:   "Windscribe is already running in another Windows user session. Please run Windscribe and the tester under the same user session.",
		})
		return
	}

	// Inject and synchronize live Anti-Censorship settings (Amnezia, SNI, Server Routing, Large TLS) into Windscribe
	sendUpdate(EventUpdate{
		Message: "Synchronizing Anti-Censorship parameters with Windscribe Engine...",
		Status:  "Configuring...",
	})
	_ = o.worker.ApplyConnectionTweaks(ctx, config, func(msg string) {
		sendUpdate(EventUpdate{
			Message: msg,
			Status:  "Configuring...",
		})
	})

	for i, task := range queue {
		o.mu.Lock()
		o.remainingQueue = queue[i:]
		o.mu.Unlock()

		if ctx.Err() != nil {
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 8*time.Second)
			_ = o.worker.EnsureDisconnected(stopCtx)
			stopCancel()

			o.mu.Lock()
			var excelPath string
			if len(completedResults) > 0 {
				timestamp := time.Now().Format("2006-01-02_15-04-05")
				outName := getCachePath(fmt.Sprintf("Windscribe_Results_%s.xlsx", timestamp))
				if path, err := GenerateExcelReport(completedResults, outName); err == nil {
					excelPath = path
				}
			}
			o.completedResults = completedResults
			sendUpdate(EventUpdate{
				Message:     fmt.Sprintf("Testing stopped by user (%d completed, %d remaining). Windscribe disconnected.", len(completedResults), len(o.remainingQueue)),
				Status:      "Stopped",
				CanResume:   len(o.remainingQueue) > 0,
				TestedCount: len(completedResults),
				TotalCount:  o.totalTasks,
				ExcelPath:   excelPath,
			})
			o.mu.Unlock()
			return
		}

		// Step 1: Reset telemetry cards for next test
		currentTested := len(completedResults) + 1
		task.Timestamp = time.Now().Format("2006-01-02 15:04:05")
		sendUpdate(EventUpdate{
			ResetCards:  true,
			Message:     fmt.Sprintf("[%d/%d] Preparing %s - %s (%s:%d)...", currentTested, o.totalTasks, task.Country, task.Location, task.Protocol, task.Port),
			Status:      "Connecting...",
			TestedCount: len(completedResults),
			TotalCount:  o.totalTasks,
		})

		discCtx1, discCancel1 := context.WithTimeout(ctx, 8*time.Second)
		_ = o.worker.EnsureDisconnected(discCtx1)
		discCancel1()

		// Target connects by Location Nickname (e.g. "1984", "Seine")
		targetLoc := task.Location
		if targetLoc == "" {
			targetLoc = task.City
		}

		err := o.worker.Connect(ctx, targetLoc, task.Protocol, task.Port)
		if err != nil && strings.Contains(err.Error(), "NOT_LOGGED_IN") {
			sendUpdate(EventUpdate{
				Message: fmt.Sprintf("Windscribe CLI glitched ('Not logged in') on %s (%s:%d). Restarting Windscribe to recover...", targetLoc, task.Protocol, task.Port),
				Status:  "Recovering...",
			})
			if o.worker.RecoverFromNotLoggedIn(ctx) {
				err = o.worker.Connect(ctx, targetLoc, task.Protocol, task.Port)
			}
		}
		if err != nil {
			if strings.Contains(err.Error(), "NOT_LOGGED_IN") {
				sendUpdate(EventUpdate{
					Message:    "Windscribe CLI still reports: Not logged in after restart. Aborting test queue.",
					Status:     "Done",
					ErrorTitle: "Windscribe Not Logged In",
					ErrorMsg:   "Windscribe CLI is not logged in even after restarting Windscribe. Please sign in to Windscribe and try again.",
				})
				return
			}
			if strings.Contains(err.Error(), "SESSION_LOCKED") {
				sendUpdate(EventUpdate{
					Message:    "Windscribe is running in another user session. Aborting test queue.",
					Status:     "Done",
					ErrorTitle: "Windscribe Session Mismatch",
					ErrorMsg:   "Windscribe is already running in another Windows user session. Please run Windscribe and the tester under the same user session.",
				})
				return
			}
			task.Status = "CLI_ERROR"
			o.db.RecordResult(task)
			completedResults = append(completedResults, task)
			o.mu.Lock()
			o.completedResults = completedResults
			o.mu.Unlock()
			sendUpdate(EventUpdate{
				Message:     fmt.Sprintf("Error launching connection to %s (%s:%d): %v", targetLoc, task.Protocol, task.Port, err),
				Status:      "CLI Error",
				TestedCount: len(completedResults),
				TotalCount:  o.totalTasks,
			})
			discCtxErr, discCancelErr := context.WithTimeout(context.Background(), 8*time.Second)
			_ = o.worker.EnsureDisconnected(discCtxErr)
			discCancelErr()
			continue
		}

		connected, duration, connErr := o.worker.WaitForConnectionWithRetry(ctx, time.Duration(config.Timeout)*time.Second, targetLoc, task.Protocol, task.Port, func(msg string) {
			sendUpdate(EventUpdate{
				Message: fmt.Sprintf("[%d/%d] %s", currentTested, o.totalTasks, msg),
				Status:  "Connecting...",
			})
		})
		task.Duration = duration

		if !connected {
			task.Status = "TIMEOUT"
			o.db.RecordResult(task)
			completedResults = append(completedResults, task)
			o.mu.Lock()
			o.completedResults = completedResults
			o.mu.Unlock()
			failMsg := fmt.Sprintf("Timeout connecting to %s (%s:%d) after %.1fs", targetLoc, task.Protocol, task.Port, duration)
			if connErr != nil && connErr != context.Canceled && connErr != context.DeadlineExceeded {
				failMsg = fmt.Sprintf("Connection aborted on %s (%s:%d) after %.1fs: %v", targetLoc, task.Protocol, task.Port, duration, connErr)
			}
			sendUpdate(EventUpdate{
				Message:     failMsg,
				Status:      "Timeout",
				TestedCount: len(completedResults),
				TotalCount:  o.totalTasks,
			})
			discCtxTo, discCancelTo := context.WithTimeout(context.Background(), 8*time.Second)
			_ = o.worker.EnsureDisconnected(discCtxTo)
			discCancelTo()
			continue
		}

		// Step 1.5: Adaptive Tuner/Adapter Packet Flow & Anti-Blinking Stabilization Loop (Configurable)
		if config.StabilizeEnabled {
			maxSec := config.StabilizeTimeout
			if maxSec <= 0 {
				maxSec = 30
			}
			targetVal := config.StabilizeMinBytes
			if targetVal <= 0 {
				targetVal = 250
			}
			unit := strings.ToUpper(strings.TrimSpace(config.StabilizeUnit))
			if unit != "MB" {
				unit = "KB"
			}
			var minBytesThreshold uint64
			if unit == "MB" {
				minBytesThreshold = uint64(targetVal) * 1024 * 1024
			} else {
				minBytesThreshold = uint64(targetVal) * 1024
			}
			targetLabel := fmt.Sprintf("%d %s", targetVal, unit)

			initialIn, initialOut, adapterFound := getPlatformWindscribeTrafficForProtocol(task.Protocol)

			sendUpdate(EventUpdate{
				Message: fmt.Sprintf("[%d/%d] Handshake reported on %s (%s:%d). Verifying active adapter & traffic flow (window: %ds, target: %s)...",
					currentTested, o.totalTasks, targetLoc, task.Protocol, task.Port, maxSec, targetLabel),
				Status:  "Stabilizing...",
			})

			startTime := time.Now()
			maxStabilizeTime := time.Duration(maxSec) * time.Second
			isStable := false
			var lastTotalDiff uint64

			// Launch background payload downloader to stimulate real packet flow through the adapter
			probeCtx, cancelProbe := context.WithCancel(ctx)
			defer cancelProbe()

			probeBytes := minBytesThreshold
			if probeBytes < 500*1024 {
				probeBytes = 500 * 1024
			} else if probeBytes > 5*1024*1024 {
				probeBytes = 5 * 1024 * 1024
			}

			go func(chunkBytes uint64) {
				client := &http.Client{Timeout: 8 * time.Second}
				probeURL := fmt.Sprintf("https://speed.cloudflare.com/__down?bytes=%d", chunkBytes)
				for {
					select {
					case <-probeCtx.Done():
						return
					default:
					}

					resp, err := client.Get(probeURL)
					if err == nil && resp != nil {
						_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, int64(chunkBytes)+65536))
						_ = resp.Body.Close()
					} else {
						for i := 0; i < 3; i++ {
							if r, e := client.Get("https://1.1.1.1/cdn-cgi/trace"); e == nil && r != nil {
								_, _ = io.Copy(io.Discard, r.Body)
								_ = r.Body.Close()
							}
						}
					}

					select {
					case <-probeCtx.Done():
						return
					case <-time.After(2 * time.Second):
					}
				}
			}(probeBytes)

			for time.Since(startTime) < maxStabilizeTime {
				select {
				case <-ctx.Done():
					break
				case <-time.After(500 * time.Millisecond):
				}

				if ctx.Err() != nil {
					break
				}

				// 1. Verify Windscribe CLI status: detect if Windscribe dropped completely
				statusOut, errStatus := o.worker.RunCLI(ctx, "status")
				if errStatus == nil {
					upperStatus := strings.ToUpper(statusOut)
					if strings.Contains(upperStatus, "DISCONNECTED") && !strings.Contains(upperStatus, "DISCONNECTING") {
						break
					}
				}

				curIn, curOut, isUp := getPlatformWindscribeTrafficForProtocol(task.Protocol)
				if !adapterFound && isUp {
					initialIn = curIn
					initialOut = curOut
					adapterFound = true
				}
				var inDiff, outDiff uint64
				if isUp {
					if curIn < initialIn {
						initialIn = 0
					}
					if curOut < initialOut {
						initialOut = 0
					}
					inDiff = curIn - initialIn
					outDiff = curOut - initialOut
				}
				totalDiff := inDiff + outDiff
				lastTotalDiff = totalDiff

				formatTransferred := func(b uint64) string {
					if unit == "MB" || b >= 1024*1024 {
						return fmt.Sprintf("%.2f MB", float64(b)/(1024*1024))
					}
					return fmt.Sprintf("%.1f KB", float64(b)/1024)
				}

				// 2. Check if virtual adapter transferred >= target threshold of payload traffic
				if adapterFound && isUp && totalDiff >= minBytesThreshold {
					isStable = true
					cancelProbe()
					sendUpdate(EventUpdate{
						Message: fmt.Sprintf("Tunnel stabilized in %.1fs (Adapter active, transferred %s [↓%.1f KB / ↑%.1f KB] >= %s).",
							time.Since(startTime).Seconds(), formatTransferred(totalDiff), float64(inDiff)/1024, float64(outDiff)/1024, targetLabel),
						Status: "Stabilized",
					})
					break
				}

				// Fallback if adapter statistics are not exposed by OS (e.g. locked down permissions)
				if !adapterFound && time.Since(startTime) >= 6*time.Second {
					expectedCode := task.CountryCode
					if expectedCode == "" {
						expectedCode = countryNameToCode[task.Country]
					}
					if ok, _, _ := verifyTunnelDestination(ctx, expectedCode); ok {
						isStable = true
						cancelProbe()
						break
					}
				}
			}
			cancelProbe()

			// If adapter didn't reach threshold within window or connection dropped:
			if !isStable || !o.worker.IsConnected(ctx) {
				task.Status = "FLAPPING"
				o.db.RecordResult(task)
				completedResults = append(completedResults, task)
				o.mu.Lock()
				o.completedResults = completedResults
				o.mu.Unlock()
				sendUpdate(EventUpdate{
					Message:     fmt.Sprintf("Tunnel stalled on %s (%s:%d): transferred %.1f KB (< %s) within %ds. Disconnecting cleanly.", targetLoc, task.Protocol, task.Port, float64(lastTotalDiff)/1024, targetLabel, maxSec),
					Status:      "Flapping",
					TestedCount: len(completedResults),
					TotalCount:  o.totalTasks,
				})
				discCtxFlap, discCancelFlap := context.WithTimeout(context.Background(), 8*time.Second)
				_ = o.worker.EnsureDisconnected(discCtxFlap)
				discCancelFlap()
				continue
			}
		} else {
			sendUpdate(EventUpdate{
				Message: fmt.Sprintf("[%d/%d] Handshake reported on %s (%s:%d) (Stabilization check disabled; proceeding).",
					currentTested, o.totalTasks, targetLoc, task.Protocol, task.Port),
				Status: "Connected",
			})
		}

		// Step 1.6: Autonomous Tunnel Integrity & Route Leak Verification
		expectedCode := task.CountryCode
		if expectedCode == "" {
			expectedCode = countryNameToCode[task.Country]
		}

		sendUpdate(EventUpdate{
			Message: fmt.Sprintf("Verifying route destination for %s (expected: %s)...", task.Country, expectedCode),
			Status:  "Verifying...",
		})

		tunnelOk, egressLoc, leakErr := verifyTunnelDestination(ctx, expectedCode)
		if !tunnelOk {
			// Quick 2-second retry in case route metric was still applying
			time.Sleep(2000 * time.Millisecond)
			tunnelOk, egressLoc, leakErr = verifyTunnelDestination(ctx, expectedCode)
		}

		if !tunnelOk {
			task.Status = "ROUTING_LEAK"
			o.db.RecordResult(task)
			completedResults = append(completedResults, task)
			o.mu.Lock()
			o.completedResults = completedResults
			o.mu.Unlock()
			sendUpdate(EventUpdate{
				Message:     fmt.Sprintf("Route verification failed on %s (%s:%d): %v. Skipping speedtest to prevent false metrics.", targetLoc, task.Protocol, task.Port, leakErr),
				Status:      "Route Leak",
				TestedCount: len(completedResults),
				TotalCount:  o.totalTasks,
			})
			discCtxLeak, discCancelLeak := context.WithTimeout(context.Background(), 8*time.Second)
			_ = o.worker.EnsureDisconnected(discCtxLeak)
			discCancelLeak()
			continue
		}

		task.Status = "SUCCESS"
		locTag := ""
		if egressLoc != "" {
			locTag = fmt.Sprintf(" [Exit Node: %s]", egressLoc)
		}
		sendUpdate(EventUpdate{
			Message:     fmt.Sprintf("Connected & verified %s (%s:%d)%s in %.2fs! Testing metrics...", targetLoc, task.Protocol, task.Port, locTag, duration),
			Status:      "Connected",
			TestedCount: len(completedResults),
			TotalCount:  o.totalTasks,
		})

			// Trigger online background sync if needed
			if onConnected != nil {
				go onConnected()
			}

			// Wait specified seconds for route stability (abortable via context)
			sleepDuration := time.Duration(config.SpeedWait) * time.Second
			if sleepDuration < 1*time.Second {
				sleepDuration = 1 * time.Second
			}
			select {
			case <-ctx.Done():
			case <-time.After(sleepDuration):
			}

			// Step 2: Progressive Speedtest Measurement (Ookla / Cloudflare / Both)
			engine := strings.ToLower(strings.TrimSpace(config.SpeedEngine))
			if engine == "" {
				engine = "ookla"
			}

			runOokla := func() bool {
				speedtestClient := speedtest.New()
				serverList, err := speedtestClient.FetchServers()
				if err != nil || len(serverList) == 0 {
					return false
				}
				targets, err := serverList.FindServer([]int{})
				if err != nil || len(targets) == 0 {
					targets = serverList
				}
				s := targets[0]

				if config.TestPing {
					sendUpdate(EventUpdate{
						Message: fmt.Sprintf("Testing Ookla Latency & Jitter on %s...", s.Name),
						Status:  "Testing Latency...",
					})
					s.PingTest(nil)
					task.Ping = float64(s.Latency.Milliseconds())
					if task.Ping < 0 {
						task.Ping = 0
					}
					task.Jitter = float64(s.Jitter.Milliseconds())
					if task.Jitter < 0 || task.Jitter > 60000 {
						task.Jitter = 0
					}
					sendUpdate(EventUpdate{
						Ping:    fmt.Sprintf("%.0f ms (Jitter: %.0f ms)", task.Ping, task.Jitter),
						Message: fmt.Sprintf("Ookla Ping: %.0f ms, Jitter: %.0f ms", task.Ping, task.Jitter),
						Status:  "Testing Download...",
					})
				}
				if config.TestDownload {
					sendUpdate(EventUpdate{
						Message: "Testing Ookla Download speed...",
						Status:  "Testing Download...",
					})
					s.DownloadTest()
					task.Download = float64(s.DLSpeed) / 125000
					if task.Download < 0 {
						task.Download = 0
					}
					sendUpdate(EventUpdate{
						Down:    fmt.Sprintf("%.2f Mbps", task.Download),
						Message: fmt.Sprintf("Ookla Download Speed: %.2f Mbps", task.Download),
						Status:  "Testing Upload...",
					})
				}
				if config.TestUpload {
					sendUpdate(EventUpdate{
						Message: "Testing Ookla Upload speed...",
						Status:  "Testing Upload...",
					})
					s.UploadTest()
					task.Upload = float64(s.ULSpeed) / 125000
					if task.Upload < 0 {
						task.Upload = 0
					}
					sendUpdate(EventUpdate{
						Up:      fmt.Sprintf("%.2f Mbps", task.Upload),
						Message: fmt.Sprintf("Ookla Upload Speed: %.2f Mbps", task.Upload),
						Status:  "Saved",
					})
				}
				return true
			}

			runCloudflare := func() bool {
				sendUpdate(EventUpdate{
					Message: "Testing Cloudflare Speedtest (speed.cloudflare.com)...",
					Status:  "Testing Cloudflare...",
				})
				cfPing, cfJitter, cfDown, cfUp, cfErr := runCloudflareSpeedtest(ctx, config.TestPing, config.TestDownload, config.TestUpload)
				if cfErr != nil {
					sendUpdate(EventUpdate{
						Message: fmt.Sprintf("Cloudflare speedtest failed: %v", cfErr),
						Status:  "Cloudflare N/A",
					})
					return false
				}
				task.Ping = cfPing
				task.Jitter = cfJitter
				task.Download = cfDown
				task.Upload = cfUp
				sendUpdate(EventUpdate{
					Ping:    fmt.Sprintf("%.0f ms (Jitter: %.0f ms)", task.Ping, task.Jitter),
					Down:    fmt.Sprintf("%.2f Mbps", task.Download),
					Up:      fmt.Sprintf("%.2f Mbps", task.Upload),
					Message: fmt.Sprintf("Cloudflare: Ping=%.0fms (Jitter=%.0fms) | Down=%.2fMbps | Up=%.2fMbps", task.Ping, task.Jitter, task.Download, task.Upload),
					Status:  "Saved",
				})
				return true
			}

			ooklaOK := false
			cfOK := false
			switch engine {
			case "cloudflare":
				cfOK = runCloudflare()
			case "both":
				ooklaOK = runOokla()
				cfOK = runCloudflare()
			default:
				ooklaOK = runOokla()
			}

			if ooklaOK || cfOK {
				sendUpdate(EventUpdate{
					Message: fmt.Sprintf("Results for %s (%s:%d): Ping=%.0fms | Down=%.2fMbps | Up=%.2fMbps",
						targetLoc, task.Protocol, task.Port, task.Ping, task.Download, task.Upload),
					Status:        "Saved",
					ConnectedItem: &task,
				})
			} else {
				sendUpdate(EventUpdate{
					Message:       "Speedtest server unavailable or blocked.",
					Status:        "Server N/A",
					ConnectedItem: &task,
				})
			}

		o.db.RecordResult(task)
		completedResults = append(completedResults, task)
		o.mu.Lock()
		o.completedResults = completedResults
		o.mu.Unlock()

		// Pause on First Connected Node: only stop on first hit (do NOT pause again after Continue Test!)
		if config.ConnectAndHold && !config.IsResume {
			o.mu.Lock()
			o.remainingQueue = queue[i+1:]
			remCount := len(o.remainingQueue)
			o.mu.Unlock()

			var excelPath string
			if len(completedResults) > 0 {
				timestamp := time.Now().Format("2006-01-02_15-04-05")
				outName := getCachePath(fmt.Sprintf("Windscribe_Results_%s.xlsx", timestamp))
				if path, err := GenerateExcelReport(completedResults, outName); err == nil {
					excelPath = path
				}
			}

			keepConnectedOnExit = true
			sendUpdate(EventUpdate{
				Message:     fmt.Sprintf("Paused on first connected node: %s (%s:%d). Windscribe stays connected (%d remaining in queue).", targetLoc, task.Protocol, task.Port, remCount),
				Status:      "Connected (Paused)",
				CanResume:   remCount > 0,
				TestedCount: len(completedResults),
				TotalCount:  o.totalTasks,
				ExcelPath:   excelPath,
			})
			return
		}

		discCtxDone, discCancelDone := context.WithTimeout(ctx, 8*time.Second)
		_ = o.worker.EnsureDisconnected(discCtxDone)
		discCancelDone()
		time.Sleep(500 * time.Millisecond)
	}

	// Always ensure complete disconnect when all tests finish
	finalCtx, finalCancel := context.WithTimeout(context.Background(), 8*time.Second)
	_ = o.worker.EnsureDisconnected(finalCtx)
	finalCancel()

	o.mu.Lock()
	o.remainingQueue = nil
	o.completedResults = completedResults
	o.mu.Unlock()

	// Automatic Excel export at end of testing
	var excelPath string
	if len(completedResults) > 0 {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		outName := getCachePath(fmt.Sprintf("Windscribe_Results_%s.xlsx", timestamp))
		if path, err := GenerateExcelReport(completedResults, outName); err == nil {
			excelPath = path
			sendUpdate(EventUpdate{
				Message: fmt.Sprintf("Excel report generated automatically: %s", filepath.Base(path)),
				Status:  "Done",
			})
		}
	}

	sendUpdate(EventUpdate{
		Message:     "All queued tests completed successfully!",
		Status:      "Done",
		CanResume:   false,
		TestedCount: len(completedResults),
		TotalCount:  o.totalTasks,
		ExcelPath:   excelPath,
	})
}

func (o *Orchestrator) Stop() {
	o.mu.Lock()
	if o.cancel != nil {
		o.cancel()
		o.cancel = nil
	}
	o.mu.Unlock()

	// Asynchronously ensure disconnected so Wails IPC returns immediately to the UI
	go func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		_ = o.worker.EnsureDisconnected(stopCtx)
	}()
}

var countryNameToCode = map[string]string{
	"US Central": "US", "US East": "US", "US West": "US", "United States": "US", "USA": "US",
	"Canada East": "CA", "Canada West": "CA", "Canada": "CA",
	"Austria": "AT", "Australia": "AU", "Belgium": "BE", "Bulgaria": "BG",
	"Switzerland": "CH", "Cyprus": "CY", "Czech Republic": "CZ",
	"Germany": "DE", "Denmark": "DK", "Estonia": "EE", "Spain": "ES",
	"Finland": "FI", "France": "FR", "United Kingdom": "GB", "UK": "GB", "Greece": "GR",
	"Hong Kong": "HK", "Croatia": "HR", "Hungary": "HU", "Ireland": "IE",
	"Israel": "IL", "India": "IN", "Italy": "IT", "Japan": "JP",
	"Iceland": "IS", "Latvia": "LV", "Lithuania": "LT", "Luxembourg": "LU",
	"Moldova": "MD", "Netherlands": "NL", "Norway": "NO", "Poland": "PL",
	"Portugal": "PT", "Romania": "RO", "Serbia": "RS", "Sweden": "SE",
	"Singapore": "SG", "Slovakia": "SK", "Slovenia": "SI", "Turkey": "TR", "Ukraine": "UA",
	"South Africa": "ZA", "Albania": "AL", "Bosnia": "BA", "Georgia": "GE",
	"Kenya": "KE", "Nigeria": "NG", "Russia": "RU", "New Zealand": "NZ",
	"Indonesia": "ID", "Malaysia": "MY", "Philippines": "PH", "South Korea": "KR",
	"Taiwan": "TW", "Thailand": "TH", "United Arab Emirates": "AE", "UAE": "AE", "Vietnam": "VN",
	"Argentina": "AR", "Brazil": "BR", "Chile": "CL", "Colombia": "CO",
	"Ecuador": "EC", "El Salvador": "SV", "Guatemala": "GT", "Mexico": "MX",
	"Panama": "PA", "Paraguay": "PY", "Peru": "PE", "Uruguay": "UY", "North Macedonia": "MK",
}

// verifyTunnelDestination checks if the VPN interface is truly routing traffic
// to the expected destination without querying or storing the user's IP address.
func verifyTunnelDestination(ctx context.Context, expectedCountryCode string) (bool, string, error) {
	client := &http.Client{
		Timeout: 4 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://1.1.1.1/cdn-cgi/trace", nil)
	if err != nil {
		return false, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := client.Do(req)
	if err != nil {
		req2, err2 := http.NewRequestWithContext(ctx, "GET", "https://cloudflare.com/cdn-cgi/trace", nil)
		if err2 != nil {
			return false, "", err
		}
		resp, err = client.Do(req2)
		if err != nil {
			return false, "", err
		}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if err != nil {
		return false, "", err
	}

	lines := strings.Split(string(bodyBytes), "\n")
	detectedLoc := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "loc=") {
			detectedLoc = strings.ToUpper(strings.TrimPrefix(line, "loc="))
			break
		}
	}

	if detectedLoc == "" {
		return false, "", fmt.Errorf("exit location not found in trace")
	}

	// Strict Destination Match & Leak Prevention:
	// Only proceed if detected country strictly matches target destination country (no hardcoded country list).
	expectedUpper := strings.ToUpper(strings.TrimSpace(expectedCountryCode))
	if expectedUpper != "" && detectedLoc != expectedUpper {
		return false, detectedLoc, fmt.Errorf("route leak / destination mismatch: egress is %s (expected %s)", detectedLoc, expectedUpper)
	}

	return true, detectedLoc, nil
}
