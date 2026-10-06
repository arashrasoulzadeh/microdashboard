package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "microdashboard_http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "microdashboard_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	DashboardRenders = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "microdashboard_dashboard_renders_total",
		Help: "Total number of dashboard renders",
	}, []string{"dashboard_id", "status"})

	MonitorChecks = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "microdashboard_monitor_checks_total",
		Help: "Total number of monitor checks",
	}, []string{"monitor_id", "status"})

	MonitorLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "microdashboard_monitor_latency_seconds",
		Help:    "Monitor check latency in seconds",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
	}, []string{"monitor_id"})

	DevicesRegistered = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "microdashboard_devices_registered",
		Help: "Number of registered devices",
	})

	DashboardsTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "microdashboard_dashboards_total",
		Help: "Total number of dashboards",
	})

	MonitorsTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "microdashboard_monitors_total",
		Help: "Total number of monitors",
	})

	ActiveWebSocketConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "microdashboard_websocket_connections_active",
		Help: "Number of active WebSocket connections",
	})

	AlertsTriggered = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "microdashboard_alerts_triggered_total",
		Help: "Total number of alerts triggered",
	}, []string{"alert_type", "severity"})

	DBQueriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "microdashboard_db_queries_total",
		Help: "Total number of database queries",
	}, []string{"query_type", "status"})

	DBQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "microdashboard_db_query_duration_seconds",
		Help:    "Database query latency in seconds",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
	}, []string{"query_type"})
)

func RecordHTTPRequest(method, path string, status int) {
	HTTPRequestsTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
}

func RecordHTTPDuration(method, path string, duration float64) {
	HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
}

func RecordDashboardRender(dashboardID string, success bool) {
	status := "success"
	if !success {
		status = "error"
	}
	DashboardRenders.WithLabelValues(dashboardID, status).Inc()
}

func RecordMonitorCheck(monitorID string, success bool, latencyMs float64) {
	status := "success"
	if !success {
		status = "error"
	}
	MonitorChecks.WithLabelValues(monitorID, status).Inc()
	MonitorLatency.WithLabelValues(monitorID).Observe(latencyMs / 1000.0)
}

func SetDevicesRegistered(count int) {
	DevicesRegistered.Set(float64(count))
}

func SetDashboardsTotal(count int) {
	DashboardsTotal.Set(float64(count))
}

func SetMonitorsTotal(count int) {
	MonitorsTotal.Set(float64(count))
}

func SetActiveWebSocketConnections(count int) {
	ActiveWebSocketConnections.Set(float64(count))
}

func RecordAlert(alertType, severity string) {
	AlertsTriggered.WithLabelValues(alertType, severity).Inc()
}

func RecordDBQuery(queryType string, success bool, duration float64) {
	status := "success"
	if !success {
		status = "error"
	}
	DBQueriesTotal.WithLabelValues(queryType, status).Inc()
	DBQueryDuration.WithLabelValues(queryType).Observe(duration)
}