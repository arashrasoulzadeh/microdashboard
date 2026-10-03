package integration_test

import (
	_ "github.com/mattn/go-sqlite3"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/julienschmidt/httprouter"
	"microdashboard/internal/auth"
	"microdashboard/internal/config"
	"microdashboard/internal/store"
	"microdashboard/internal/ui"
)

func setupTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db: %v", err)
	}
	tmpfile.Close()

	cfg := config.Load()
	cfg.DBPath = tmpfile.Name()

	st := store.Open(cfg.DBPath)
	if err := st.Migrate(); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	router := httprouter.New()
	authMiddleware := auth.New(st)
	ui.SetupRoutes(router, st, authMiddleware)

	server := httptest.NewServer(router)

	t.Cleanup(func() {
		server.Close()
		st.Close()
		os.Remove(tmpfile.Name())
	})

	return server, st
}

func TestUI_Index(t *testing.T) {
	server, _ := setupTestServer(t)

	req, _ := http.NewRequest("GET", server.URL+"/ui/?token=admin-change-me", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestUI_StaticFiles(t *testing.T) {
	server, _ := setupTestServer(t)

	// Test CSS
	resp, err := http.Get(server.URL + "/ui/static/style.css")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for CSS, got %d", resp.StatusCode)
	}

	// Test JS
	resp, err = http.Get(server.URL + "/ui/static/app.js")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for JS, got %d", resp.StatusCode)
	}
}

func TestUI_ApiDashboards(t *testing.T) {
	server, _ := setupTestServer(t)

	// Create a dashboard
	dashboard := map[string]interface{}{
		"id":              "test-dash",
		"name":            "Test Dashboard",
		"width":           128,
		"height":          64,
		"refresh_interval": 10000,
		"widgets":         []interface{}{},
	}
	body, _ := json.Marshal(dashboard)

	req, _ := http.NewRequest("POST", server.URL+"/api/dashboards?token=admin-change-me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	// List dashboards
	req, _ = http.NewRequest("GET", server.URL+"/api/dashboards?token=admin-change-me", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var dashboards []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&dashboards)
	if len(dashboards) != 1 {
		t.Errorf("Expected 1 dashboard, got %d", len(dashboards))
	}
}

func TestUI_ApiDevices(t *testing.T) {
	server, _ := setupTestServer(t)

	// Create a device
	device := map[string]interface{}{
		"device_id":    "test-device",
		"api_key":      "test-key",
		"dashboard_id": "test-dash",
	}
	body, _ := json.Marshal(device)

	req, _ := http.NewRequest("POST", server.URL+"/api/devices?token=admin-change-me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	// List devices
	req, _ = http.NewRequest("GET", server.URL+"/api/devices?token=admin-change-me", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestUI_ApiMonitors(t *testing.T) {
	server, _ := setupTestServer(t)

	// Create a monitor
	monitor := map[string]interface{}{
		"id":      "test-monitor",
		"url":     "http://example.com",
		"method":  "GET",
		"timeout": 5000,
	}
	body, _ := json.Marshal(monitor)

	req, _ := http.NewRequest("POST", server.URL+"/api/monitors?token=admin-change-me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected 201, got %d, body: %s", resp.StatusCode, string(respBody))
		return
	}

	// List monitors
	req, _ = http.NewRequest("GET", server.URL+"/api/monitors?token=admin-change-me", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	respBody, _ = io.ReadAll(resp.Body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d, body: %s", resp.StatusCode, string(respBody))
		return
	}

	var monitors []map[string]interface{}
	json.Unmarshal(respBody, &monitors)
	if len(monitors) != 1 {
		t.Errorf("Expected 1 monitor, got %d", len(monitors))
	}
}