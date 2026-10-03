package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	_ "github.com/mattn/go-sqlite3"
	"microdashboard/internal/config"
	"microdashboard/internal/store"
	"microdashboard/internal/monitor"
	"microdashboard/internal/auth"
	"microdashboard/internal/dashboard"
	"microdashboard/internal/ui"
)

func main() {
	cfg := config.Load()
	st := store.Open(cfg.DBPath)
	defer st.Close()

	// Run migrations
	if err := st.Migrate(); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// Start latency monitor checker in background
	_ = monitor.Start(st)

	// Setup router
	router := httprouter.New()

	// Auth middleware
	authMiddleware := auth.New(st)

	// Health endpoint (no auth)
	router.GET("/health", healthHandler)

	// Device endpoints (require key)
	router.GET("/dev/:device_id", authMiddleware.HTTP(deviceGet(st)))
	router.POST("/devices", authMiddleware.Admin(deviceCreate(st)))

	// Dashboard endpoints (admin only via token)
	router.GET("/dashboards", authMiddleware.Admin(dashboardList(st)))
	router.POST("/dashboards", authMiddleware.Admin(dashboardCreate(st)))
	router.GET("/dashboards/:id", authMiddleware.Admin(dashboardGet(st)))
	router.PUT("/dashboards/:id", authMiddleware.Admin(dashboardUpdate(st)))
	router.DELETE("/dashboards/:id", authMiddleware.Admin(dashboardDelete(st)))
	// Render endpoint (device key or admin token)
	router.GET("/dashboards/:id/render", authMiddleware.HTTP(dashboardRender(st)))

	// Latency monitor endpoints (admin)
	router.GET("/monitors", authMiddleware.Admin(monitorList(st)))
	router.POST("/monitors", authMiddleware.Admin(monitorCreate(st)))

	// Metrics endpoint (device key)
	router.GET("/metrics", authMiddleware.HTTP(metricsGet(st)))

	// UI Routes
	ui.SetupRoutes(router, st, authMiddleware)

	// Server setup
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ListenIP, cfg.HTTPPort),
		Handler: router,
	}

	// Listen
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("listening on %s", listener.Addr().String())

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	log.Println("awaiting shutdown")
	<-stop
	log.Println("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server shutdown failed:", err)
	}
	log.Println("exiting")
}

// Helper to wrap handlers with auth that returns httprouter.Handle
func h(h func(*store.Store, http.ResponseWriter, *http.Request, httprouter.Params)) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		h(nil, w, r, p)
	}
}

// Health endpoint
func healthHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Handler factories that return httprouter.Handle

func deviceGet(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		deviceID := p.ByName("device_id")
		device, err := st.GetDevice(deviceID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(device)
	}
}

func deviceCreate(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var req struct {
			DeviceID    string `json:"device_id"`
			APIKey      string `json:"api_key"`
			DashboardID string `json:"dashboard_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if req.DeviceID == "" || req.APIKey == "" {
			http.Error(w, "device_id and api_key required", http.StatusBadRequest)
			return
		}
		var dash *string
		if req.DashboardID != "" {
			dash = &req.DashboardID
		}
		if err := st.UpsertDevice(req.DeviceID, req.APIKey, dash); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	}
}

func dashboardList(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
	}
}

func dashboardCreate(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var req dashboard.Dashboard
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if err := dashboard.ValidateDashboard(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Marshal widgets to JSON for storage
		widgetsJSON, _ := json.Marshal(req.Widgets)
		if err := st.UpsertDashboard(req.ID, req.Name, string(widgetsJSON), req.Width, req.Height, req.RefreshInterval); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	}
}

func dashboardGet(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		id := p.ByName("id")
		dash, err := st.GetDashboard(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		// Parse widgets from JSON
		var dashResp dashboard.Dashboard
		dashResp.ID = dash.ID
		dashResp.Name = dash.Name
		dashResp.Width = dash.Width
		dashResp.Height = dash.Height
		dashResp.RefreshInterval = dash.RefreshInterval
		dashResp.CreatedAt = dash.CreatedAt
		json.Unmarshal([]byte(dash.JSONDef), &dashResp.Widgets)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dashResp)
	}
}

func dashboardUpdate(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		id := p.ByName("id")
		var req dashboard.Dashboard
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		req.ID = id // ensure ID matches
		if err := dashboard.ValidateDashboard(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		widgetsJSON, _ := json.Marshal(req.Widgets)
		if err := st.UpsertDashboard(id, req.Name, string(widgetsJSON), req.Width, req.Height, req.RefreshInterval); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}
}

func dashboardDelete(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		id := p.ByName("id")
		if err := st.DeleteDashboard(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func monitorList(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		monitors, err := st.GetAllMonitors()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(monitors)
	}
}

func monitorCreate(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var req struct {
			ID       string `json:"id"`
			URL      string `json:"url"`
			Method   string `json:"method"`
			Timeout  int    `json:"timeout"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if req.ID == "" || req.URL == "" {
			http.Error(w, "id and url required", http.StatusBadRequest)
			return
		}
		if req.Method == "" {
			req.Method = "GET"
		}
		if req.Timeout == 0 {
			req.Timeout = 5000
		}
		if err := st.UpsertMonitor(req.ID, req.URL, req.Method, req.Timeout); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	}
}

func metricsGet(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}
		metrics, err := st.GetMetrics(limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	}
}

func dashboardRender(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		id := p.ByName("id")
		dash, err := st.GetDashboard(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		var d dashboard.Dashboard
		d.ID = dash.ID
		d.Name = dash.Name
		d.Width = dash.Width
		d.Height = dash.Height
		d.RefreshInterval = dash.RefreshInterval
		d.CreatedAt = dash.CreatedAt
		json.Unmarshal([]byte(dash.JSONDef), &d.Widgets)

		rendered, err := dashboard.Render(&d, st)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rendered)
	}
}