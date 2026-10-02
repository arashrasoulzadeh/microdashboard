package dashboard_test

import (
	_ "github.com/mattn/go-sqlite3"
	"os"
	"testing"

	"microdashboard/internal/dashboard"
	"microdashboard/internal/store"
)

func setupTestStore(t *testing.T) *store.Store {
	return setupTestStoreHelper(t)
}

func setupTestStoreHelper(t *testing.T) *store.Store {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db: %v", err)
	}
	tmpfile.Close()

	s := store.Open(tmpfile.Name())
	if err := s.Migrate(); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	s.UpsertMonitor("mon1", "http://example.com", "GET", 5000)
	s.UpdateMonitorResult("mon1", 150, 200)

	s.UpsertDevice("dev1", "key1", nil)
	s.InsertMetric("dev1", "mon1", 200, 200, 512)

	t.Cleanup(func() {
		s.Close()
		os.Remove(tmpfile.Name())
	})

	return s
}

func TestValidateDashboard(t *testing.T) {
	tests := []struct {
		name    string
		d       *dashboard.Dashboard
		wantErr bool
	}{
		{
			name: "valid dashboard",
			d: &dashboard.Dashboard{
				ID:              "dash1",
				Name:            "Test",
				Width:           128,
				Height:          64,
				RefreshInterval: 15000,
				Widgets: []dashboard.Widget{
					{ID: "w1", Type: "gauge", X: 0, Y: 0, Width: 32, Height: 16, Expression: "${latency:mon1}", Unit: "ms"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing id",
			d: &dashboard.Dashboard{
				Name:  "Test",
				Widgets: []dashboard.Widget{{ID: "w1", Type: "gauge", Expression: "${latency:mon1}"}},
			},
			wantErr: true,
		},
		{
			name: "missing name",
			d: &dashboard.Dashboard{
				ID:      "dash1",
				Widgets: []dashboard.Widget{{ID: "w1", Type: "gauge", Expression: "${latency:mon1}"}},
			},
			wantErr: true,
		},
		{
			name: "no widgets",
			d: &dashboard.Dashboard{
				ID:   "dash1",
				Name: "Test",
			},
			wantErr: true,
		},
		{
			name: "invalid widget type",
			d: &dashboard.Dashboard{
				ID:   "dash1",
				Name: "Test",
				Widgets: []dashboard.Widget{{ID: "w1", Type: "invalid", Expression: "${latency:mon1}"}},
			},
			wantErr: true,
		},
		{
			name: "widget missing expression",
			d: &dashboard.Dashboard{
				ID:   "dash1",
				Name: "Test",
				Widgets: []dashboard.Widget{{ID: "w1", Type: "gauge"}},
			},
			wantErr: true,
		},
		{
			name: "widget expression without placeholder",
			d: &dashboard.Dashboard{
				ID:   "dash1",
				Name: "Test",
				Widgets: []dashboard.Widget{{ID: "w1", Type: "gauge", Expression: "static text"}},
			},
			wantErr: true,
		},
		{
			name: "defaults applied",
			d: &dashboard.Dashboard{
				ID:   "dash1",
				Name: "Test",
				Widgets: []dashboard.Widget{{ID: "w1", Type: "gauge", Expression: "${latency:mon1}"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dashboard.ValidateDashboard(tt.d)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDashboard() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if tt.d.Width == 0 {
					t.Error("Width default not applied")
				}
				if tt.d.Height == 0 {
					t.Error("Height default not applied")
				}
				if tt.d.RefreshInterval == 0 {
					t.Error("RefreshInterval default not applied")
				}
			}
		})
	}
}

func TestRenderDashboard(t *testing.T) {
	s := setupTestStoreHelper(t)

	d := &dashboard.Dashboard{
		ID:              "dash1",
		Name:            "Test",
		Width:           128,
		Height:          64,
		RefreshInterval: 15000,
		Widgets: []dashboard.Widget{
			{ID: "w1", Type: "gauge", X: 0, Y: 0, Width: 32, Height: 16, Expression: "${latency:mon1}", Unit: "ms"},
			{ID: "w2", Type: "numeric", X: 40, Y: 0, Width: 30, Height: 10, Expression: "${latency:mon1}", Unit: "ms"},
			{ID: "w3", Type: "text", X: 0, Y: 20, Width: 128, Height: 10, Expression: "Latency: ${latency:mon1} OK"},
			{ID: "w4", Type: "status", X: 0, Y: 35, Width: 20, Height: 10, Expression: "${latency:mon1}"},
		},
	}

	rendered, err := dashboard.Render(d, s)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if rendered.ID != "dash1" {
		t.Errorf("Rendered ID = %q, want %q", rendered.ID, "dash1")
	}
	if len(rendered.Widgets) != 4 {
		t.Errorf("Widget count = %d, want 4", len(rendered.Widgets))
	}

	w1 := rendered.Widgets[0]
	if w1.ID != "w1" || w1.Type != "gauge" {
		t.Errorf("w1: ID=%q Type=%q", w1.ID, w1.Type)
	}
	if w1.Value != 150.0 {
		t.Errorf("w1 value = %v, want 150", w1.Value)
	}
	if w1.Text != "150ms" {
		t.Errorf("w1 text = %q, want %q", w1.Text, "150ms")
	}
	if w1.Status != "ok" {
		t.Errorf("w1 status = %q, want ok", w1.Status)
	}

	w3 := rendered.Widgets[2]
	if w3.Text != "Latency: 150ms OK" {
		t.Errorf("w3 text = %q, want %q", w3.Text, "Latency: 150ms OK")
	}
}

func TestRenderDashboard_MonitorNotFound(t *testing.T) {
	s := setupTestStoreHelper(t)

	d := &dashboard.Dashboard{
		ID:   "dash1",
		Name: "Test",
		Widgets: []dashboard.Widget{
			{ID: "w1", Type: "gauge", Expression: "${latency:nonexistent}"},
		},
	}

	rendered, err := dashboard.Render(d, s)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if len(rendered.Widgets) != 1 {
		t.Errorf("Widget count = %d", len(rendered.Widgets))
	}
	w := rendered.Widgets[0]
	if w.Status != "crit" {
		t.Errorf("Status = %q, want crit", w.Status)
	}
	if len(w.Text) < 4 || w.Text[:4] != "ERR:" {
		t.Errorf("Error text = %q", w.Text)
	}
}

func TestRender_StaticText(t *testing.T) {
	s := setupTestStoreHelper(t)

	d := &dashboard.Dashboard{
		ID:   "dash1",
		Name: "Test",
		Widgets: []dashboard.Widget{
			{ID: "w1", Type: "text", Expression: "Static label"},
		},
	}

	rendered, err := dashboard.Render(d, s)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	w := rendered.Widgets[0]
	if w.Text != "Static label" {
		t.Errorf("Text = %q, want %q", w.Text, "Static label")
	}
	if w.Value != nil {
		t.Errorf("Value should be nil for static text")
	}
}

func TestRender_MultiplePlaceholders(t *testing.T) {
	s := setupTestStoreHelper(t)

	d := &dashboard.Dashboard{
		ID:   "dash1",
		Name: "Test",
		Widgets: []dashboard.Widget{
			{ID: "w1", Type: "text", Expression: "A: ${latency:mon1} B: ${latency:mon1}"},
		},
	}

	rendered, err := dashboard.Render(d, s)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	w := rendered.Widgets[0]
	if w.Text != "A: 150ms B: 150ms" {
		t.Errorf("Text = %q", w.Text)
	}
}