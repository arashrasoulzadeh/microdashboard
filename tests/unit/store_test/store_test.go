package store_test

import (
	_ "github.com/mattn/go-sqlite3"
	"os"
	"testing"

	"microdashboard/internal/store"
)

func setupTestDB(t *testing.T) *store.Store {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db: %v", err)
	}
	tmpfile.Close()

	s := store.Open(tmpfile.Name())
	if err := s.Migrate(); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	t.Cleanup(func() {
		s.Close()
		os.Remove(tmpfile.Name())
	})
	return s
}

func TestStore_Migrate(t *testing.T) {
	s := setupTestDB(t)

	tables := []string{"devices", "dashboards", "latency_monitors", "metrics"}
	for _, table := range tables {
		var count int
		err := s.DB().QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err != nil {
			t.Errorf("table %s not accessible: %v", table, err)
		}
	}
}

func TestStore_UpsertAndGetDevice(t *testing.T) {
	s := setupTestDB(t)

	err := s.UpsertDevice("test-device", "api-key-123", nil)
	if err != nil {
		t.Fatalf("UpsertDevice failed: %v", err)
	}

	d, err := s.GetDevice("test-device")
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	if d.DeviceID != "test-device" {
		t.Errorf("DeviceID = %q, want %q", d.DeviceID, "test-device")
	}
	if d.APIKeyHash != "api-key-123" {
		t.Errorf("APIKeyHash = %q, want %q", d.APIKeyHash, "api-key-123")
	}
	if d.DashboardID != nil {
		t.Errorf("DashboardID should be nil, got %v", *d.DashboardID)
	}

	dashID := "dash-1"
	if err := s.UpsertDevice("test-device", "new-key-456", &dashID); err != nil {
		t.Fatalf("UpsertDevice update failed: %v", err)
	}

	d, err = s.GetDevice("test-device")
	if err != nil {
		t.Fatalf("GetDevice after update failed: %v", err)
	}
	if d.APIKeyHash != "new-key-456" {
		t.Errorf("APIKeyHash not updated: %q", d.APIKeyHash)
	}
	if d.DashboardID == nil || *d.DashboardID != "dash-1" {
		t.Errorf("DashboardID not updated: %v", d.DashboardID)
	}
}

func TestStore_GetDeviceByKey(t *testing.T) {
	s := setupTestDB(t)

	s.UpsertDevice("dev1", "key-abc", nil)

	d, err := s.GetDeviceByKey("key-abc")
	if err != nil {
		t.Fatalf("GetDeviceByKey failed: %v", err)
	}
	if d.DeviceID != "dev1" {
		t.Errorf("DeviceID = %q, want %q", d.DeviceID, "dev1")
	}

	_, err = s.GetDeviceByKey("nonexistent")
	if err == nil {
		t.Error("GetDeviceByKey should fail for non-existent key")
	}
}

func TestStore_DashboardCRUD(t *testing.T) {
	s := setupTestDB(t)

	widgetsJSON := `[{"id":"w1","type":"gauge","x":0,"y":0,"width":32,"height":16,"expression":"${latency:monitor1}","unit":"ms"}]`

	err := s.UpsertDashboard("dash1", "Test Dashboard", widgetsJSON, 128, 64, 15000)
	if err != nil {
		t.Fatalf("UpsertDashboard failed: %v", err)
	}

	d, err := s.GetDashboard("dash1")
	if err != nil {
		t.Fatalf("GetDashboard failed: %v", err)
	}
	if d.Name != "Test Dashboard" {
		t.Errorf("Name = %q, want %q", d.Name, "Test Dashboard")
	}
	if d.JSONDef != widgetsJSON {
		t.Errorf("JSONDef mismatch")
	}

	err = s.UpsertDashboard("dash1", "Updated Dashboard", widgetsJSON, 128, 64, 15000)
	if err != nil {
		t.Fatalf("UpsertDashboard update failed: %v", err)
	}

	d, err = s.GetDashboard("dash1")
	if err != nil {
		t.Fatalf("GetDashboard after update failed: %v", err)
	}
	if d.Name != "Updated Dashboard" {
		t.Errorf("Name not updated: %q", d.Name)
	}

	err = s.DeleteDashboard("dash1")
	if err != nil {
		t.Fatalf("DeleteDashboard failed: %v", err)
	}

	_, err = s.GetDashboard("dash1")
	if err == nil {
		t.Error("GetDashboard should fail after delete")
	}
}

func TestStore_MonitorCRUD(t *testing.T) {
	s := setupTestDB(t)

	err := s.UpsertMonitor("mon1", "http://example.com", "GET", 5000)
	if err != nil {
		t.Fatalf("UpsertMonitor failed: %v", err)
	}

	m, err := s.GetMonitor("mon1")
	if err != nil {
		t.Fatalf("GetMonitor failed: %v", err)
	}
	if m.URL != "http://example.com" {
		t.Errorf("URL = %q, want %q", m.URL, "http://example.com")
	}

	err = s.UpdateMonitorResult("mon1", 123, 200)
	if err != nil {
		t.Fatalf("UpdateMonitorResult failed: %v", err)
	}

	m, err = s.GetMonitor("mon1")
	if err != nil {
		t.Fatalf("GetMonitor after update failed: %v", err)
	}
	if m.LastElapsedMs != 123 {
		t.Errorf("LastElapsedMs = %d, want 123", m.LastElapsedMs)
	}
	if m.LastStatus != 200 {
		t.Errorf("LastStatus = %d, want 200", m.LastStatus)
	}

	monitors, err := s.ListMonitors()
	if err != nil {
		t.Fatalf("ListMonitors failed: %v", err)
	}
	if len(monitors) != 1 {
		t.Errorf("ListMonitors count = %d, want 1", len(monitors))
	}

	err = s.DeleteMonitor("mon1")
	if err != nil {
		t.Fatalf("DeleteMonitor failed: %v", err)
	}

	_, err = s.GetMonitor("mon1")
	if err == nil {
		t.Error("GetMonitor should fail after delete")
	}
}

func TestStore_Metrics(t *testing.T) {
	s := setupTestDB(t)

	s.InsertMetric("dev1", "mon1", 100, 200, 512)
	s.InsertMetric("dev1", "mon1", 150, 200, 256)
	s.InsertMetric("dev2", "mon2", 200, 500, 1024)

	metrics, err := s.GetMetrics(10)
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}
	if len(metrics) != 3 {
		t.Errorf("GetMetrics count = %d, want 3", len(metrics))
	}

	devMetrics, err := s.GetMetricsByDevice("dev1", 10)
	if err != nil {
		t.Fatalf("GetMetricsByDevice failed: %v", err)
	}
	if len(devMetrics) != 2 {
		t.Errorf("GetMetricsByDevice count = %d, want 2", len(devMetrics))
	}
}

func TestStore_DeviceList(t *testing.T) {
	s := setupTestDB(t)

	s.UpsertDevice("dev1", "key1", nil)
	s.UpsertDevice("dev2", "key2", func() *string { s := "dash1"; return &s }())

	devices, err := s.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("ListDevices count = %d, want 2", len(devices))
	}
}

func TestStore_DashboardList(t *testing.T) {
	s := setupTestDB(t)

	widgetsJSON := `[]`
	s.UpsertDashboard("dash1", "Dash 1", widgetsJSON, 128, 64, 15000)
	s.UpsertDashboard("dash2", "Dash 2", widgetsJSON, 320, 240, 10000)

	dashboards, err := s.ListDashboards()
	if err != nil {
		t.Fatalf("ListDashboards failed: %v", err)
	}
	if len(dashboards) != 2 {
		t.Errorf("ListDashboards count = %d, want 2", len(dashboards))
	}
	if dashboards[0].WidgetCount != 0 {
		t.Errorf("WidgetCount = %d, want 0", dashboards[0].WidgetCount)
	}

	widgetsWithCount := `[{"id":"w1","type":"gauge"}]`
	s.UpsertDashboard("dash1", "Dash 1", widgetsWithCount, 128, 64, 15000)

	dashboards, err = s.ListDashboards()
	if err != nil {
		t.Fatalf("ListDashboards failed: %v", err)
	}
	if dashboards[0].WidgetCount != 1 {
		t.Errorf("WidgetCount = %d, want 1", dashboards[0].WidgetCount)
	}
}

func TestStore_DeleteDevice(t *testing.T) {
	s := setupTestDB(t)

	s.UpsertDevice("dev1", "key1", nil)

	err := s.DeleteDevice("dev1")
	if err != nil {
		t.Fatalf("DeleteDevice failed: %v", err)
	}

	_, err = s.GetDevice("dev1")
	if err == nil {
		t.Error("GetDevice should fail after delete")
	}
}

func TestStore_AssignDeviceDashboard(t *testing.T) {
	s := setupTestDB(t)

	s.UpsertDevice("dev1", "key1", nil)
	s.UpsertDashboard("dash1", "Dash 1", `[]`, 128, 64, 15000)

	err := s.AssignDeviceDashboard("dev1", "dash1")
	if err != nil {
		t.Fatalf("AssignDeviceDashboard failed: %v", err)
	}

	d, err := s.GetDevice("dev1")
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	if d.DashboardID == nil || *d.DashboardID != "dash1" {
		t.Errorf("DashboardID not assigned: %v", d.DashboardID)
	}
}

func TestStore_GetAllMonitors(t *testing.T) {
	s := setupTestDB(t)

	s.UpsertMonitor("mon1", "http://example.com/1", "GET", 5000)
	s.UpsertMonitor("mon2", "http://example.com/2", "POST", 3000)

	monitors, err := s.GetAllMonitors()
	if err != nil {
		t.Fatalf("GetAllMonitors failed: %v", err)
	}
	if len(monitors) != 2 {
		t.Errorf("GetAllMonitors count = %d, want 2", len(monitors))
	}
	// Just verify both monitors are returned
	ids := map[string]bool{}
	for _, m := range monitors {
		ids[m.ID] = true
	}
	if !ids["mon1"] || !ids["mon2"] {
		t.Errorf("Missing monitor IDs, got: %v", ids)
	}
}