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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/time/rate"

	"microdashboard/internal/config"
	"microdashboard/internal/store"
	"microdashboard/internal/monitor"
	"microdashboard/internal/auth"
	"microdashboard/internal/dashboard"
	"microdashboard/internal/ui"
	"microdashboard/internal/websocket"
	"microdashboard/internal/metrics"
	"microdashboard/internal/logger"
	"microdashboard/internal/ratelimit"
)

func main() {
	cfg := config.Load()

	log.SetFlags(0)
	appLogger := logger.NewLogger(logger.InfoLevel)
	logger.SetDefault(appLogger)

	st := store.Open(cfg.DBPath)
	defer st.Close()

	if err := st.Migrate(); err != nil {
		logger.Error("migration failed", logger.Fields{"error": err})
		os.Exit(1)
	}

	metrics.SetDevicesRegistered(countDevices(st))
	metrics.SetDashboardsTotal(countDashboards(st))
	metrics.SetMonitorsTotal(countMonitors(st))

	ipLimiter := ratelimit.NewIPRateLimiter(rate.Limit(100), 200)
	go ipLimiter.Cleanup(5 * time.Minute)

	authMiddleware := auth.New(st)
	wsHub := websocket.NewHub(st, authMiddleware)
	go wsHub.Run()

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			metrics.SetActiveWebSocketConnections(wsHub.ClientCount())
		}
	}()

	monitorInstance := monitor.Start(st)
	defer monitorInstance.Stop()

	router := httprouter.New()

	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		w.WriteHeader(http.StatusOK)
	})

	router.GET("/health", healthHandler)
	router.GET("/metrics/prometheus", prometheusHandler)

	router.GET("/ws", wsHandler(wsHub))

	router.GET("/dev/:device_id", authMiddleware.HTTP(deviceGet(st)))
	router.POST("/devices", authMiddleware.Admin(deviceCreate(st)))

	router.GET("/dashboards", authMiddleware.Admin(dashboardList(st)))
	router.POST("/dashboards", authMiddleware.Admin(dashboardCreate(st)))
	router.GET("/dashboards/:id", authMiddleware.Admin(dashboardGet(st)))
	router.PUT("/dashboards/:id", authMiddleware.Admin(dashboardUpdate(st)))
	router.DELETE("/dashboards/:id", authMiddleware.Admin(dashboardDelete(st)))

	router.GET("/dashboards/:id/render", authMiddleware.HTTP(dashboardRender(st)))

	router.GET("/monitors", authMiddleware.Admin(monitorList(st)))
	router.POST("/monitors", authMiddleware.Admin(monitorCreate(st)))

	router.GET("/metrics", authMiddleware.HTTP(metricsGet(st)))

	ui.SetupRoutes(router, st, authMiddleware)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ListenIP, cfg.HTTPPort),
		Handler: router,
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", cfg.ListenIP, cfg.HTTPPort))
	if err != nil {
		logger.Error("failed to listen", logger.Fields{"error": err})
		os.Exit(1)
	}
	logger.Info("server listening", logger.Fields{"address": listener.Addr().String()})

	logger.Info("server started", logger.Fields{"address": listener.Addr().String()})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", logger.Fields{"error": err})
			os.Exit(1)
		}
	}()

	logger.Info("server started", logger.Fields{"address": listener.Addr().String()})

	<-stop
	logger.Info("shutting down", logger.Fields{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server shutdown failed", logger.Fields{"error": err})
	}
	logger.Info("exited", logger.Fields{})
}

func wsHandler(hub *websocket.Hub) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		hub.ServeWS(w, r)
	}
}

func prometheusHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	promhttp.Handler().ServeHTTP(w, r)
}

func healthHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func countDevices(st *store.Store) int {
	devices, _ := st.ListDevices()
	return len(devices)
}

func countDashboards(st *store.Store) int {
	dashboards, _ := st.ListDashboards()
	return len(dashboards)
}

func countMonitors(st *store.Store) int {
	monitors, _ := st.GetAllMonitors()
	return len(monitors)
}

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
		dashboards, err := st.ListDashboards()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dashboards)
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
		req.ID = id
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

func monitorList(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		monitors, err := st.ListMonitors()
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
			ID      string `json:"id"`
			URL     string `json:"url"`
			Method  string `json:"method"`
			Timeout int    `json:"timeout"`
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
		metricsData, err := st.GetMetrics(limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metricsData)
	}
}