package ui

import (
	"embed"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"microdashboard/internal/auth"
	"microdashboard/internal/dashboard"
	"microdashboard/internal/store"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

func SetupRoutes(router *httprouter.Router, st *store.Store, authMiddleware *auth.Auth) {
	// Static files - use a different pattern to avoid conflict
	router.GET("/ui/static/*filepath", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		filepath := p.ByName("filepath")
		if filepath == "" {
			http.NotFound(w, r)
			return
		}
		data, err := staticFiles.ReadFile("static" + filepath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		setContentType(w, filepath)
		w.Write(data)
	})

	// Main UI page (requires admin token)
	router.GET("/ui/", authMiddleware.Admin(uiIndexHandler))

	// Alternative direct index
	router.GET("/ui", authMiddleware.Admin(uiIndexHandler))

	// API endpoints for UI (admin token)
	router.GET("/api/dashboards", authMiddleware.Admin(apiDashboardList(st)))
	router.POST("/api/dashboards", authMiddleware.Admin(apiDashboardCreate(st)))
	router.GET("/api/dashboards/:id", authMiddleware.Admin(apiDashboardGet(st)))
	router.PUT("/api/dashboards/:id", authMiddleware.Admin(apiDashboardUpdate(st)))
	router.DELETE("/api/dashboards/:id", authMiddleware.Admin(apiDashboardDelete(st)))

	router.GET("/api/devices", authMiddleware.Admin(apiDeviceList(st)))
	router.POST("/api/devices", authMiddleware.Admin(apiDeviceCreate(st)))
	router.DELETE("/api/devices/:id", authMiddleware.Admin(apiDeviceDelete(st)))
	router.PUT("/api/devices/:id/dashboard", authMiddleware.Admin(apiDeviceAssignDashboard(st)))

	router.GET("/api/monitors", authMiddleware.Admin(apiMonitorList(st)))
	router.POST("/api/monitors", authMiddleware.Admin(apiMonitorCreate(st)))
	router.DELETE("/api/monitors/:id", authMiddleware.Admin(apiMonitorDelete(st)))

	router.GET("/api/alerts", authMiddleware.Admin(apiAlertConfigGet(st)))
	router.POST("/api/alerts", authMiddleware.Admin(apiAlertConfigUpdate(st)))

	// Render endpoint (device key or admin)
	router.GET("/api/dashboards/:id/render", authMiddleware.HTTP(apiDashboardRender(st)))
}

func uiIndexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	data, err := templateFiles.ReadFile("templates/index.html")
	if err != nil {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}

func setContentType(w http.ResponseWriter, path string) {
	if strings.HasSuffix(path, ".js") {
		w.Header().Set("Content-Type", "application/javascript")
	} else if strings.HasSuffix(path, ".css") {
		w.Header().Set("Content-Type", "text/css")
	} else if strings.HasSuffix(path, ".html") {
		w.Header().Set("Content-Type", "text/html")
	} else if strings.HasSuffix(path, ".json") {
		w.Header().Set("Content-Type", "application/json")
	}
}

// --- API Handlers ---

func apiDashboardList(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		dashboards, err := st.ListDashboards()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(dashboards)
	}
}

func apiDashboardCreate(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var req struct {
			ID             string          `json:"id"`
			Name           string          `json:"name"`
			Width          int             `json:"width"`
			Height         int             `json:"height"`
			RefreshInterval int            `json:"refresh_interval"`
			Widgets        json.RawMessage `json:"widgets"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		if req.ID == "" || req.Name == "" {
			http.Error(w, "id and name required", 400)
			return
		}
		if req.Width == 0 { req.Width = 128 }
		if req.Height == 0 { req.Height = 64 }
		if req.RefreshInterval == 0 { req.RefreshInterval = 15000 }
		if req.Widgets == nil { req.Widgets = json.RawMessage("[]") }

		if err := st.UpsertDashboard(req.ID, req.Name, string(req.Widgets), req.Width, req.Height, req.RefreshInterval); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	}
}

func apiDashboardGet(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		id := p.ByName("id")
		d, err := st.GetDashboard(id)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		json.NewEncoder(w).Encode(d)
	}
}

func apiDashboardUpdate(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		id := p.ByName("id")
		var req struct {
			Name           string          `json:"name"`
			Width          int             `json:"width"`
			Height         int             `json:"height"`
			RefreshInterval int            `json:"refresh_interval"`
			Widgets        json.RawMessage `json:"widgets"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		if req.Width == 0 { req.Width = 128 }
		if req.Height == 0 { req.Height = 64 }
		if req.RefreshInterval == 0 { req.RefreshInterval = 15000 }
		if req.Widgets == nil { req.Widgets = json.RawMessage("[]") }

		if err := st.UpsertDashboard(id, req.Name, string(req.Widgets), req.Width, req.Height, req.RefreshInterval); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}
}

func apiDashboardDelete(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		id := p.ByName("id")
		if err := st.DeleteDashboard(id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(204)
	}
}

func apiDashboardRender(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		id := p.ByName("id")
		d, err := st.GetDashboard(id)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		var dash dashboard.Dashboard
		dash.ID = d.ID
		dash.Name = d.Name
		dash.Width = d.Width
		dash.Height = d.Height
		dash.RefreshInterval = d.RefreshInterval
		dash.CreatedAt = d.CreatedAt
		json.Unmarshal([]byte(d.JSONDef), &dash.Widgets)

		rendered, err := dashboard.Render(&dash, st)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rendered)
	}
}

// Device handlers
func apiDeviceList(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		devices, err := st.ListDevices()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(devices)
	}
}

func apiDeviceCreate(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var req struct {
			DeviceID    string `json:"device_id"`
			APIKey      string `json:"api_key"`
			DashboardID string `json:"dashboard_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		if req.DeviceID == "" || req.APIKey == "" {
			http.Error(w, "device_id and api_key required", 400)
			return
		}
		var dash *string
		if req.DashboardID != "" { dash = &req.DashboardID }
		if err := st.UpsertDevice(req.DeviceID, req.APIKey, dash); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	}
}

func apiDeviceDelete(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		id := p.ByName("id")
		if err := st.DeleteDevice(id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(204)
	}
}

func apiDeviceAssignDashboard(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		id := p.ByName("id")
		var req struct {
			DashboardID string `json:"dashboard_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		if err := st.AssignDeviceDashboard(id, req.DashboardID); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}
}

// Monitor handlers
func apiMonitorList(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		monitors, err := st.ListMonitors()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(monitors)
	}
}

func apiMonitorCreate(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var req struct {
			ID      string `json:"id"`
			URL     string `json:"url"`
			Method  string `json:"method"`
			Timeout int    `json:"timeout"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		if req.ID == "" || req.URL == "" {
			http.Error(w, "id and url required", 400)
			return
		}
		if req.Method == "" { req.Method = "GET" }
		if req.Timeout == 0 { req.Timeout = 5000 }
		if err := st.UpsertMonitor(req.ID, req.URL, req.Method, req.Timeout); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	}
}

func apiMonitorDelete(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		id := p.ByName("id")
		if err := st.DeleteMonitor(id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(204)
	}
}

// Alert config handlers
func apiAlertConfigGet(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		json.NewEncoder(w).Encode(map[string]string{
			"webhook_url": "",
			"mqtt_topic":  "md/alerts",
		})
	}
}

func apiAlertConfigUpdate(st *store.Store) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}
}

func init() {
	log.Println("UI package initialized")
}