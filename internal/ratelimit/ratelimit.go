package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
	}
	return limiter
}

func (i *IPRateLimiter) Cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		i.mu.Lock()
		for ip, limiter := range i.ips {
			if limiter.TokensAt(time.Now()) == float64(i.b) {
				delete(i.ips, ip)
			}
		}
		i.mu.Unlock()
	}
}

func RateLimitMiddleware(limiter *IPRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			ipLimiter := limiter.GetLimiter(ip)

			if !ipLimiter.Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for reverse proxy)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := len(ip) - 1; idx >= 0 {
		for i := idx; i >= 0; i-- {
			if ip[i] == ':' {
				return ip[:i]
			}
		}
	}
	return ip
}

// DeviceRateLimiter limits based on device API key
type DeviceRateLimiter struct {
	devices map[string]*rate.Limiter
	mu      *sync.RWMutex
	r       rate.Limit
	b       int
}

func NewDeviceRateLimiter(r rate.Limit, b int) *DeviceRateLimiter {
	return &DeviceRateLimiter{
		devices: make(map[string]*rate.Limiter),
		mu:      &sync.RWMutex{},
		r:       r,
		b:       b,
	}
}

func (d *DeviceRateLimiter) GetLimiter(deviceID string) *rate.Limiter {
	d.mu.Lock()
	defer d.mu.Unlock()

	limiter, exists := d.devices[deviceID]
	if !exists {
		limiter = rate.NewLimiter(d.r, d.b)
		d.devices[deviceID] = limiter
	}
	return limiter
}

func DeviceRateLimitMiddleware(limiter *DeviceRateLimiter, getDeviceID func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			deviceID := getDeviceID(r)
			if deviceID == "" {
				next.ServeHTTP(w, r)
				return
			}

			limiter := limiter.GetLimiter(deviceID)

			if !limiter.Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}