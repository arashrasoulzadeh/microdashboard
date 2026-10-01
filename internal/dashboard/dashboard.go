package dashboard

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"microdashboard/internal/store"
)

type Dashboard struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	RefreshInterval int      `json:"refresh_interval"` // ms
	Widgets        []Widget  `json:"widgets"`
	CreatedAt      int64     `json:"created_at"`
}

type Widget struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`         // gauge, sparkline, status, numeric, progress, text
	X           int         `json:"x"`
	Y           int         `json:"y"`
	Width       int         `json:"width"`
	Height      int         `json:"height"`
	Expression  string      `json:"expression"`   // ${latency:monitor1}, ${metric:device1.temp}
	Unit        string      `json:"unit,omitempty"`
	Options     WidgetOptions `json:"options,omitempty"`
}

type WidgetOptions struct {
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	ThresholdWarn *float64 `json:"threshold_warn,omitempty"`
	ThresholdCrit *float64 `json:"threshold_crit,omitempty"`
	Precision   int       `json:"precision,omitempty"`
	Decimals    int       `json:"decimals,omitempty"`
	Label       string    `json:"label,omitempty"`
}

type RenderedWidget struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Value  interface{} `json:"value"`  // computed value
	Text   string `json:"text"`       // formatted text for display
	Unit   string `json:"unit,omitempty"`
	Status string `json:"status,omitempty"` // ok, warn, crit
}

var validWidgetTypes = map[string]bool{
	"gauge":      true,
	"sparkline":  true,
	"status":     true,
	"numeric":    true,
	"progress":   true,
	"text":       true,
}

var exprRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

func ValidateDashboard(d *Dashboard) error {
	if d.ID == "" {
		return fmt.Errorf("id required")
	}
	if d.Name == "" {
		return fmt.Errorf("name required")
	}
	if d.Width == 0 {
		d.Width = 128
	}
	if d.Height == 0 {
		d.Height = 64
	}
	if d.RefreshInterval == 0 {
		d.RefreshInterval = 15000
	}
	if len(d.Widgets) == 0 {
		return fmt.Errorf("at least one widget required")
	}
	for i, w := range d.Widgets {
		if w.ID == "" {
			return fmt.Errorf("widget %d: id required", i)
		}
		if !validWidgetTypes[w.Type] {
			return fmt.Errorf("widget %s: invalid type %q", w.ID, w.Type)
		}
		if w.Expression == "" {
			return fmt.Errorf("widget %s: expression required", w.ID)
		}
		// Validate expression syntax
		if !exprRegex.MatchString(w.Expression) {
			return fmt.Errorf("widget %s: expression must contain at least one ${...} placeholder", w.ID)
		}
	}
	return nil
}

// Render evaluates all widget expressions against current data
func Render(d *Dashboard, store *store.Store) (*RenderedDashboard, error) {
	rendered := &RenderedDashboard{
		ID:        d.ID,
		Name:      d.Name,
		Width:     d.Width,
		Height:    d.Height,
		Widgets:   make([]RenderedWidget, len(d.Widgets)),
		RenderedAt: time.Now().UnixMilli(),
	}

	for i, w := range d.Widgets {
		val, text, status, err := evaluateExpression(w.Expression, store)
		if err != nil {
			// Return error placeholder but continue rendering other widgets
			rendered.Widgets[i] = RenderedWidget{
				ID:     w.ID,
				Type:   w.Type,
				X:      w.X,
				Y:      w.Y,
				Width:  w.Width,
				Height: w.Height,
				Value:  nil,
				Text:   "ERR: " + err.Error(),
				Unit:   w.Unit,
				Status: "crit",
			}
			continue
		}
		rendered.Widgets[i] = RenderedWidget{
			ID:     w.ID,
			Type:   w.Type,
			X:      w.X,
			Y:      w.Y,
			Width:  w.Width,
			Height: w.Height,
			Value:  val,
			Text:   text,
			Unit:   w.Unit,
			Status: status,
		}
	}
	return rendered, nil
}

type RenderedDashboard struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Width       int              `json:"width"`
	Height      int              `json:"height"`
	Widgets     []RenderedWidget `json:"widgets"`
	RenderedAt  int64            `json:"rendered_at"`
}

func evaluateExpression(expr string, st *store.Store) (interface{}, string, string, error) {
	matches := exprRegex.FindAllStringSubmatch(expr, -1)
	if len(matches) == 0 {
		return nil, expr, "ok", nil // static text
	}

	// Evaluate all placeholders and build result
	text := expr
	overallStatus := "ok"
	var lastVal interface{}

	for _, match := range matches {
		fullMatch := match[0]    // "${latency:monitor1}"
		innerMatch := match[1]   // "latency:monitor1"

		parts := strings.SplitN(innerMatch, ":", 2)
		if len(parts) != 2 {
			continue
		}

		source := parts[0]
		target := parts[1]

		var val interface{}
		var valText string
		var status string
		var err error

		switch source {
		case "latency":
			val, valText, status, err = evaluateLatency(target, st)
		case "metric":
			val, valText, status, err = evaluateMetric(target, st)
		default:
			err = fmt.Errorf("unknown source: %s", source)
		}

		if err != nil {
			return nil, "", "crit", err
		}

		// Replace placeholder in text
		text = strings.Replace(text, fullMatch, valText, 1)
		lastVal = val

		// Worst status wins
		if status == "crit" {
			overallStatus = "crit"
		} else if status == "warn" && overallStatus == "ok" {
			overallStatus = "warn"
		}
	}

	return lastVal, text, overallStatus, nil
}

func evaluateLatency(monitorID string, st *store.Store) (float64, string, string, error) {
	mon, err := st.GetMonitor(monitorID)
	if err != nil {
		return 0, "", "crit", err
	}
	if mon.LastStatus == 0 {
		return 0, "no data", "warn", nil
	}
	val := float64(mon.LastElapsedMs)
	text := fmt.Sprintf("%dms", mon.LastElapsedMs)
	status := "ok"
	if mon.LastStatus >= 500 || mon.LastStatus == 0 {
		status = "crit"
	} else if mon.LastStatus >= 400 {
		status = "warn"
	}
	return val, text, status, nil
}

func evaluateMetric(target string, st *store.Store) (float64, string, string, error) {
	// target format: "device_id.metric_name"
	parts := strings.SplitN(target, ".", 2)
	if len(parts) != 2 {
		return 0, "", "crit", fmt.Errorf("metric target must be device_id.metric_name")
	}
	deviceID := parts[0]
	_ = parts[1] // metricName - reserved for future use

	// Get latest metric for this device
	metrics, err := st.GetMetricsByDevice(deviceID, 1)
	if err != nil || len(metrics) == 0 {
		return 0, "no data", "warn", nil
	}
	m := metrics[0]

	// For now, return elapsed_ms as the metric value
	// TODO: support different metric fields
	val := float64(m.ElapsedMs)
	text := fmt.Sprintf("%dms", m.ElapsedMs)
	return val, text, "ok", nil
}

