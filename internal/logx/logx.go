package logx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Level string

const (
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
	LevelAudit Level = "audit"
)

type Field struct {
	Key   string
	Value any
}

type Logger interface {
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Audit(msg string, fields ...Field)
}

type fileLogger struct {
	mu sync.Mutex
}

func New() Logger {
	return &fileLogger{}
}

func (l *fileLogger) Info(msg string, fields ...Field) {
	l.write(LevelInfo, msg, fields...)
}

func (l *fileLogger) Warn(msg string, fields ...Field) {
	l.write(LevelWarn, msg, fields...)
}

func (l *fileLogger) Error(msg string, fields ...Field) {
	l.write(LevelError, msg, fields...)
}

func (l *fileLogger) Audit(msg string, fields ...Field) {
	l.write(LevelAudit, msg, fields...)
}

func (l *fileLogger) write(level Level, msg string, fields ...Field) {
	l.mu.Lock()
	defer l.mu.Unlock()

	path, err := resolveLogPath(time.Now())
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}

	payload := map[string]any{
		"time":    time.Now().Format(time.RFC3339),
		"level":   level,
		"message": msg,
	}
	for _, field := range fields {
		if field.Key == "" {
			continue
		}
		payload[field.Key] = field.Value
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	_, _ = fmt.Fprintln(f, string(raw))
}

func resolveLogPath(now time.Time) (string, error) {
	baseDir := os.Getenv("LOCALAPPDATA")
	if baseDir == "" {
		var err error
		baseDir, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(baseDir, "WebShare", "logs", now.Format("2006-01-02")+".log"), nil
}
