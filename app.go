package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	goRuntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// LocationData represents a parsed location
type LocationData struct {
	Location    string `json:"location"`
	City        string `json:"city"`
	Region      string `json:"region"`
	CountryCode string `json:"countryCode"`
	Continent   string `json:"continent"`
	Display     string `json:"display"`
}

// AmneziaConfigItem represents an Amnezia preset
type AmneziaConfigItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ClientSettingsSnapshot represents current Windscribe client engine settings (read-only)
type ClientSettingsSnapshot struct {
	ProtocolTweaksMode string `json:"protocolTweaksMode"`
	AmneziaConfig      string `json:"amneziaConfig"`
	ServerRouting      string `json:"serverRouting"`
	CustomSNI          string `json:"customSni"`
	LargeTLS           bool   `json:"largeTls"`
	InternalDNS        string `json:"internalDns"`
}

var defaultAmneziaConfigs = []AmneziaConfigItem{
	{ID: "A", Name: "A - No Junk Russia Primary"},
	{ID: "B", Name: "B - No Junk Russia Alt"},
	{ID: "C", Name: "C - With Junk Russia Primary"},
	{ID: "D", Name: "D - With Junk Russia Alt"},
	{ID: "E", Name: "E - Baseline No Junk"},
	{ID: "F", Name: "F - Baseline With Junk"},
	{ID: "G", Name: "G - Popular Russia Website with Junk"},
	{ID: "H", Name: "H - Iran 26 May 2026"},
	{ID: "I", Name: "I - UAE and Iran Legacy"},
	{ID: "J", Name: "J - Iran Junk Packets"},
}

// App struct
type App struct {
	ctx               context.Context
	db                *DatabaseManager
	worker            *WindscribeWorker
	orchestrator      *Orchestrator
	pendingOnlineSync bool
	syncMu            sync.Mutex
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
	dbPath := getCachePath("windscribe_results.db")
	db, err := NewDatabaseManager(dbPath)
	if err != nil {
		fmt.Println("DB Init Error:", err)
	}
	a.db = db

	// Init Worker
	a.worker = NewWindscribeWorker()

	// Init Orchestrator
	a.orchestrator = NewOrchestrator(a.db, a.worker)

	// Automatically sanitize legacy database records if any exist
	if locs, err := a.GetAllLocations(); err == nil && len(locs) > 0 {
		a.db.SanitizeLegacyRecords(locs)
	}

	// Ensure local cache files exist beside the app on first run and attempt background refresh
	go func() {
		_, _ = a.fetchLocations(false)
		_, _ = a.fetchAmneziaConfigs(false)
		a.UpdateOnlineResources()
	}()
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
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "US", "CA", "MX", "PA", "SV", "GT", "CR", "HN", "NI", "BZ", "DO", "PR", "JM", "BS", "TT", "CU":
		return "North America"
	case "BR", "CO", "AR", "CL", "PY", "PE", "UY", "EC", "VE", "BO", "GY", "SR":
		return "South America"
	case "GB", "FR", "DE", "NL", "CH", "NO", "RO", "CZ", "IT", "ES", "SE", "IE", "DK", "PL", "AT", "HU", "BG", "LT", "LV", "EE", "FI", "GR", "PT", "RS", "SK", "SI", "HR", "BA", "CY", "IS", "MD", "MK", "AL", "GE", "BE", "LU", "RU", "UA", "MT", "MC", "AD", "SM", "LI", "BY", "ME", "AM", "AZ":
		return "Europe"
	case "JP", "SG", "HK", "KR", "IN", "MY", "ID", "PH", "VN", "TH", "TW", "KH", "AE", "IL", "TR", "KZ", "UZ", "PK", "BD", "LK", "NP", "MM", "LA", "MN", "QA", "BH", "KW", "OM", "JO", "LB", "SA", "IQ", "IR", "CN":
		return "Asia"
	case "AU", "NZ", "FJ", "PG":
		return "Oceania"
	case "ZA", "EG", "KE", "GH", "MA", "TN", "NG", "DZ", "ET", "TZ", "MU", "SN", "AO":
		return "Africa"
	case "AQ":
		return "Antarctica"
	default:
		return "Other"
	}
}

func getCachePath(filename string) string {
	if ex, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(ex), filename)
	}
	return filepath.Join(".", filename)
}

// GetAllLocations fetches available locations from live API or cached file
func (a *App) GetAllLocations() ([]LocationData, error) {
	return a.fetchLocations(false)
}

// RefreshLocations forces online fetch and updates cache
func (a *App) RefreshLocations() ([]LocationData, error) {
	locs, err := a.fetchLocations(true)
	if err != nil {
		a.pendingOnlineSync = true
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "sync_queued", "Online resources unreachable. Will auto-sync once a VPN tunnel connects.")
		}
		return a.fetchLocations(false)
	}
	return locs, nil
}

func (a *App) fetchLocations(forceOnline bool) ([]LocationData, error) {
	cachePath := getCachePath("cached_locations.json")

	// 1. If not forcing online, try reading cache first if it exists
	if !forceOnline {
		if cachedData, err := os.ReadFile(cachePath); err == nil {
			var cachedLocs []LocationData
			if err := json.Unmarshal(cachedData, &cachedLocs); err == nil && len(cachedLocs) > 0 {
				for i := range cachedLocs {
					if cachedLocs[i].CountryCode == "" {
						if code, ok := countryNameToCode[cachedLocs[i].Region]; ok {
							cachedLocs[i].CountryCode = code
						}
					}
					if cachedLocs[i].CountryCode != "" {
						cachedLocs[i].Continent = getContinent(cachedLocs[i].CountryCode)
					}
				}
				return cachedLocs, nil
			}
		}
	}

	// 2. Try online fetch
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get("https://assets.windscribe.com/serverlist/mob-v2/1/0")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var apiResp WindscribeAPIResponse
			if err := json.NewDecoder(resp.Body).Decode(&apiResp); err == nil {
				var locs []LocationData
				for _, country := range apiResp.Data {
					continent := getContinent(country.CountryCode)
					for _, group := range country.Groups {
						locs = append(locs, LocationData{
							Location:    group.Nick,
							City:        group.City,
							Region:      country.Name,
							CountryCode: country.CountryCode,
							Continent:   continent,
							Display:     fmt.Sprintf("%s - %s (%s)", country.Name, group.City, group.Nick),
						})
					}
				}
				if len(locs) > 0 {
					if data, err := json.MarshalIndent(locs, "", "  "); err == nil {
						_ = os.WriteFile(cachePath, data, 0644)
					}
					return locs, nil
				}
			}
		}
	}

	// 3. Fallback to cached file
	if cachedData, err := os.ReadFile(cachePath); err == nil {
		var cachedLocs []LocationData
		if err := json.Unmarshal(cachedData, &cachedLocs); err == nil && len(cachedLocs) > 0 {
			for i := range cachedLocs {
				if cachedLocs[i].CountryCode == "" {
					if code, ok := countryNameToCode[cachedLocs[i].Region]; ok {
						cachedLocs[i].CountryCode = code
					}
				}
			}
			return cachedLocs, nil
		}
	}

	return nil, fmt.Errorf("failed to fetch locations from network and no valid cache found")
}

// GetAmneziaConfigs gets configs from cache or returns defaults
func (a *App) GetAmneziaConfigs() ([]AmneziaConfigItem, error) {
	return a.fetchAmneziaConfigs(false)
}

// RefreshAmneziaConfigs forces refreshing amnezia configs online
func (a *App) RefreshAmneziaConfigs() ([]AmneziaConfigItem, error) {
	configs, err := a.fetchAmneziaConfigs(true)
	if err != nil {
		a.pendingOnlineSync = true
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "sync_queued", "Amnezia presets unreachable. Will auto-sync once a VPN tunnel connects.")
		}
		return a.fetchAmneziaConfigs(false)
	}
	return configs, nil
}

func (a *App) fetchAmneziaConfigs(forceOnline bool) ([]AmneziaConfigItem, error) {
	cachePath := getCachePath("cached_amnezia_configs.json")

	// 1. Always attempt to fetch live presets directly from local Windscribe CLI first
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if presets, err := a.worker.GetAmneziaPresets(ctx); err == nil && len(presets) > 0 {
		if data, err := json.MarshalIndent(presets, "", "  "); err == nil {
			_ = os.WriteFile(cachePath, data, 0644)
		}
		return presets, nil
	}

	// 2. Fallback to cached file if available
	if data, err := os.ReadFile(cachePath); err == nil {
		var cached []AmneziaConfigItem
		if err := json.Unmarshal(data, &cached); err == nil && len(cached) > 0 {
			return cached, nil
		}
	}

	// 3. Fallback to defaults
	if data, err := json.MarshalIndent(defaultAmneziaConfigs, "", "  "); err == nil {
		_ = os.WriteFile(cachePath, data, 0644)
	}

	return defaultAmneziaConfigs, nil
}

// UpdateOnlineResources updates both locations and configs in background once VPN is up
func (a *App) UpdateOnlineResources() {
	a.syncMu.Lock()
	defer a.syncMu.Unlock()

	// 1. Fetch live locations from Windscribe asset API
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get("https://assets.windscribe.com/serverlist/mob-v2/1/0")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var apiResp WindscribeAPIResponse
			if err := json.NewDecoder(resp.Body).Decode(&apiResp); err == nil {
				var locs []LocationData
				for _, country := range apiResp.Data {
					continent := getContinent(country.CountryCode)
					for _, group := range country.Groups {
						locs = append(locs, LocationData{
							Location:    group.Nick,
							City:        group.City,
							Region:      country.Name,
							CountryCode: country.CountryCode,
							Continent:   continent,
							Display:     fmt.Sprintf("%s - %s (%s)", country.Name, group.City, group.Nick),
						})
					}
				}
				if len(locs) > 0 {
					cachePath := getCachePath("cached_locations.json")
					if data, err := json.MarshalIndent(locs, "", "  "); err == nil {
						_ = os.WriteFile(cachePath, data, 0644)
					}
					if a.ctx != nil {
						runtime.EventsEmit(a.ctx, "locations_updated", locs)
					}
				}
			}
		}
	}

	// 2. Fetch live Amnezia presets from native Windscribe CLI or client
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	if presets, err := a.worker.GetAmneziaPresets(ctx); err == nil && len(presets) > 0 {
		cachePath := getCachePath("cached_amnezia_configs.json")
		if data, err := json.MarshalIndent(presets, "", "  "); err == nil {
			_ = os.WriteFile(cachePath, data, 0644)
		}
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "amnezia_updated", presets)
		}
	}

	a.pendingOnlineSync = false
}

// ExportExcelReport exports all historical test results into a styled Excel file
func (a *App) ExportExcelReport() (string, error) {
	results, err := a.db.GetAllResults()
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", fmt.Errorf("no test results recorded in database yet")
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	outName := getCachePath(fmt.Sprintf("Windscribe_Results_%s.xlsx", timestamp))
	return GenerateExcelReport(results, outName)
}

// OpenExcelFile opens the generated Excel file in default OS application
func (a *App) OpenExcelFile(filePath string) error {
	return openFileInOS(filePath)
}

// StartTest starts the testing routine
func (a *App) StartTest(config TestConfig) {
	go a.orchestrator.StartTesting(a.ctx, config, a.UpdateOnlineResources, func(update EventUpdate) {
		runtime.EventsEmit(a.ctx, "test_update", update)
	})
}

// StopTest cancels the testing routine
func (a *App) StopTest() {
	a.orchestrator.Stop()
}

// GetOS returns current host operating system (windows, linux, darwin)
func (a *App) GetOS() string {
	return goRuntime.GOOS
}

// ReadClientSettings reads the active Windscribe client settings in read-only mode (explicit user opt-in)
func (a *App) ReadClientSettings() (*ClientSettingsSnapshot, error) {
	return readPlatformClientSettings()
}
