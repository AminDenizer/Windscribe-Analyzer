package main

import (
	"database/sql"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type DatabaseManager struct {
	db *sql.DB
}

type TestResult struct {
	ID                 int     `json:"id"`
	Location           string  `json:"location"`
	City               string  `json:"city"`
	Country            string  `json:"country"`
	Continent          string  `json:"continent"`
	Protocol           string  `json:"protocol"`
	Port               int     `json:"port"`
	Status             string  `json:"status"`
	Duration           float64 `json:"duration"`
	Ping               float64 `json:"ping"`
	Jitter             float64 `json:"jitter"`
	Download           float64 `json:"download"`
	Upload             float64 `json:"upload"`
	ProfileName        string  `json:"profileName"`
	ProtocolTweaks     bool    `json:"protocolTweaks"`
	AntiCensorshipMode string  `json:"antiCensorshipMode"`
	ServerRouting      string  `json:"serverRouting"`
	CustomSNI          string  `json:"customSni"`
	LargeTLS           bool    `json:"largeTls"`
	InternalDNS        string  `json:"internalDns"`
	ISP                string  `json:"isp"`
	Timestamp          string  `json:"timestamp"`
}

func NewDatabaseManager(dbPath string) (*DatabaseManager, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	manager := &DatabaseManager{db: db}
	err = manager.createTableAndMigrate()
	if err != nil {
		return nil, err
	}

	return manager, nil
}

func (m *DatabaseManager) createTableAndMigrate() error {
	query := `
		CREATE TABLE IF NOT EXISTS tests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			location TEXT,
			city TEXT,
			country TEXT,
			continent TEXT DEFAULT '',
			protocol TEXT,
			port INTEGER,
			status TEXT,
			duration REAL,
			ping REAL,
			jitter REAL DEFAULT 0,
			download REAL,
			upload REAL,
			profile_name TEXT,
			protocol_tweaks BOOLEAN DEFAULT 1,
			anti_censorship_mode TEXT,
			server_routing TEXT DEFAULT 'Auto',
			custom_sni TEXT DEFAULT '',
			large_tls BOOLEAN,
			internal_dns TEXT,
			isp TEXT DEFAULT '',
			timestamp TEXT,
			UNIQUE(location, city, country, protocol, port, anti_censorship_mode, server_routing, custom_sni, large_tls, internal_dns, isp)
		)
	`
	if _, err := m.db.Exec(query); err != nil {
		return err
	}

	// Schema migrations for existing databases
	migrations := []string{
		"ALTER TABLE tests ADD COLUMN continent TEXT DEFAULT ''",
		"ALTER TABLE tests ADD COLUMN jitter REAL DEFAULT 0",
		"ALTER TABLE tests ADD COLUMN protocol_tweaks BOOLEAN DEFAULT 1",
		"ALTER TABLE tests ADD COLUMN server_routing TEXT DEFAULT 'Auto'",
		"ALTER TABLE tests ADD COLUMN custom_sni TEXT DEFAULT ''",
		"ALTER TABLE tests ADD COLUMN isp TEXT DEFAULT ''",
	}

	for _, mig := range migrations {
		_, _ = m.db.Exec(mig) // Ignore error if column already exists
	}

	return nil
}

func (m *DatabaseManager) RecordResult(res TestResult) error {
	query := `
		INSERT OR REPLACE INTO tests 
		(location, city, country, continent, protocol, port, status, duration, ping, jitter, download, upload, profile_name, protocol_tweaks, anti_censorship_mode, server_routing, custom_sni, large_tls, internal_dns, isp, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	ts := res.Timestamp
	if strings.TrimSpace(ts) == "" {
		ts = time.Now().Format("2006-01-02 15:04:05")
	}

	_, err := m.db.Exec(query,
		res.Location, res.City, res.Country, res.Continent, res.Protocol, res.Port, res.Status,
		res.Duration, res.Ping, res.Jitter, res.Download, res.Upload,
		res.ProfileName, res.ProtocolTweaks, res.AntiCensorshipMode,
		res.ServerRouting, res.CustomSNI, res.LargeTLS, res.InternalDNS, res.ISP, ts)
	return err
}

func (m *DatabaseManager) GetRowCount() (int, error) {
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM tests").Scan(&count)
	return count, err
}

func (m *DatabaseManager) ClearDB() error {
	_, err := m.db.Exec("DELETE FROM tests")
	return err
}

// SanitizeLegacyRecords repairs old database records where location/country/continent were blank
// and replaces boolean "true" anti_censorship_mode with clean preset names
func (m *DatabaseManager) SanitizeLegacyRecords(locs []LocationData) {
	if m.db == nil || len(locs) == 0 {
		return
	}

	lookup := make(map[string]LocationData)
	for _, l := range locs {
		lookup[strings.ToLower(l.Location)] = l
		lookup[strings.ToLower(l.City)] = l
	}

	rows, err := m.db.Query("SELECT id, location, city, country, continent, anti_censorship_mode FROM tests")
	if err != nil {
		return
	}
	defer rows.Close()

	type toUpdate struct {
		id                                      int
		loc, city, country, continent, antiMode string
	}
	var updates []toUpdate

	for rows.Next() {
		var id int
		var loc, city, country, continent, antiMode string
		if err := rows.Scan(&id, &loc, &city, &country, &continent, &antiMode); err == nil {
			changed := false

			// If country is missing or location is blank, resolve from lookup
			if country == "" || loc == "" || continent == "" {
				key := strings.ToLower(city)
				if key == "" {
					key = strings.ToLower(loc)
				}
				if match, ok := lookup[key]; ok {
					loc = match.Location
					city = match.City
					country = match.Region
					continent = match.Continent
					changed = true
				}
			}

			if antiMode == "true" {
				antiMode = "Legacy Tweak"
				changed = true
			} else if antiMode == "false" || antiMode == "" {
				antiMode = "None"
				changed = true
			}

			if changed {
				updates = append(updates, toUpdate{id, loc, city, country, continent, antiMode})
			}
		}
	}

	for _, u := range updates {
		_, _ = m.db.Exec("UPDATE tests SET location=?, city=?, country=?, continent=?, anti_censorship_mode=? WHERE id=?",
			u.loc, u.city, u.country, u.continent, u.antiMode, u.id)
	}
}

func (m *DatabaseManager) GetAllResults() ([]TestResult, error) {
	query := `SELECT id, location, city, country, continent, protocol, port, status, duration, ping, jitter, download, upload, profile_name, protocol_tweaks, anti_censorship_mode, server_routing, custom_sni, large_tls, internal_dns, isp, timestamp 
	          FROM tests ORDER BY timestamp DESC`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TestResult
	for rows.Next() {
		var r TestResult
		err := rows.Scan(
			&r.ID, &r.Location, &r.City, &r.Country, &r.Continent,
			&r.Protocol, &r.Port, &r.Status, &r.Duration, &r.Ping,
			&r.Jitter, &r.Download, &r.Upload, &r.ProfileName,
			&r.ProtocolTweaks, &r.AntiCensorshipMode, &r.ServerRouting,
			&r.CustomSNI, &r.LargeTLS, &r.InternalDNS, &r.ISP, &r.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

func (m *DatabaseManager) GetBestLocations() ([]TestResult, error) {
	query := `SELECT id, location, city, country, continent, protocol, port, status, duration, ping, jitter, download, upload, profile_name, protocol_tweaks, anti_censorship_mode, server_routing, custom_sni, large_tls, internal_dns, isp, timestamp 
	          FROM tests WHERE status='SUCCESS' ORDER BY download DESC, ping ASC LIMIT 50`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TestResult
	for rows.Next() {
		var r TestResult
		err := rows.Scan(
			&r.ID, &r.Location, &r.City, &r.Country, &r.Continent,
			&r.Protocol, &r.Port, &r.Status, &r.Duration, &r.Ping,
			&r.Jitter, &r.Download, &r.Upload, &r.ProfileName,
			&r.ProtocolTweaks, &r.AntiCensorshipMode, &r.ServerRouting,
			&r.CustomSNI, &r.LargeTLS, &r.InternalDNS, &r.ISP, &r.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}
