// Package logging: parse JSON log lines (mirrors src/logging/parse-log-line.ts).
package logging

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ParsedLogLine holds extracted fields from a JSON log line.
type ParsedLogLine struct {
	Time      string
	Level     string
	Subsystem string
	Module    string
	Message   string
	Raw       string
}

var numericKeyRE = regexp.MustCompile(`^\d+$`)

func extractMessage(m map[string]interface{}) string {
	var parts []string
	for k, v := range m {
		if !numericKeyRE.MatchString(k) {
			continue
		}
		switch x := v.(type) {
		case string:
			parts = append(parts, x)
		case nil:
			continue
		default:
			b, _ := json.Marshal(v)
			parts = append(parts, string(b))
		}
	}
	return strings.Join(parts, " ")
}

// ParseLogLine parses a raw JSON log line and returns time, level, message, etc.
// Returns nil if the line is not valid JSON.
func ParseLogLine(raw string) *ParsedLogLine {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil
	}
	meta, _ := parsed["_meta"].(map[string]interface{})
	levelName := ""
	if meta != nil {
		if n, ok := meta["logLevelName"].(string); ok {
			levelName = strings.ToLower(n)
		}
	}
	timeStr := ""
	if t, ok := parsed["time"].(string); ok {
		timeStr = t
	} else if meta != nil {
		if d, ok := meta["date"].(string); ok {
			timeStr = d
		}
	}
	return &ParsedLogLine{
		Time:    timeStr,
		Level:   levelName,
		Message: extractMessage(parsed),
		Raw:     raw,
	}
}
