// Package logging levels: match tslog ordering (fatal=0, error=1, warn=2, info=3, debug=4, trace=5).
package logging

const (
	LevelFatal = 0
	LevelError = 1
	LevelWarn  = 2
	LevelInfo  = 3
	LevelDebug = 4
	LevelTrace = 5
)

// LevelName returns the log level name for _meta.logLevelName.
func LevelName(id int) string {
	switch id {
	case LevelFatal:
		return "FATAL"
	case LevelError:
		return "ERROR"
	case LevelWarn:
		return "WARN"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	case LevelTrace:
		return "TRACE"
	default:
		return "INFO"
	}
}

// LevelID returns the numeric level id for a level name (for _meta.logLevelId).
func LevelID(name string) int {
	switch name {
	case "FATAL":
		return LevelFatal
	case "ERROR":
		return LevelError
	case "WARN":
		return LevelWarn
	case "INFO":
		return LevelInfo
	case "DEBUG":
		return LevelDebug
	case "TRACE":
		return LevelTrace
	default:
		return LevelInfo
	}
}
