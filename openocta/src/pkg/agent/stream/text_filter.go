package stream

import "strings"

// IsLeakedAssistantText reports raw model/API payloads that must not be shown to users.
func IsLeakedAssistantText(text string) bool {
	t := strings.TrimSpace(text)
	if t == "" {
		return false
	}
	lower := strings.ToLower(t)
	if strings.HasPrefix(lower, "(empty response") || strings.HasPrefix(lower, "(emptyresponse") {
		return true
	}
	if strings.Contains(lower, "stop_reason") &&
		(strings.Contains(lower, "'type':'thinking'") || strings.Contains(lower, `"type":"thinking"`)) {
		return true
	}
	if strings.Contains(lower, "input_tokens") && strings.Contains(lower, "output_tokens") &&
		strings.Contains(lower, "content") && strings.Contains(lower, "thinking") {
		return true
	}
	return false
}
