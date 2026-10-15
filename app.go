package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// LocationData represents a parsed location
type LocationData struct {
	Location  string `json:"location"`
	City      string `json:"city"`
	Region    string `json:"region"`
	Continent string `json:"continent"`
	Display   string `json:"display"`
}


// App struct
type App struct {
	ctx          context.Context
	db           *DatabaseManager
	worker       *WindscribeWorker
	orchestrator *Orchestrator
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Init DB
	dbPath := filepath.Join(".", "windscribe_results.db")
	db, err := NewDatabaseManager(dbPath)
	if err != nil {
		fmt.Println("DB Init Error:", err)
	}
	a.db = db

	// Init Worker
	a.worker = NewWindscribeWorker()

	// Init Orchestrator
	a.orchestrator = NewOrchestrator(a.db, a.worker)
}

// GetLocations returns history stats to help frontend populate stuff
func (a *App) GetBestLocations() ([]TestResult, error) {
	return a.db.GetBestLocations()
}

// WindscribeAPIResponse represents the JSON structure of the Windscribe server list
type WindscribeAPIResponse struct {
	Data []struct {
		Name        string `json:"name"`
		CountryCode string `json:"country_code"`
		Groups      []struct {
			City string `json:"city"`
			Nick string `json:"nick"`
			Pro  int    `json:"pro"`
		} `json:"groups"`
	} `json:"data"`
}

func getContinent(code string) string {
	switch code {
	case "US", "CA", "MX", "PA":
		return "North America"
	case "BR", "CO", "AR", "CL":
		return "South America"
	case "GB", "FR", "DE", "NL", "CH", "NO", "RO", "CZ", "IT", "ES", "SE", "IE", "DK", "PL", "AT", "HU", "BG", "LT", "LV", "EE", "FI", "GR", "PT", "RS", "SK", "SI", "HR", "BA", "CY", "IS", "MD", "MK", "AL", "GE":
		return "Europe"
	case "JP", "SG", "HK", "KR", "IN", "MY", "ID", "PH", "VN", "TH", "TW", "KH", "AE", "IL", "TR":
		return "Asia"
	case "AU", "NZ":
		return "Oceania"
	case "ZA", "EG", "KE", "GH":
		return "Africa"
	case "AQ":
		return "Antarctica"
	default:
		return "Other"
	}
}

// GetAllLocations fetches available locations from the live API, cached JSON, or embedded CSV
func (a *App) GetAllLocations() ([]LocationData, error) {
	var locs []LocationData
	
	cachePath := "cached_locations.json"
	if ex, err := os.Executable(); err == nil {
		cachePath = filepath.Join(filepath.Dir(ex), "cached_locations.json")
	}

	// 1. Try to fetch from live API
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://assets.windscribe.com/serverlist/mob-v2/1/0")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var apiResp WindscribeAPIResponse
			if err := json.NewDecoder(resp.Body).Decode(&apiResp); err == nil {
				for _, country := range apiResp.Data {
					continent := getContinent(country.CountryCode)
					for _, group := range country.Groups {
						locs = append(locs, LocationData{
							Location:  group.Nick,
							City:      group.City,
							Region:    country.Name,
							Continent: continent,
							Display:   fmt.Sprintf("%s - %s (%s)", country.Name, group.City, group.Nick),
						})
					}
				}
				if len(locs) > 0 {
					// Save to cache asynchronously
					go func(l []LocationData) {
						if data, err := json.Marshal(l); err == nil {
							_ = os.WriteFile(cachePath, data, 0644)
						}
					}(locs)
					return locs, nil // Success from API
				}
			}
		}
	}

	// 2. Fallback to cached file if API fails
	if cachedData, err := os.ReadFile(cachePath); err == nil {
		var cachedLocs []LocationData
		if err := json.Unmarshal(cachedData, &cachedLocs); err == nil && len(cachedLocs) > 0 {
			return cachedLocs, nil // Success from Cache
		}
	}

	return nil, fmt.Errorf("failed to fetch locations from network and no valid cache found")
}

// StartTest starts the testing routine
func (a *App) StartTest(config TestConfig) {
	// Run in goroutine so it doesn't block UI
	go a.orchestrator.StartTesting(a.ctx, config, func(update EventUpdate) {
		runtime.EventsEmit(a.ctx, "test_update", update)
	})
}

// StopTest cancels the testing routine
func (a *App) StopTest() {
	a.orchestrator.Stop()
}
