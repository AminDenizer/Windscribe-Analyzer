package main

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type DatabaseManager struct {
	db *sql.DB
}

type TestResult struct {
	ID                 int
	Location           string
	City               string
	Country            string
	Protocol           string
	Port               int
	Status             string
	Duration           float64
	Ping               float64
	Download           float64
	Upload             float64
	ProfileName        string
	AntiCensorshipMode string
	LargeTLS           bool
	InternalDNS        string
	Timestamp          string
}

func NewDatabaseManager(dbPath string) (*DatabaseManager, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	manager := &DatabaseManager{db: db}
	err = manager.createTable()
	if err != nil {
		return nil, err
	}

	return manager, nil
}

func (m *DatabaseManager) createTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS tests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			location TEXT,
			city TEXT,
			country TEXT,
			protocol TEXT,
			port INTEGER,
			status TEXT,
			duration REAL,
			ping REAL,
			download REAL,
			upload REAL,
			profile_name TEXT,
			anti_censorship_mode TEXT,
			large_tls BOOLEAN,
			internal_dns TEXT,
			timestamp TEXT,
			UNIQUE(location, city, country, protocol, port, anti_censorship_mode, large_tls, internal_dns)
		)
	`
	_, err := m.db.Exec(query)
	return err
}

func (m *DatabaseManager) RecordResult(res TestResult) error {
	query := `
		INSERT OR REPLACE INTO tests 
		(location, city, country, protocol, port, status, duration, ping, download, upload, profile_name, anti_censorship_mode, large_tls, internal_dns, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	ts := time.Now().Format("2006-01-02 15:04:05")
	_, err := m.db.Exec(query,
		res.Location, res.City, res.Country, res.Protocol, res.Port, res.Status, res.Duration, res.Ping, res.Download, res.Upload, res.ProfileName, res.AntiCensorshipMode, res.LargeTLS, res.InternalDNS, ts)
	return err
}

func (m *DatabaseManager) GetRowCount() (int, error) {
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM tests").Scan(&count)
	return count, err
}

func (m *DatabaseManager) IsTested(city, protocol string, port int, acMode string, tls bool, dns string) (bool, error) {
	var id int
	err := m.db.QueryRow(`
		SELECT id FROM tests 
		WHERE city=? AND protocol=? AND port=? AND anti_censorship_mode=? AND large_tls=? AND internal_dns=?`,
		city, protocol, port, acMode, tls, dns).Scan(&id)

	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (m *DatabaseManager) ClearDB() error {
	_, err := m.db.Exec("DELETE FROM tests")
	return err
}

func (m *DatabaseManager) GetBestLocations() ([]TestResult, error) {
	query := `SELECT location, city, country, protocol, port, status, duration, ping, download, upload, anti_censorship_mode, large_tls, internal_dns 
	          FROM tests WHERE status='SUCCESS' ORDER BY download DESC, ping ASC LIMIT 50`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TestResult
	for rows.Next() {
		var r TestResult
		err := rows.Scan(&r.Location, &r.City, &r.Country, &r.Protocol, &r.Port, &r.Status, &r.Duration, &r.Ping, &r.Download, &r.Upload, &r.AntiCensorshipMode, &r.LargeTLS, &r.InternalDNS)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}
