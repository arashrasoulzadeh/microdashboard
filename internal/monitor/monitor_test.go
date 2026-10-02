package monitor

import (
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"microdashboard/internal/store"
)

func setupTestStore(t *testing.T) *store.Store {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db: %v", err)
	}
	tmpfile.Close()
	
	s := store.Open(tmpfile.Name())
	if err := s.Migrate(); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	
	s.UpsertMonitor("mon1", "http://localhost:18080/test", "GET", 5000)
	
	t.Cleanup(func() {
		s.Close()
		os.Remove(tmpfile.Name())
	})
	
	return s
}

func TestMonitor_CheckOne_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Ensure some elapsed time
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()
	
	s := setupTestStore(t)
	// Update monitor URL to test server
	s.UpsertMonitor("mon1", server.URL, "GET", 5000)
	
	m := &Monitor{
		store:    s,
		interval: 30 * time.Second,
	}
	
	result := m.checkOne("mon1", server.URL, "GET", 5000)
	
	if result.Error != nil {
		t.Errorf("checkOne error: %v", result.Error)
	}
	if result.Status != 200 {
		t.Errorf("Status = %d, want 200", result.Status)
	}
	if result.ElapsedMs < 5 { // At least 5ms (including 10ms sleep)
		t.Errorf("ElapsedMs = %d, want >= 5", result.ElapsedMs)
	}
	if result.BodyBytes != 2 {
		t.Errorf("BodyBytes = %d, want 2", result.BodyBytes)
	}
}

func TestMonitor_CheckOne_Error(t *testing.T) {
	s := setupTestStore(t)
	
	// Non-existent server
	m := &Monitor{
		store:    s,
		interval: 30 * time.Second,
	}
	
	result := m.checkOne("mon1", "http://127.0.0.1:9999", "GET", 1000)
	
	if result.Error == nil {
		t.Error("Expected error for unreachable host")
	}
	if result.Status != 0 {
		t.Errorf("Status = %d, want 0 for error", result.Status)
	}
}

func TestMonitor_CheckOne_Timeout(t *testing.T) {
	// Slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	s := setupTestStore(t)
	s.UpsertMonitor("mon1", server.URL, "GET", 5000)
	
	m := &Monitor{
		store:    s,
		interval: 30 * time.Second,
	}
	
	// Use short timeout (500ms)
	result := m.checkOne("mon1", server.URL, "GET", 500)
	
	if result.Error == nil {
		t.Error("Expected timeout error")
	}
	if result.Status != 0 {
		t.Errorf("Status = %d, want 0 for timeout", result.Status)
	}
}

func TestMonitor_CheckOne_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	
	s := setupTestStore(t)
	s.UpsertMonitor("mon1", server.URL, "GET", 5000)
	
	m := &Monitor{
		store:    s,
		interval: 30 * time.Second,
	}
	
	result := m.checkOne("mon1", server.URL, "GET", 5000)
	
	if result.Error != nil {
		t.Errorf("Unexpected error: %v", result.Error)
	}
	if result.Status != 500 {
		t.Errorf("Status = %d, want 500", result.Status)
	}
}