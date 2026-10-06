package ratelimit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"microdashboard/internal/ratelimit"
)

func TestIPRateLimiter_GetLimiter(t *testing.T) {
	limiter := ratelimit.NewIPRateLimiter(10, 20)

	l1 := limiter.GetLimiter("192.168.1.1")
	l2 := limiter.GetLimiter("192.168.1.1")
	l3 := limiter.GetLimiter("192.168.1.2")

	if l1 != l2 {
		t.Error("same IP should return same limiter")
	}
	if l1 == l3 {
		t.Error("different IP should return different limiter")
	}
}

func TestIPRateLimiter_Allow(t *testing.T) {
	limiter := ratelimit.NewIPRateLimiter(10, 3)

	l := limiter.GetLimiter("10.0.0.1")
	for i := 0; i < 3; i++ {
		if !l.Allow() {
			t.Errorf("request %d should be allowed", i+1)
		}
	}
	if l.Allow() {
		t.Error("4th request should be denied")
	}
}

func TestIPRateLimiter_Cleanup(t *testing.T) {
	limiter := ratelimit.NewIPRateLimiter(10, 2)

	l := limiter.GetLimiter("10.0.0.1")

	done := make(chan struct{})
	go func() {
		limiter.Cleanup(1 * time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
	}

	if l.TokensAt(time.Now()) != 2 {
		t.Error("limiter should be cleaned up after tokens refill")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	limiter := ratelimit.NewIPRateLimiter(5, 2)
	middleware := ratelimit.RateLimitMiddleware(limiter)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("first request: expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("second request: expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("third request: expected 429, got %d", w.Code)
	}
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
	req.RemoteAddr = "192.168.1.1:12345"

	ip := ratelimit.GetClientIPForTest(req)
	expected := "203.0.113.195, 70.41.3.18, 150.172.238.178"
	if ip != expected {
		t.Errorf("X-Forwarded-For: expected %q, got %q", expected, ip)
	}
}

func TestGetClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-IP", "203.0.113.195")
	req.RemoteAddr = "192.168.1.1:12345"

	ip := ratelimit.GetClientIPForTest(req)
	if ip != "203.0.113.195" {
		t.Errorf("X-Real-IP: expected 203.0.113.195, got %s", ip)
	}
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	ip := ratelimit.GetClientIPForTest(req)
	if ip != "192.168.1.100" {
		t.Errorf("RemoteAddr: expected 192.168.1.100, got %s", ip)
	}
}

func TestDeviceRateLimiter_GetLimiter(t *testing.T) {
	limiter := ratelimit.NewDeviceRateLimiter(10, 20)

	l1 := limiter.GetLimiter("device1")
	l2 := limiter.GetLimiter("device1")
	l3 := limiter.GetLimiter("device2")

	if l1 != l2 {
		t.Error("same device should return same limiter")
	}
	if l1 == l3 {
		t.Error("different device should return different limiter")
	}
}

func TestDeviceRateLimitMiddleware(t *testing.T) {
	limiter := ratelimit.NewDeviceRateLimiter(5, 2)
	middleware := ratelimit.DeviceRateLimitMiddleware(limiter, func(r *http.Request) string {
		return r.Header.Get("X-Device-ID")
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Device-ID", "dev1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("first: expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("second: expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("third: expected 429, got %d", w.Code)
	}
}

func TestDeviceRateLimitMiddleware_NoDeviceID(t *testing.T) {
	limiter := ratelimit.NewDeviceRateLimiter(1, 1)
	middleware := ratelimit.DeviceRateLimitMiddleware(limiter, func(r *http.Request) string {
		return r.Header.Get("X-Device-ID")
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("no device id should pass through: got %d", w.Code)
	}
}