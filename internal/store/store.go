package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type Store struct {
	db *sql.DB
}

func Open(path string) *Store {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	return &Store{db: db}
}

func (s *Store) Close() {
	s.db.Close()
}

// DB returns the underlying database connection for testing.
// This is only for testing purposes.
func (s *Store) DB() *sql.DB {
	return s.db
}

// CreateTempDB creates a temporary database file for testing.
// Returns the path to the temp file.
// Caller is responsible for cleaning up the file.
func CreateTempDB(t interface{ Fatalf(string, ...interface{}) }) string {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db: %v", err)
	}
	tmpfile.Close()
	return tmpfile.Name()
}

func (s *Store) Migrate() error {
	const queryDevices = `CREATE TABLE IF NOT EXISTS devices (
		device_id TEXT PRIMARY KEY,
		api_key_hash TEXT NOT NULL,
		dashboard_id TEXT,
		created_at INTEGER NOT NULL
	);`

	const queryDashboards = `CREATE TABLE IF NOT EXISTS dashboards (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		json_definition TEXT NOT NULL,
		width INTEGER NOT NULL,
		height INTEGER NOT NULL,
		refresh_interval INTEGER NOT NULL DEFAULT 15000,
		created_at INTEGER NOT NULL
	);`

	const queryLatencyMonitors = `CREATE TABLE IF NOT EXISTS latency_monitors (
		id TEXT PRIMARY KEY,
		url TEXT NOT NULL,
		method TEXT NOT NULL DEFAULT 'GET',
		timeout INTEGER NOT NULL DEFAULT 5000,
		last_elapsed_ms INTEGER,
		last_status INTEGER,
		last_checked INTEGER NOT NULL
	);`

	const queryMetrics = `CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id TEXT,
		target_id TEXT,
		elapsed_ms INTEGER,
		status INTEGER,
		body_bytes INTEGER,
		checked_at INTEGER NOT NULL
	);`

	for _, q := range []string{
		queryDevices,
		queryDashboards,
		queryLatencyMonitors,
		queryMetrics,
	} {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("migration failed for query %q: %w", q, err)
		}
	}
	return nil
}

// --- Device methods ---

func (s *Store) UpsertDevice(deviceID, apiKeyHash string, dashboardID *string) error {
	now := time.Now().Unix()
	if dashboardID != nil {
		_, err := s.db.Exec(`INSERT INTO devices (device_id, api_key_hash, dashboard_id, created_at) VALUES (?, ?, ?, ?) ON CONFLICT(device_id) DO UPDATE SET api_key_hash = ?, dashboard_id = ?, created_at = ?`,
			deviceID, apiKeyHash, *dashboardID, now, apiKeyHash, *dashboardID, now)
		return err
	}
	_, err := s.db.Exec(`INSERT INTO devices (device_id, api_key_hash, created_at) VALUES (?, ?, ?) ON CONFLICT(device_id) DO UPDATE SET api_key_hash = ?, created_at = ?`,
		deviceID, apiKeyHash, now, apiKeyHash, now)
	return err
}

func (s *Store) GetDevice(deviceID string) (*DeviceRow, error) {
	row := s.db.QueryRow(`SELECT device_id, api_key_hash, dashboard_id, created_at FROM devices WHERE device_id = ?`, deviceID)
	r := &DeviceRow{}
	err := row.Scan(&r.DeviceID, &r.APIKeyHash, &r.DashboardID, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// GetDeviceByKey looks up a device by its raw API key.
func (s *Store) GetDeviceByKey(key string) (*DeviceRow, error) {
	row := s.db.QueryRow(`SELECT device_id, api_key_hash, dashboard_id, created_at FROM devices WHERE api_key_hash = ?`, key)
	r := &DeviceRow{}
	err := row.Scan(&r.DeviceID, &r.APIKeyHash, &r.DashboardID, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid API key: %s", key)
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

type DeviceRow struct {
	DeviceID   string
	APIKeyHash string
	DashboardID *string
	CreatedAt  int64
}

// --- Dashboard methods ---

func (s *Store) UpsertDashboard(id, name, jsonDef string, width, height int, refreshInterval int) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(`INSERT INTO dashboards (id, name, json_definition, width, height, refresh_interval, created_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = ?, json_definition = ?, width = ?, height = ?, refresh_interval = ?, created_at = ?`,
		id, name, jsonDef, width, height, refreshInterval, now, name, jsonDef, width, height, refreshInterval, now)
	return err
}

func (s *Store) GetDashboard(id string) (*DashboardRow, error) {
	row := s.db.QueryRow(`SELECT id, name, json_definition, width, height, refresh_interval, created_at FROM dashboards WHERE id = ?`, id)
	r := &DashboardRow{}
	err := row.Scan(&r.ID, &r.Name, &r.JSONDef, &r.Width, &r.Height, &r.RefreshInterval, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("dashboard not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Store) DeleteDashboard(id string) error {
	_, err := s.db.Exec(`DELETE FROM dashboards WHERE id = ?`, id)
	return err
}

type DashboardRow struct {
	ID            string
	Name          string
	JSONDef       string
	Width         int
	Height        int
	RefreshInterval int
	CreatedAt    int64
}

// --- Latency monitor methods ---

func (s *Store) UpsertMonitor(id, url, method string, timeout int) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(`INSERT INTO latency_monitors (id, url, method, timeout, last_checked) VALUES (?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET url = ?, method = ?, timeout = ?, last_checked = ?`,
		id, url, method, timeout, now, url, method, timeout, now)
	return err
}

func (s *Store) GetMonitor(id string) (*MonitorRow, error) {
	row := s.db.QueryRow(`SELECT id, url, method, timeout, COALESCE(last_elapsed_ms, 0), COALESCE(last_status, 0), last_checked FROM latency_monitors WHERE id = ?`, id)
	r := &MonitorRow{}
	err := row.Scan(&r.ID, &r.URL, &r.Method, &r.Timeout, &r.LastElapsedMs, &r.LastStatus, &r.LastChecked)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("monitor not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Store) GetAllMonitors() ([]MonitorRow, error) {
	rows, err := s.db.Query(`SELECT id, url, method, timeout, COALESCE(last_elapsed_ms, 0), COALESCE(last_status, 0), last_checked FROM latency_monitors ORDER BY last_checked DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []MonitorRow
	for rows.Next() {
		var r MonitorRow
		if err := rows.Scan(&r.ID, &r.URL, &r.Method, &r.Timeout, &r.LastElapsedMs, &r.LastStatus, &r.LastChecked); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

func (s *Store) UpdateMonitorResult(id string, elapsedMs int64, status int) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(`UPDATE latency_monitors SET last_elapsed_ms = ?, last_status = ?, last_checked = ? WHERE id = ?`, elapsedMs, status, now, id)
	return err
}

type MonitorRow struct {
	ID         string
	URL        string
	Method     string
	Timeout    int
	LastElapsedMs   int64
	LastStatus   int64
	LastChecked int64
}

// --- Metrics methods ---

func (s *Store) InsertMetric(deviceID, targetID string, elapsedMS, status, bodyBytes int) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(`INSERT INTO metrics (device_id, target_id, elapsed_ms, status, body_bytes, checked_at) VALUES (?, ?, ?, ?, ?, ?)`,
		deviceID, targetID, elapsedMS, status, bodyBytes, now)
	return err
}

func (s *Store) GetMetrics(limit int) ([]MetricRow, error) {
	rows, err := s.db.Query(`SELECT id, device_id, target_id, elapsed_ms, status, body_bytes, checked_at FROM metrics ORDER BY checked_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []MetricRow
	for rows.Next() {
		var r MetricRow
		if err := rows.Scan(&r.ID, &r.DeviceID, &r.TargetID, &r.ElapsedMs, &r.Status, &r.BodyBytes, &r.CheckedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

func (s *Store) GetMetricsByDevice(deviceID string, limit int) ([]MetricRow, error) {
	rows, err := s.db.Query(
		`SELECT id, device_id, target_id, elapsed_ms, status, body_bytes, checked_at 
		 FROM metrics WHERE device_id = ? ORDER BY checked_at DESC LIMIT ?`,
		deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []MetricRow
	for rows.Next() {
		var r MetricRow
		if err := rows.Scan(&r.ID, &r.DeviceID, &r.TargetID, &r.ElapsedMs, &r.Status, &r.BodyBytes, &r.CheckedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

type MetricRow struct {
	ID        int
	DeviceID  string
	TargetID  string
	ElapsedMs int64
	Status    int64
	BodyBytes int64
	CheckedAt int64
}

func (s *Store) GetAdminToken() string {
	// In production, this would come from config/env
	return "admin-change-me"
}

// --- UI helper methods ---

type DashboardSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	RefreshInterval int    `json:"refresh_interval"`
	WidgetCount     int    `json:"widget_count"`
	CreatedAt       int64  `json:"created_at"`
}

func (s *Store) ListDashboards() ([]DashboardSummary, error) {
	rows, err := s.db.Query(`SELECT id, name, json_definition, width, height, refresh_interval, created_at FROM dashboards ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DashboardSummary
	for rows.Next() {
		var d DashboardSummary
		var jsonDef string
		if err := rows.Scan(&d.ID, &d.Name, &jsonDef, &d.Width, &d.Height, &d.RefreshInterval, &d.CreatedAt); err != nil {
			return nil, err
		}
		var widgets []interface{}
		json.Unmarshal([]byte(jsonDef), &widgets)
		d.WidgetCount = len(widgets)
		results = append(results, d)
	}
	return results, nil
}

type DeviceSummary struct {
	DeviceID   string  `json:"device_id"`
	DashboardID *string `json:"dashboard_id"`
	CreatedAt  int64   `json:"created_at"`
}

func (s *Store) ListDevices() ([]DeviceSummary, error) {
	rows, err := s.db.Query(`SELECT device_id, dashboard_id, created_at FROM devices ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DeviceSummary
	for rows.Next() {
		var d DeviceSummary
		if err := rows.Scan(&d.DeviceID, &d.DashboardID, &d.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, d)
	}
	return results, nil
}

func (s *Store) DeleteDevice(deviceID string) error {
	_, err := s.db.Exec(`DELETE FROM devices WHERE device_id = ?`, deviceID)
	return err
}

func (s *Store) AssignDeviceDashboard(deviceID, dashboardID string) error {
	_, err := s.db.Exec(`UPDATE devices SET dashboard_id = ? WHERE device_id = ?`, dashboardID, deviceID)
	return err
}

type MonitorSummary struct {
	ID            string `json:"id"`
	URL           string `json:"url"`
	Method        string `json:"method"`
	Timeout       int    `json:"timeout"`
	LastElapsedMs int64  `json:"last_elapsed_ms"`
	LastStatus    int64  `json:"last_status"`
	LastChecked   int64  `json:"last_checked"`
}

func (s *Store) ListMonitors() ([]MonitorSummary, error) {
	rows, err := s.db.Query(`SELECT id, url, method, timeout, COALESCE(last_elapsed_ms, 0), COALESCE(last_status, 0), last_checked FROM latency_monitors ORDER BY last_checked DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MonitorSummary
	for rows.Next() {
		var m MonitorSummary
		if err := rows.Scan(&m.ID, &m.URL, &m.Method, &m.Timeout, &m.LastElapsedMs, &m.LastStatus, &m.LastChecked); err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, nil
}

func (s *Store) DeleteMonitor(id string) error {
	_, err := s.db.Exec(`DELETE FROM latency_monitors WHERE id = ?`, id)
	return err
}