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
}

func (o *Orchestrator) StartTesting(parentCtx context.Context, config TestConfig, onConnected func(), sendUpdate func(EventUpdate)) {
	o.mu.Lock()
	ctx, cancel := context.WithCancel(parentCtx)
	o.cancel = cancel
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		_ = o.worker.EnsureDisconnected(stopCtx)
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

		// Build the queue matrix
		for _, loc := range locationsToTest {
			for protocol, portsToTest := range config.ProtocolPorts {
				if len(portsToTest) == 0 {
					continue
				}
				for _, port := range portsToTest {
					amneziaMode := config.AmneziaConfig
					if !config.ProtocolTweaks {
						amneziaMode = "Disabled"
					} else if amneziaMode == "" {
						amneziaMode = "None"
					}

					sni := config.CustomSNI
					if protocol != "stealth" && protocol != "wstunnel" {
						sni = "-"
					} else if sni == "" {
						sni = "-"
					}

					serverRouting := config.ServerRouting
					if !config.ProtocolTweaks {
						serverRouting = "-"
					} else if serverRouting == "" {
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
						ProtocolTweaks:     config.ProtocolTweaks,
						AntiCensorshipMode: amneziaMode,
						ServerRouting:      serverRouting,
						CustomSNI:          sni,
						LargeTLS:           config.LargeTLS,
						InternalDNS:        config.InternalDNS,
						ISP:                ispName,
					})
				}
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
		if err != nil {
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

		connected, duration, _ := o.worker.WaitForConnection(ctx, time.Duration(config.Timeout)*time.Second)
		task.Duration = duration

		if !connected {
			task.Status = "TIMEOUT"
			o.db.RecordResult(task)
			completedResults = append(completedResults, task)
			o.mu.Lock()
			o.completedResults = completedResults
			o.mu.Unlock()
			sendUpdate(EventUpdate{
				Message:     fmt.Sprintf("Timeout connecting to %s (%s:%d) after %.1fs", targetLoc, task.Protocol, task.Port, duration),
				Status:      "Timeout",
				TestedCount: len(completedResults),
				TotalCount:  o.totalTasks,
			})
			discCtxTo, discCancelTo := context.WithTimeout(context.Background(), 8*time.Second)
			_ = o.worker.EnsureDisconnected(discCtxTo)
			discCancelTo()
			continue
		}

		// Step 1.5: Adaptive Tuner/Adapter Packet Flow & Anti-Blinking Stabilization Loop (Up to 60s)
		initialIn, initialOut, adapterFound := getPlatformWindscribeTraffic()

		sendUpdate(EventUpdate{
			Message: fmt.Sprintf("[%d/%d] Handshake connected on %s (%s:%d). Monitoring virtual adapter packet flow (up to 60s)...",
				currentTested, o.totalTasks, targetLoc, task.Protocol, task.Port),
			Status:  "Stabilizing...",
		})

		startTime := time.Now()
		maxStabilizeTime := 60 * time.Second
		minBytesThreshold := uint64(20 * 1024) // 20 KB threshold per real-world benchmarks
		isStable := false
		lastLogTime := time.Now()

		for time.Since(startTime) < maxStabilizeTime {
			select {
			case <-ctx.Done():
				break
			case <-time.After(500 * time.Millisecond):
			}

			if ctx.Err() != nil {
				break
			}

			// 1. Verify CLI status is still CONNECTED
			if !o.worker.IsConnected(ctx) {
				break
			}

			curIn, curOut, isUp := getPlatformWindscribeTraffic()
			var inDiff, outDiff uint64
			if curIn >= initialIn {
				inDiff = curIn - initialIn
			}
			if curOut >= initialOut {
				outDiff = curOut - initialOut
			}

			// 2. Check if virtual adapter received sufficient packets/bytes
			if adapterFound && isUp && (inDiff >= minBytesThreshold || (inDiff >= 10*1024 && outDiff >= 5*1024)) {
				isStable = true
				sendUpdate(EventUpdate{
					Message: fmt.Sprintf("Tunnel stabilized in %.1fs (Virtual adapter traffic: ↓%.1f KB / ↑%.1f KB).",
						time.Since(startTime).Seconds(), float64(inDiff)/1024, float64(outDiff)/1024),
					Status: "Stabilized",
				})
				break
			}

			// Periodic UI status update every 3 seconds while waiting
			if time.Since(lastLogTime) >= 3*time.Second {
				lastLogTime = time.Now()
				elapsedSec := int(time.Since(startTime).Seconds())
				if adapterFound {
					sendUpdate(EventUpdate{
						Message: fmt.Sprintf("[%d/%d] Stabilizing %s (00:%02d / 01:00) — Virtual adapter: ↓%.1f KB / ↑%.1f KB...",
							currentTested, o.totalTasks, targetLoc, elapsedSec, float64(inDiff)/1024, float64(outDiff)/1024),
						Status: "Stabilizing...",
					})
				} else {
					sendUpdate(EventUpdate{
						Message: fmt.Sprintf("[%d/%d] Stabilizing %s (00:%02d / 01:00)...",
							currentTested, o.totalTasks, targetLoc, elapsedSec),
						Status: "Stabilizing...",
					})
				}

				// Send lightweight stimulus packet across tunnel to provoke RX/TX activity
				go func() {
					client := &http.Client{Timeout: 2 * time.Second}
					resp, _ := client.Get("http://1.1.1.1/cdn-cgi/trace")
					if resp != nil {
						_ = resp.Body.Close()
					}
				}()
			}

			// Fallback if adapter statistics are not exposed by OS (e.g. locked down permissions)
			if !adapterFound && time.Since(startTime) >= 6*time.Second {
				expectedCode := task.CountryCode
				if expectedCode == "" {
					expectedCode = countryNameToCode[task.Country]
				}
				if ok, _, _ := verifyTunnelDestination(ctx, expectedCode); ok {
					isStable = true
					break
				}
			}
		}

		// If adapter didn't stabilize within 60s or connection dropped:
		if !isStable || !o.worker.IsConnected(ctx) {
			task.Status = "FLAPPING"
			o.db.RecordResult(task)
			completedResults = append(completedResults, task)
			o.mu.Lock()
			o.completedResults = completedResults
			o.mu.Unlock()
			sendUpdate(EventUpdate{
				Message:     fmt.Sprintf("Tunnel stalled on %s (%s:%d): virtual adapter packet flow did not rise (< 20 KB after 60s). Stuck in blinking phase. Disconnecting cleanly.", targetLoc, task.Protocol, task.Port),
				Status:      "Flapping",
				TestedCount: len(completedResults),
				TotalCount:  o.totalTasks,
			})
			discCtxFlap, discCancelFlap := context.WithTimeout(context.Background(), 8*time.Second)
			_ = o.worker.EnsureDisconnected(discCtxFlap)
			discCancelFlap()
			continue
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

			// Step 2: Progressive Speedtest Measurement
			var speedtestClient = speedtest.New()
			serverList, err := speedtestClient.FetchServers()

			if err == nil && len(serverList) > 0 {
				targets, err := serverList.FindServer([]int{})
				if err != nil || len(targets) == 0 {
					targets = serverList
				}
				s := targets[0]

				// Measure Ping & Jitter
				if config.TestPing {
					sendUpdate(EventUpdate{
						Message: fmt.Sprintf("Testing Latency & Jitter on %s...", s.Name),
						Status:  "Testing Latency...",
					})

					s.PingTest(nil)
					task.Ping = float64(s.Latency.Milliseconds())
					task.Jitter = float64(s.Jitter.Milliseconds())

					// Immediately update Ping card
					sendUpdate(EventUpdate{
						Ping:    fmt.Sprintf("%.0f ms (Jitter: %.0f ms)", task.Ping, task.Jitter),
						Message: fmt.Sprintf("Ping: %.0f ms, Jitter: %.0f ms", task.Ping, task.Jitter),
						Status:  "Testing Download...",
					})
				}

				// Measure Download
				if config.TestDownload {
					sendUpdate(EventUpdate{
						Message: "Testing Download speed...",
						Status:  "Testing Download...",
					})

					s.DownloadTest()
					task.Download = float64(s.DLSpeed) / 125000 // Convert bytes/s to Mbps
					if task.Download < 0 {
						task.Download = 0
					}

					// Immediately update Download card
					sendUpdate(EventUpdate{
						Down:    fmt.Sprintf("%.2f Mbps", task.Download),
						Message: fmt.Sprintf("Download Speed: %.2f Mbps", task.Download),
						Status:  "Testing Upload...",
					})
				}

				// Measure Upload
				if config.TestUpload {
					sendUpdate(EventUpdate{
						Message: "Testing Upload speed...",
						Status:  "Testing Upload...",
					})

					s.UploadTest()
					task.Upload = float64(s.ULSpeed) / 125000 // Convert bytes/s to Mbps
					if task.Upload < 0 {
						task.Upload = 0
					}

					// Immediately update Upload card
					sendUpdate(EventUpdate{
						Up:      fmt.Sprintf("%.2f Mbps", task.Upload),
						Message: fmt.Sprintf("Upload Speed: %.2f Mbps", task.Upload),
						Status:  "Saved",
					})
				}

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
	// Only proceed if detected country matches target destination country!
	expectedUpper := strings.ToUpper(strings.TrimSpace(expectedCountryCode))
	if expectedUpper != "" {
		if detectedLoc != expectedUpper {
			if detectedLoc == "IR" || detectedLoc == "RU" || detectedLoc == "CN" {
				return false, detectedLoc, fmt.Errorf("local network leak detected: egress is %s instead of %s", detectedLoc, expectedUpper)
			}
			return false, detectedLoc, fmt.Errorf("destination country mismatch: egress is %s (expected %s)", detectedLoc, expectedUpper)
		}
	} else {
		if detectedLoc == "IR" || detectedLoc == "RU" || detectedLoc == "CN" {
			return false, detectedLoc, fmt.Errorf("local network leak detected: egress is %s", detectedLoc)
		}
	}

	return true, detectedLoc, nil
}
