package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

// Orchestrator coordinates the testing queue
type Orchestrator struct {
	db       *DatabaseManager
	worker   *WindscribeWorker
	cancel   context.CancelFunc
}

func NewOrchestrator(db *DatabaseManager, worker *WindscribeWorker) *Orchestrator {
	return &Orchestrator{
		db:       db,
		worker:   worker,
	}
}

type TestConfig struct {
	SelectedLocations    []LocationData   `json:"selectedLocations"`
	ProtocolPorts        map[string][]int `json:"protocolPorts"`
	ProtocolTweaks       bool             `json:"protocolTweaks"`
	AmneziaConfig        string           `json:"amneziaConfig"`
	ServerRouting        string           `json:"serverRouting"`
	CustomSNI            string           `json:"customSni"`
	LargeTLS             bool             `json:"largeTls"`
	InternalDNS          string           `json:"internalDns"`
	SelectionMode        string           `json:"selectionMode"` // "manual", "smart", "discovery", "all"
	TestPing             bool             `json:"testPing"`
	TestDownload         bool             `json:"testDownload"`
	TestUpload           bool             `json:"testUpload"`
	SpeedWait            int              `json:"speedWait"`
	Timeout              int              `json:"timeout"`
}

type EventUpdate struct {
	Message    string `json:"message"`
	Ping       string `json:"ping"`
	Down       string `json:"down"`
	Up         string `json:"up"`
	Status     string `json:"status"`
	ResetCards bool   `json:"resetCards"`
	ExcelPath  string `json:"excelPath,omitempty"`
}

func (o *Orchestrator) StartTesting(parentCtx context.Context, config TestConfig, onConnected func(), sendUpdate func(EventUpdate)) {
	ctx, cancel := context.WithCancel(parentCtx)
	o.cancel = cancel

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
						Location:  r.Location,
						City:      r.City,
						Region:    r.Country,
						Continent: r.Continent,
						Display:   fmt.Sprintf("%s - %s (%s)", r.Country, r.City, r.Location),
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

	// Build the queue matrix
	var queue []TestResult
	for _, loc := range locationsToTest {
		for protocol, portsToTest := range config.ProtocolPorts {
			if len(portsToTest) == 0 {
				continue
			}
			for _, port := range portsToTest {
				queue = append(queue, TestResult{
					Location:           loc.Location,
					City:               loc.City,
					Country:            loc.Region,
					Continent:          loc.Continent,
					Protocol:           protocol,
					Port:               port,
					ProtocolTweaks:     config.ProtocolTweaks,
					AntiCensorshipMode: config.AmneziaConfig,
					ServerRouting:      config.ServerRouting,
					CustomSNI:          config.CustomSNI,
					LargeTLS:           config.LargeTLS,
					InternalDNS:        config.InternalDNS,
				})
			}
		}
	}

	sendUpdate(EventUpdate{
		Message: fmt.Sprintf("Queued %d tests across %d location(s).", len(queue), len(locationsToTest)),
		Status:  "Ready",
	})

	var completedResults []TestResult

	for i, task := range queue {
		if ctx.Err() != nil {
			sendUpdate(EventUpdate{Message: "Testing stopped by user.", Status: "Stopped"})
			return
		}

		// Step 1: Reset telemetry cards for next test
		sendUpdate(EventUpdate{
			ResetCards: true,
			Message:    fmt.Sprintf("[%d/%d] Preparing %s - %s (%s:%d)...", i+1, len(queue), task.Country, task.Location, task.Protocol, task.Port),
			Status:     "Connecting...",
		})

		o.worker.EnsureDisconnected(ctx)

		// Target connects by Location Nickname (e.g. "1984", "Seine")
		targetLoc := task.Location
		if strings.TrimSpace(targetLoc) == "" {
			targetLoc = task.City
		}

		err := o.worker.Connect(ctx, targetLoc, task.Protocol, task.Port)
		if err != nil {
			task.Status = "CLI_ERROR"
			o.db.RecordResult(task)
			completedResults = append(completedResults, task)
			sendUpdate(EventUpdate{
				Message: fmt.Sprintf("Error launching connection to %s: %v", targetLoc, err),
				Status:  "CLI Error",
			})
			continue
		}

		connected, duration, _ := o.worker.WaitForConnection(ctx, time.Duration(config.Timeout)*time.Second)
		task.Duration = duration

		if !connected {
			task.Status = "TIMEOUT"
			o.db.RecordResult(task)
			completedResults = append(completedResults, task)
			sendUpdate(EventUpdate{
				Message: fmt.Sprintf("Timeout connecting to %s (%s:%d) after %.1fs", targetLoc, task.Protocol, task.Port, duration),
				Status:  "Timeout",
			})
			continue
		}

		// Connected!
		task.Status = "SUCCESS"
		sendUpdate(EventUpdate{
			Message: fmt.Sprintf("Connected to %s in %.2fs! Waiting %ds...", targetLoc, duration, config.SpeedWait),
			Status:  fmt.Sprintf("Connected (%.2fs)", duration),
		})

		// Trigger background online resource refresh when connected
		if onConnected != nil {
			go onConnected()
		}

		time.Sleep(time.Duration(config.SpeedWait) * time.Second)

		// Speedtest and Latency measurements
		if config.TestPing || config.TestDownload || config.TestUpload {
			sendUpdate(EventUpdate{
				Message: "Finding nearest speedtest server...",
				Status:  "Finding Server...",
			})

			serverList, err := speedtest.FetchServers()
			var targets speedtest.Servers
			if err == nil {
				targets, _ = serverList.FindServer([]int{})
			}

			if len(targets) > 0 {
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
					Status:  "Saved",
				})
			} else {
				sendUpdate(EventUpdate{
					Message: "Speedtest server unavailable or blocked.",
					Status:  "Server N/A",
				})
			}
		}

		o.db.RecordResult(task)
		completedResults = append(completedResults, task)
		o.worker.Disconnect(ctx)
		time.Sleep(2 * time.Second)
	}

	// Automatic Excel export at end of testing
	var excelPath string
	if len(completedResults) > 0 {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		outName := fmt.Sprintf("Windscribe_Results_%s.xlsx", timestamp)
		if path, err := GenerateExcelReport(completedResults, outName); err == nil {
			excelPath = path
		}
	}

	sendUpdate(EventUpdate{
		Message:   "All queued tests completed successfully!",
		Status:    "Done",
		ExcelPath: excelPath,
	})
}

func (o *Orchestrator) Stop() {
	if o.cancel != nil {
		o.cancel()
		o.cancel = nil
	}
}
