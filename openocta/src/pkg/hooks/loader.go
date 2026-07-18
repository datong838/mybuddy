package hooks

import (
	"os"
	"path/filepath"
	"strings"
)

// Entry represents a discovered hook entry.
type Entry struct {
	Name        string
	HandlerPath string
	Events      []string
	Export      string
}

// LoadWorkspaceEntry discovers a HOOK.md in a directory.
func LoadWorkspaceEntry(dir string) (*Entry, error) {
	hookPath := filepath.Join(dir, "HOOK.md")
	if _, err := os.Stat(hookPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	data, err := os.ReadFile(hookPath)
	if err != nil {
		return nil, err
	}
	return parseHookFrontmatter(string(data), dir)
}

// parseHookFrontmatter extracts metadata from HOOK.md frontmatter.
func parseHookFrontmatter(content string, dir string) (*Entry, error) {
	// Phase 6a: minimal parser - name from dir, events from content
	name := filepath.Base(dir)
	lines := strings.Split(content, "\n")
	var inMetadata bool
	var events []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			inMetadata = !inMetadata
			continue
		}
		if inMetadata && strings.HasPrefix(trimmed, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
		}
		if inMetadata && strings.Contains(trimmed, "\"events\"") {
			if idx := strings.Index(trimmed, "["); idx >= 0 {
				if end := strings.Index(trimmed[idx:], "]"); end >= 0 {
					inner := trimmed[idx+1 : idx+end]
					for _, p := range strings.Split(inner, ",") {
						s := strings.Trim(strings.TrimSpace(p), "\"")
						if s != "" {
							events = append(events, s)
						}
					}
				}
			}
		}
	}
	handlerPath := filepath.Join(dir, "handler.ts")
	return &Entry{
		Name:        name,
		HandlerPath: handlerPath,
		Events:      events,
		Export:      "default",
	}, nil
}

// BundledDir returns the path to bundled hooks (placeholder).
func BundledDir() string {
	return "hooks/bundled"
}
