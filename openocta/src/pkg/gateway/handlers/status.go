package handlers

import (
	"time"
)

// StatusSummary is a minimal status payload (compatible with protocol).
type StatusSummary struct {
	OK       bool                   `json:"ok"`
	Ts       int64                  `json:"ts"`
	Channels map[string]interface{} `json:"channels"`
	Agents   []interface{}          `json:"agents"`
	Sessions interface{}            `json:"sessions"`
}

// DefaultStatusSummary returns a minimal status for Phase 2c.
func DefaultStatusSummary() *StatusSummary {
	return &StatusSummary{
		OK:       true,
		Ts:       time.Now().UnixMilli(),
		Channels: map[string]interface{}{},
		Agents:   []interface{}{},
		Sessions: map[string]interface{}{},
	}
}
