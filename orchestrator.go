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
	Locations            []string         `json:"locations"`
	ProtocolPorts        map[string][]int `json:"protocolPorts"`
	CircumventCensorship bool             `json:"circumventCensorship"`
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
	Message string `json:"message"`
	Ping    string `json:"ping"`
	Down    string `json:"down"`
	Up      string `json:"up"`
	Status  string `json:"status"`
}

func (o *Orchestrator) StartTesting(parentCtx context.Context, config TestConfig, sendUpdate func(EventUpdate)) {
	ctx, cancel := context.WithCancel(parentCtx)
	o.cancel = cancel

	var queue []TestResult 
	
	if config.SelectionMode == "smart" {
		sendUpdate(EventUpdate{Message: "Calculating Smart Locations...", Status: "Analyzing"})
		best, err := o.db.GetBestLocations()
		if err == nil && len(best) > 0 {
			cityMap := make(map[string]bool)
			for _, r := range best {
				cityMap[r.City] = true
			}
			config.Locations = []string{}
			for c := range cityMap {
				config.Locations = append(config.Locations, c)
			}
		} else {
			sendUpdate(EventUpdate{Message: "No history found for Smart mode. Using provided locations.", Status: "Fallback"})
		}
	} else if config.SelectionMode == "discovery" {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(config.Locations), func(i, j int) { config.Locations[i], config.Locations[j] = config.Locations[j], config.Locations[i] })
		if len(config.Locations) > 5 {
			config.Locations = config.Locations[:5] // Test 5 random locations
		}
	}

	// Handle "all" locations if SelectionMode == "all"
	// Wait, we assume frontend provides the list of all locations for "all" mode,
	// or we can fetch it here. We'll let frontend pass it in Locations.

	// Build the queue matrix
	for _, city := range config.Locations {
		for protocol, portsToTest := range config.ProtocolPorts {
			if len(portsToTest) == 0 {
				continue
			}
			for _, port := range portsToTest {
				queue = append(queue, TestResult{
					City:               strings.TrimSpace(city),
					Protocol:             protocol,
					Port:                 port,
					AntiCensorshipMode:   fmt.Sprintf("%t", config.CircumventCensorship),
					LargeTLS:             config.LargeTLS,
					InternalDNS:          config.InternalDNS,
				})
			}
		}
	}

	sendUpdate(EventUpdate{Message: fmt.Sprintf("Queued %d tests.", len(queue)), Status: "Ready"})

	for i, task := range queue {
		if ctx.Err() != nil {
			sendUpdate(EventUpdate{Message: "Testing stopped by user.", Status: "Stopped"})
			return
		}

		sendUpdate(EventUpdate{
			Message: fmt.Sprintf("[%d/%d] Connecting to %s (%s:%d)...", i+1, len(queue), task.City, task.Protocol, task.Port),
			Status:  "Connecting",
		})

		o.worker.EnsureDisconnected(ctx)

		err := o.worker.Connect(ctx, task.City, task.Protocol, task.Port)
		if err != nil {
			task.Status = "CLI_ERROR"
			o.db.RecordResult(task)
			continue
		}

		connected, duration, _ := o.worker.WaitForConnection(ctx, time.Duration(config.Timeout)*time.Second)
		task.Duration = duration

		if !connected {
			task.Status = "TIMEOUT"
			o.db.RecordResult(task)
			sendUpdate(EventUpdate{Message: fmt.Sprintf("Timeout on %s", task.City), Status: "Timeout"})
			continue
		}

		task.Status = "SUCCESS"
		sendUpdate(EventUpdate{Message: fmt.Sprintf("Connected to %s in %.2fs. Waiting %ds...", task.City, duration, config.SpeedWait), Status: "Connected"})
		
		time.Sleep(time.Duration(config.SpeedWait) * time.Second)

		if config.TestPing || config.TestDownload || config.TestUpload {
			sendUpdate(EventUpdate{Message: "Finding speedtest server...", Status: "Speedtest"})
			serverList, _ := speedtest.FetchServers()
			targets, _ := serverList.FindServer([]int{})
			if len(targets) > 0 {
				s := targets[0]
				if config.TestPing {
					s.PingTest(nil)
					task.Ping = float64(s.Latency.Milliseconds())
				}
				if config.TestDownload {
					s.DownloadTest()
					task.Download = float64(s.DLSpeed) / 125000 // Convert bytes/s to Mbps
				}
				if config.TestUpload {
					s.UploadTest()
					task.Upload = float64(s.ULSpeed) / 125000 // Convert bytes/s to Mbps
				}
				sendUpdate(EventUpdate{
					Message: fmt.Sprintf("Speedtest Complete on %s", task.City),
					Ping:    fmt.Sprintf("%.0f ms", task.Ping),
					Down:    fmt.Sprintf("%.2f Mbps", task.Download),
					Up:      fmt.Sprintf("%.2f Mbps", task.Upload),
					Status:  "Saved",
				})
			}
		}

		o.db.RecordResult(task)
		o.worker.Disconnect(ctx)
		time.Sleep(3 * time.Second)
	}

	sendUpdate(EventUpdate{Message: "All tests finished.", Status: "Done"})
}

func (o *Orchestrator) Stop() {
	if o.cancel != nil {
		o.cancel()
		o.cancel = nil
	}
}
