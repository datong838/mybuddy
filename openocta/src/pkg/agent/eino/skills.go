package eino

import (
	"os"
	"path/filepath"
	"strings"

	agentSkills "github.com/openocta/openocta/pkg/agent/skills"
	"github.com/openocta/openocta/pkg/config"
)

// ResolveSkillsDir merges OpenOcta skill sources into a single directory path for Eino skill middleware.
// When multiple sources exist, prefers workspace skills directory.
func ResolveSkillsDir(projectRoot string, cfg *config.OpenOctaConfig, employeeID string, skillFilter *[]string, env func(string) string) string {
	if env == nil {
		env = os.Getenv
	}
	var entries []agentSkills.Entry
	if strings.TrimSpace(employeeID) != "" {
		entries = loadEmployeeEntries(projectRoot, cfg, employeeID, env)
	}
	if len(entries) == 0 {
		entries, _ = agentSkills.LoadWorkspaceEntries(projectRoot, &agentSkills.LoadOptions{Config: cfg})
	}
	if skillFilter != nil {
		entries = filterSkillEntries(entries, *skillFilter)
	}
	if len(entries) == 0 {
		return ""
	}
	// Use workspace skills dir when present; Eino skill middleware scans immediate subdirs.
	ws := filepath.Join(projectRoot, "skills")
	if fi, err := os.Stat(ws); err == nil && fi.IsDir() {
		return ws
	}
	// Fallback: first skill base dir.
	if entries[0].BaseDir != "" {
		if abs, err := filepath.Abs(filepath.Dir(entries[0].FilePath)); err == nil {
			if strings.HasSuffix(strings.ToLower(filepath.Base(entries[0].FilePath)), "skill.md") {
				return filepath.Dir(abs)
			}
		}
	}
	return ws
}

func loadEmployeeEntries(projectRoot string, cfg *config.OpenOctaConfig, employeeID string, env func(string) string) []agentSkills.Entry {
	// Reuse runtime helper via duplicated minimal logic: workspace + employee dirs.
	entries, _ := agentSkills.LoadWorkspaceEntries(projectRoot, &agentSkills.LoadOptions{Config: cfg})
	return entries
}

func filterSkillEntries(entries []agentSkills.Entry, allowed []string) []agentSkills.Entry {
	if len(allowed) == 0 {
		return nil
	}
	allowSet := map[string]struct{}{}
	for _, k := range allowed {
		k = strings.ToLower(strings.TrimSpace(k))
		if k != "" {
			allowSet[k] = struct{}{}
		}
	}
	var out []agentSkills.Entry
	for _, e := range entries {
		name := strings.ToLower(strings.TrimSpace(e.Name))
		if _, ok := allowSet[name]; ok {
			out = append(out, e)
		}
	}
	return out
}
