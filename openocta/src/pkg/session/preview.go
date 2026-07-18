// Package session provides transcript preview reading.
// Mirrors readSessionPreviewItemsFromTranscript from src/gateway/session-utils.fs.ts.
package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// PreviewItem is a single preview entry (role + truncated text).
type PreviewItem struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

const previewMaxLines = 50
const previewReadBytes = 64 * 1024

// ReadSessionPreviewItems reads the last N messages from a transcript and returns preview items.
func ReadSessionPreviewItems(transcriptPath string, maxItems, maxChars int) []PreviewItem {
	if maxItems < 1 {
		maxItems = 12
	}
	if maxItems > 50 {
		maxItems = 50
	}
	if maxChars < 20 {
		maxChars = 240
	}
	if maxChars > 2000 {
		maxChars = 2000
	}
	f, err := os.Open(transcriptPath)
	if err != nil {
		return nil
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil || stat.Size() == 0 {
		return nil
	}
	readStart := int64(0)
	readLen := stat.Size()
	if readLen > previewReadBytes {
		readStart = readLen - previewReadBytes
		readLen = previewReadBytes
	}
	buf := make([]byte, readLen)
	_, err = f.ReadAt(buf, readStart)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(buf), "\n")
	var tail []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			tail = append(tail, line)
		}
	}
	if len(tail) > previewMaxLines {
		tail = tail[len(tail)-previewMaxLines:]
	}
	var items []PreviewItem
	for i := len(tail) - 1; i >= 0 && len(items) < maxItems; i-- {
		line := tail[i]
		var parsed struct {
			Message *struct {
				Role    string      `json:"role"`
				Content interface{} `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal([]byte(line), &parsed) != nil {
			var m TranscriptMessage
			if json.Unmarshal([]byte(line), &m) != nil {
				continue
			}
			text := extractTextFromContent(m.Content)
			if text == "" {
				continue
			}
			role := m.Role
			if role == "" {
				role = "assistant"
			}
			items = append(items, PreviewItem{
				Role: role,
				Text: truncate(text, maxChars),
			})
			continue
		}
		if parsed.Message == nil {
			continue
		}
		text := extractTextFromContentAny(parsed.Message.Content)
		if text == "" {
			continue
		}
		role := parsed.Message.Role
		if role == "" {
			role = "assistant"
		}
		items = append(items, PreviewItem{
			Role: role,
			Text: truncate(text, maxChars),
		})
	}
	// Items are newest-first; reverse for chronological order
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return items
}

func extractTextFromContent(content []ContentBlock) string {
	for _, b := range content {
		if (b.Type == "text" || b.Type == "output_text" || b.Type == "input_text") && b.Text != "" {
			return strings.TrimSpace(b.Text)
		}
	}
	return ""
}

func extractTextFromContentAny(content interface{}) string {
	if s, ok := content.(string); ok {
		return strings.TrimSpace(s)
	}
	arr, ok := content.([]interface{})
	if !ok {
		return ""
	}
	for _, v := range arr {
		m, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		txt, _ := m["text"].(string)
		if (t == "text" || t == "output_text" || t == "input_text") && txt != "" {
			return strings.TrimSpace(txt)
		}
	}
	return ""
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// ResolveTranscriptPathForPreview resolves transcript path for a session key.
func ResolveTranscriptPathForPreview(key, storePath string, sessionFile string, agentID string, env func(string) string) string {
	if sessionFile != "" {
		sessionsDir := filepath.Dir(storePath)
		joined := filepath.Join(sessionsDir, sessionFile)
		if !strings.Contains(joined, "..") {
			return joined
		}
	}
	sessionID := key
	if strings.HasPrefix(strings.ToLower(key), "agent:") {
		parts := strings.SplitN(key, ":", 3)
		if len(parts) >= 3 {
			sessionID = parts[2]
		}
	}
	return ResolveSessionFilePath(sessionID, &SessionPathOptions{AgentID: agentID}, env)
}
