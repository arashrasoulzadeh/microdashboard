package logger

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"time"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

type Logger struct {
	logger *log.Logger
	level  Level
}

type Fields map[string]interface{}

func NewLogger(level Level) *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", 0),
		level:  level,
	}
}

func (l *Logger) SetOutput(w *bytes.Buffer) {
	l.logger.SetOutput(w)
}

func (l *Logger) log(level Level, msg string, fields Fields) {
	if level < l.level {
		return
	}

	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"level":     level.String(),
		"message":   msg,
	}

	for k, v := range fields {
		entry[k] = v
	}

	data, _ := json.Marshal(entry)
	l.logger.Println(string(data))
}

func (l *Logger) Debug(msg string, fields Fields) {
	l.log(DebugLevel, msg, fields)
}

func (l *Logger) Info(msg string, fields Fields) {
	l.log(InfoLevel, msg, fields)
}

func (l *Logger) Warn(msg string, fields Fields) {
	l.log(WarnLevel, msg, fields)
}

func (l *Logger) Error(msg string, fields Fields) {
	l.log(ErrorLevel, msg, fields)
}

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return "unknown"
	}
}

var defaultLogger = NewLogger(InfoLevel)

func SetDefault(l *Logger) {
	defaultLogger = l
}

func Debug(msg string, fields Fields) {
	defaultLogger.Debug(msg, fields)
}

func Info(msg string, fields Fields) {
	defaultLogger.Info(msg, fields)
}

func Warn(msg string, fields Fields) {
	defaultLogger.Warn(msg, fields)
}

func Error(msg string, fields Fields) {
	defaultLogger.Error(msg, fields)
}