package logging

import (
	"log"
	"strings"
)

// stdLogWriter implements io.Writer and forwards standard library log output
// into the global logging pipeline at a fixed level.
type stdLogWriter struct {
	level int
}

func (w *stdLogWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimRight(string(p), "\n")
	switch w.level {
	case LevelFatal, LevelError:
		Error("%s", msg)
	case LevelWarn:
		Warn("%s", msg)
	case LevelDebug, LevelTrace:
		Debug("%s", msg)
	default:
		Info("%s", msg)
	}
	return len(p), nil
}

// RedirectStdLog redirects the default standard library logger (log.Printf, etc.)
// into the global logger at the given level. Should be called after InitGlobal.
func RedirectStdLog(level int) {
	log.SetFlags(0)
	//log.SetOutput(io.Discard)
}
