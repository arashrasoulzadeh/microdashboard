package auth

import (
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/julienschmidt/httprouter"
	"microdashboard/internal/store"
)

func setupTestAuth(t *testing.T) (*Auth, *store.Store) {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db: %v", err)
	}
	tmpfile.Close()
	
	s := store.Open(tmpfile.Name())
	if err := s.Migrate(); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	
	// Create test device
	s.UpsertDevice("test-device", "test-api-key", nil)
	
	a := New(s)
	
	t.Cleanup(func() {
		s.Close()
		os.Remove(tmpfile.Name())
	})
	
	return a, s
}

func TestAuth_HTTP_ValidKey(t *testing.T) {
	a, _ := setupTestAuth(t)
	
	handler := a.HTTP(httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	w := httptest.NewRecorder()
	
	handler(w, req, nil)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("Body = %q, want %q", w.Body.String(), "OK")
	}
}

func TestAuth_HTTP_MissingKey(t *testing.T) {
	a, _ := setupTestAuth(t)
	
	handler := a.HTTP(httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	handler(w, req, nil)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestAuth_HTTP_InvalidKey(t *testing.T) {
	a, _ := setupTestAuth(t)
	
	handler := a.HTTP(httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	w := httptest.NewRecorder()
	
	handler(w, req, nil)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestAuth_HTTP_ContextValue(t *testing.T) {
	a, _ := setupTestAuth(t)
	
	var capturedDevice *store.DeviceRow
	handler := a.HTTP(httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		d, ok := DeviceFromContext(r)
		if ok {
			capturedDevice = d
		}
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	w := httptest.NewRecorder()
	
	handler(w, req, nil)
	
	if capturedDevice == nil {
		t.Fatal("Device not found in context")
	}
	if capturedDevice.DeviceID != "test-device" {
		t.Errorf("DeviceID = %q, want %q", capturedDevice.DeviceID, "test-device")
	}
}

func TestAuth_Admin_ValidToken(t *testing.T) {
	a, _ := setupTestAuth(t)
	
	handler := a.Admin(httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("admin ok"))
	}))
	
	req := httptest.NewRequest("GET", "/test?token=admin-change-me", nil)
	w := httptest.NewRecorder()
	
	handler(w, req, nil)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	if w.Body.String() != "admin ok" {
		t.Errorf("Body = %q, want %q", w.Body.String(), "admin ok")
	}
}

func TestAuth_Admin_InvalidToken(t *testing.T) {
	a, _ := setupTestAuth(t)
	
	handler := a.Admin(httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("GET", "/test?token=wrong-token", nil)
	w := httptest.NewRecorder()
	
	handler(w, req, nil)
	
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", w.Code)
	}
}

func TestAuth_Admin_MissingToken(t *testing.T) {
	a, _ := setupTestAuth(t)
	
	handler := a.Admin(httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	handler(w, req, nil)
	
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", w.Code)
	}
}

func TestDeviceFromContext_NotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	d, ok := DeviceFromContext(req)
	if ok || d != nil {
		t.Errorf("Expected no device in empty context")
	}
}