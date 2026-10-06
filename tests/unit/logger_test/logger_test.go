package logger_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"microdashboard/internal/logger"
)

func TestLogger_Levels(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewLogger(logger.DebugLevel)
	l.SetOutput(&buf)

	l.Debug("debug msg", logger.Fields{"key": "val"})
	l.Info("info msg", logger.Fields{})
	l.Warn("warn msg", nil)
	l.Error("error msg", logger.Fields{"err": "test"})

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	if len(lines) != 4 {
		t.Errorf("expected 4 log lines, got %d", len(lines))
	}

	for i, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Errorf("line %d: invalid JSON: %v", i, err)
			continue
		}
		if entry["timestamp"] == nil {
			t.Errorf("line %d: missing timestamp", i)
		}
		if entry["level"] == nil {
			t.Errorf("line %d: missing level", i)
		}
		if entry["message"] == nil {
			t.Errorf("line %d: missing message", i)
		}
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewLogger(logger.WarnLevel)
	l.SetOutput(&buf)

	l.Debug("debug msg", nil)
	l.Info("info msg", nil)
	l.Warn("warn msg", nil)
	l.Error("error msg", nil)

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))

	if len(lines) != 2 {
		t.Errorf("expected 2 lines (warn, error), got %d", len(lines))
	}
}

func TestLogger_SetDefault(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewLogger(logger.InfoLevel)
	l.SetOutput(&buf)
	logger.SetDefault(l)

	logger.Info("test default", nil)

	output := buf.String()
	if output == "" {
		t.Error("default logger not working")
	}
}

func TestFields_Map(t *testing.T) {
	fields := logger.Fields{"key1": "val1", "key2": 123}
	if fields["key1"] != "val1" {
		t.Error("fields map not working")
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level logger.Level
		want  string
	}{
		{logger.DebugLevel, "debug"},
		{logger.InfoLevel, "info"},
		{logger.WarnLevel, "warn"},
		{logger.ErrorLevel, "error"},
		{logger.Level(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("Level.String() = %q, want %q", got, tt.want)
		}
	}
}

func TestLogger_WithTimestamp(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewLogger(logger.InfoLevel)
	l.SetOutput(&buf)

	before := time.Now()
	l.Info("timestamp test", nil)
	after := time.Now()

	output := buf.String()
	var entry map[string]interface{}
	json.Unmarshal([]byte(output), &entry)
	
	ts, ok := entry["timestamp"].(string)
	if !ok {
		t.Fatal("timestamp not string")
	}
	
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t.Fatalf("invalid timestamp format: %v", err)
	}
	
	if parsed.Before(before) || parsed.After(after) {
		t.Errorf("timestamp out of range: %v", parsed)
	}
}

func TestLogger_NilFields(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewLogger(logger.InfoLevel)
	l.SetOutput(&buf)

	l.Info("nil fields", nil)

	output := buf.String()
	if output == "" {
		t.Error("logger with nil fields failed")
	}
}

func TestLogger_DefaultLogger(t *testing.T) {
	var buf bytes.Buffer
	l := logger.NewLogger(logger.InfoLevel)
	l.SetOutput(&buf)
	logger.SetDefault(l)

	logger.Info("default test", logger.Fields{"test": "value"})

	output := buf.String()
	if output == "" {
		t.Error("default logger failed")
	}
}