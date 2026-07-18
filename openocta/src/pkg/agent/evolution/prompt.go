package evolution

import "strings"

// AugmentSystemPrompt injects frozen evolution snapshot blocks into the base system prompt.
func AugmentSystemPrompt(base string, snap Snapshot) string {
	out := strings.TrimSpace(base)
	if snap.Soul != "" {
		out = joinPromptBlocks(snap.Soul, out)
	}
	if snap.Prompt != "" {
		out = joinPromptBlocks(out, snap.Prompt)
	}
	if snap.Memory != "" {
		out = joinPromptBlocks(out, snap.Memory)
	}
	if snap.User != "" {
		out = joinPromptBlocks(out, snap.User)
	}
	return strings.TrimSpace(out)
}

func joinPromptBlocks(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + "\n\n" + b
	}
}
