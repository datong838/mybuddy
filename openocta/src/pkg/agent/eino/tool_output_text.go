package eino

import "strings"

// IsLeakedToolOutputText detects shell / execute tool stdout that was incorrectly
// streamed as assistant markdown (and would pollute A2UI surfaces).
func IsLeakedToolOutputText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "<persisted-output>") {
		return true
	}
	prefixes := []string{
		"command exited with non-zero code",
		"[stderr]:",
		"[command failed with exit code",
		"output too large (",
		"full output saved to:",
		"launching skill:",
		"launch chromium:",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	// JSON tool_search style results belong in toolResult rows, not assistant bubbles.
	if strings.HasPrefix(trimmed, `{"matches":`) {
		return true
	}
	return false
}
