package logx

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

const (
	logFileName   = "web-share.log"
	maxLogSize    = 512 * 1024
	maxLogBackups = 5
)

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

	path, err := resolveLogPath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	if err := rotateIfNeeded(path); err != nil {
		return
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	_, _ = fmt.Fprintln(f, formatLine(time.Now(), level, msg, fields...))
}

func resolveLogPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exePath), logFileName), nil
}

func rotateIfNeeded(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Size() < maxLogSize {
		return nil
	}
	_ = os.Remove(rotatedLogPath(path, maxLogBackups))
	for i := maxLogBackups - 1; i >= 1; i-- {
		src := rotatedLogPath(path, i)
		if _, err := os.Stat(src); err == nil {
			if err := os.Rename(src, rotatedLogPath(path, i+1)); err != nil {
				return err
			}
		}
	}
	return os.Rename(path, rotatedLogPath(path, 1))
}

func rotatedLogPath(path string, index int) string {
	return path + "." + strconv.Itoa(index)
}

func formatLine(now time.Time, level Level, msg string, fields ...Field) string {
	parts := []string{
		now.Format("2006-01-02 15:04:05"),
		"[" + strings.ToUpper(string(level)) + "]",
		msg,
	}
	for _, field := range fields {
		if field.Key == "" {
			continue
		}
		parts = append(parts, field.Key+"="+formatFieldValue(field.Value))
	}
	return strings.Join(parts, " ")
}

func formatFieldValue(value any) string {
	switch v := value.(type) {
	case string:
		return strconv.Quote(v)
	case fmt.Stringer:
		return strconv.Quote(v.String())
	case []string:
		return strconv.Quote(strings.Join(v, ", "))
	default:
		return strconv.Quote(fmt.Sprint(value))
	}
}
