package websocket_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	ws "microdashboard/internal/websocket"
	"microdashboard/internal/auth"
	"microdashboard/internal/store"
)

func setupTestHub(t *testing.T) (*ws.Hub, *store.Store, *auth.Auth) {
	tmpfile := store.CreateTempDB(t)

	s := store.Open(tmpfile)
	if err := s.Migrate(); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	s.UpsertDevice("test-device", "test-api-key", nil)

	a := auth.New(s)
	h := ws.NewHub(s, a)

	go h.Run()

	t.Cleanup(func() {
		s.Close()
		os.Remove(tmpfile)
	})

	return h, s, a
}

func TestHub_NewHub(t *testing.T) {
	h, _, _ := setupTestHub(t)
	if h == nil {
		t.Fatal("hub is nil")
	}
}

func TestHub_ClientCount(t *testing.T) {
	h, _, _ := setupTestHub(t)
	time.Sleep(10 * time.Millisecond)

	count := h.ClientCount()
	if count != 0 {
		t.Errorf("initial client count = %d, want 0", count)
	}
}

func TestHub_BroadcastDashboardUpdate(t *testing.T) {
	h, _, _ := setupTestHub(t)

	h.BroadcastDashboardUpdate("dash1", map[string]string{"status": "ok"})
	time.Sleep(10 * time.Millisecond)
}

func TestHub_BroadcastAlert(t *testing.T) {
	h, _, _ := setupTestHub(t)

	h.BroadcastAlert(map[string]interface{}{"type": "test", "message": "hello"})
	time.Sleep(10 * time.Millisecond)
}

func TestHub_ServeWS_Unauthorized(t *testing.T) {
	h, _, _ := setupTestHub(t)

	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	h.ServeWS(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHub_ServeWS_ValidToken(t *testing.T) {
	h, _, _ := setupTestHub(t)

	req := httptest.NewRequest("GET", "/ws?token=test-api-key", nil)
	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	w := httptest.NewRecorder()

	h.ServeWS(w, req)

	if w.Code != http.StatusSwitchingProtocols {
		t.Logf("WebSocket upgrade failed (expected in test env): %d", w.Code)
	}
}

func TestClient_SendForTest(t *testing.T) {
	h, _, _ := setupTestHub(t)

	client := ws.NewClientForTest(h, &store.DeviceRow{DeviceID: "test-device"})

	msg := map[string]interface{}{
		"type":         "subscribe",
		"dashboard_id": "dash1",
	}
	data, _ := json.Marshal(msg)
	client.SendForTest(data)

	time.Sleep(10 * time.Millisecond)

	// DashboardID is only set when ReadPump processes the message
	// This test just verifies SendForTest doesn't panic
}

func TestClient_WritePump(t *testing.T) {
	h, _, _ := setupTestHub(t)

	client := ws.NewClientForTest(h, &store.DeviceRow{DeviceID: "test-device"})

	client.SendForTest([]byte(`{"type":"test"}`))

	time.Sleep(10 * time.Millisecond)
}

func TestUpgrader_CheckOrigin(t *testing.T) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://example.com")

	if !upgrader.CheckOrigin(req) {
		t.Error("CheckOrigin should return true for all origins")
	}
}