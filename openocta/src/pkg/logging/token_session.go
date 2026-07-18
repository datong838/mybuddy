// Package logging: append token_usage JSON lines to session transcript files.
package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TokenUsageSessionLine is one JSON line appended to ~/.openclaw/agents/<agentID>/sessions/<sessionID>.jsonl.
// Field names match transcript usage style (camelCase) and session contract.
type TokenUsageSessionLine struct {
	Type          string `json:"type"`                    // "token_usage"
	Timestamp     string `json:"timestamp"`               // RFC3339
	SessionID     string `json:"sessionId,omitempty"`     // session identifier
	RequestID     string `json:"requestId,omitempty"`     // request identifier
	Model         string `json:"model,omitempty"`         // model name
	Provider      string `json:"provider,omitempty"`      // provider id
	Input         int64  `json:"input"`                   // input tokens
	Output        int64  `json:"output"`                  // output tokens
	CacheRead     int64  `json:"cacheRead,omitempty"`     // cache read tokens
	CacheCreation int64  `json:"cacheCreation,omitempty"` // cache creation tokens
	TotalTokens   int64  `json:"totalTokens"`             // total tokens
}

var tokenSessionMu sync.Mutex

// AppendTokenUsageToSession appends a single token_usage JSON line to the session transcript file at path.
// Path should be the full path to the session file (e.g. ~/.openclaw/agents/main/sessions/<sessionID>.jsonl).
// Creates parent directory if needed. Uses O_APPEND so concurrent appends from the same process are safe.
func AppendTokenUsageToSession(path string, line TokenUsageSessionLine) error {
	if path == "" {
		return nil
	}
	line.Type = "token_usage"
	if line.Timestamp == "" {
		line.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	payload, err := json.Marshal(line)
	if err != nil {
		return err
	}
	tokenSessionMu.Lock()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	tokenSessionMu.Unlock()
	if err != nil {
		return err
	}
	_, err = f.Write(append(payload, '\n'))
	_ = f.Close()
	return err
}
