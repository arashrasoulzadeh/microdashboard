package metrics_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"microdashboard/internal/metrics"
)

func TestMetrics_RecordHTTPRequest(t *testing.T) {
	metrics.RecordHTTPRequest("GET", "/test", 200)
	metrics.RecordHTTPRequest("POST", "/api", 404)

	val := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("GET", "/test", "200"))
	if val != 1 {
		t.Errorf("GET /test 200 = %v, want 1", val)
	}

	val = testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("POST", "/api", "404"))
	if val != 1 {
		t.Errorf("POST /api 404 = %v, want 1", val)
	}
}

func TestMetrics_RecordDashboardRender(t *testing.T) {
	metrics.RecordDashboardRender("dash1", true)
	metrics.RecordDashboardRender("dash1", false)

	val := testutil.ToFloat64(metrics.DashboardRenders.WithLabelValues("dash1", "success"))
	if val != 1 {
		t.Errorf("success renders = %v, want 1", val)
	}

	val = testutil.ToFloat64(metrics.DashboardRenders.WithLabelValues("dash1", "error"))
	if val != 1 {
		t.Errorf("error renders = %v, want 1", val)
	}
}

func TestMetrics_RecordMonitorCheck(t *testing.T) {
	metrics.RecordMonitorCheck("mon1", true, 100)
	metrics.RecordMonitorCheck("mon1", false, 200)

	val := testutil.ToFloat64(metrics.MonitorChecks.WithLabelValues("mon1", "success"))
	if val != 1 {
		t.Errorf("success checks = %v, want 1", val)
	}

	val = testutil.ToFloat64(metrics.MonitorChecks.WithLabelValues("mon1", "error"))
	if val != 1 {
		t.Errorf("error checks = %v, want 1", val)
	}
}

func TestMetrics_SetDevicesRegistered(t *testing.T) {
	metrics.SetDevicesRegistered(5)
	val := testutil.ToFloat64(metrics.DevicesRegistered)
	if val != 5 {
		t.Errorf("devices = %v, want 5", val)
	}

	metrics.SetDevicesRegistered(0)
	val = testutil.ToFloat64(metrics.DevicesRegistered)
	if val != 0 {
		t.Errorf("devices = %v, want 0", val)
	}
}

func TestMetrics_SetDashboardsTotal(t *testing.T) {
	metrics.SetDashboardsTotal(10)
	val := testutil.ToFloat64(metrics.DashboardsTotal)
	if val != 10 {
		t.Errorf("dashboards = %v, want 10", val)
	}
}

func TestMetrics_SetMonitorsTotal(t *testing.T) {
	metrics.SetMonitorsTotal(3)
	val := testutil.ToFloat64(metrics.MonitorsTotal)
	if val != 3 {
		t.Errorf("monitors = %v, want 3", val)
	}
}

func TestMetrics_SetActiveWebSocketConnections(t *testing.T) {
	metrics.SetActiveWebSocketConnections(7)
	val := testutil.ToFloat64(metrics.ActiveWebSocketConnections)
	if val != 7 {
		t.Errorf("connections = %v, want 7", val)
	}
}

func TestMetrics_RecordAlert(t *testing.T) {
	metrics.RecordAlert("threshold", "warn")
	metrics.RecordAlert("threshold", "crit")

	val := testutil.ToFloat64(metrics.AlertsTriggered.WithLabelValues("threshold", "warn"))
	if val != 1 {
		t.Errorf("warn alerts = %v, want 1", val)
	}

	val = testutil.ToFloat64(metrics.AlertsTriggered.WithLabelValues("threshold", "crit"))
	if val != 1 {
		t.Errorf("crit alerts = %v, want 1", val)
	}
}

func TestMetrics_RecordDBQuery(t *testing.T) {
	metrics.RecordDBQuery("select", true, 0.001)
	metrics.RecordDBQuery("insert", false, 0.002)

	val := testutil.ToFloat64(metrics.DBQueriesTotal.WithLabelValues("select", "success"))
	if val != 1 {
		t.Errorf("select success = %v, want 1", val)
	}

	val = testutil.ToFloat64(metrics.DBQueriesTotal.WithLabelValues("insert", "error"))
	if val != 1 {
		t.Errorf("insert error = %v, want 1", val)
	}
}

func TestMetrics_HistogramMetrics(t *testing.T) {
	// Just call the functions to ensure they don't panic
	metrics.RecordHTTPDuration("GET", "/test", 0.123)
	metrics.RecordMonitorCheck("mon1", true, 100)
	metrics.RecordDBQuery("select", true, 0.001)
}