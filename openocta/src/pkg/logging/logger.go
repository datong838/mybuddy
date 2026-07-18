package logging

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Logger provides simple structured logging.
type Logger struct {
	prefix string
	w      io.Writer
	mu     sync.Mutex
}

// New creates a logger writing to w with optional prefix.
func New(w io.Writer, prefix string) *Logger {
	if w == nil {
		w = os.Stderr
	}
	return &Logger{prefix: prefix, w: w}
}

// Sub returns a child logger with an additional prefix.
func (l *Logger) Sub(prefix string) *Logger {
	p := l.prefix
	if p != "" {
		p = p + " "
	}
	return &Logger{prefix: p + prefix, w: l.w}
}

// Info logs at info level.
func (l *Logger) Info(format string, args ...any) {
	l.log("INFO", format, args...)
}

// Warn logs at warn level.
func (l *Logger) Warn(format string, args ...any) {
	l.log("WARN", format, args...)
}

// Error logs at error level.
func (l *Logger) Error(format string, args ...any) {
	l.log("ERROR", format, args...)
}

// Debug logs at debug level (when verbose).
func (l *Logger) Debug(format string, args ...any) {
	l.log("DEBUG", format, args...)
}

func (l *Logger) log(level, format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s%s\n", ts, level, l.prefix, msg)
	fmt.Fprint(l.w, line)
}
