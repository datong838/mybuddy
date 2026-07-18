// Package runtime provides skill integration for agent runtime.
package runtime

import (
	"os"
	"path/filepath"
	"strings"

	agentSkills "github.com/openocta/openocta/pkg/agent/skills"
	"github.com/openocta/openocta/pkg/config"
	"github.com/openocta/openocta/pkg/employees"
	"github.com/openocta/openocta/pkg/paths"
)

// LoadSkillRegistrationsWithBaseDirs returns skill base dirs for sandbox allowlists.
func LoadSkillRegistrationsWithBaseDirs(workspaceDir string, cfg *config.OpenOctaConfig) ([]string, []string) {
	entries, err := LoadWorkspaceSkillEntries(workspaceDir, cfg)
	if err != nil || len(entries) == 0 {
		return nil, nil
	}
	return nil, uniqueAbsSkillBaseDirs(entries)
}

func LoadWorkspaceSkillEntries(workspaceDir string, cfg *config.OpenOctaConfig) ([]agentSkills.Entry, error) {
	opts := &agentSkills.LoadOptions{Config: cfg}
	return agentSkills.LoadWorkspaceEntries(workspaceDir, opts)
}

// LoadSkillsForWorkspace loads skill entries from workspace, managed, and bundled dirs.
func LoadSkillsForWorkspace(workspaceDir string, cfg *config.OpenOctaConfig) ([]agentSkills.Entry, error) {
	return LoadWorkspaceSkillEntries(workspaceDir, cfg)
}

func LoadEmployeeSkillEntries(workspaceDir string, cfg *config.OpenOctaConfig, employeeID string, env func(string) string) []agentSkills.Entry {
	employeeID = strings.TrimSpace(employeeID)
	if employeeID == "" {
		return nil
	}
	if env == nil {
		env = os.Getenv
	}
	var entries []agentSkills.Entry
	if ws, err := LoadWorkspaceSkillEntries(workspaceDir, cfg); err == nil {
		entries = append(entries, ws...)
	}
	empDir := filepath.Join(paths.ResolveStateDir(env), "employee_skills", employeeID)
	if empDir != "" {
		if empEntries, err := agentSkills.LoadEntriesFromDir(empDir, "openocta-employee"); err == nil {
			entries = mergeSkillEntries(entries, empEntries)
		}
	}
	if m, err := employees.LoadManifest(employeeID, env); err == nil && m != nil && len(m.SkillIDs) > 0 {
		entries = FilterSkillEntries(entries, m.SkillIDs)
	}
	return entries
}

func mergeSkillEntries(a, b []agentSkills.Entry) []agentSkills.Entry {
	byKey := map[string]agentSkills.Entry{}
	for _, e := range a {
		byKey[skillKey(e)] = e
	}
	for _, e := range b {
		byKey[skillKey(e)] = e
	}
	out := make([]agentSkills.Entry, 0, len(byKey))
	for _, e := range byKey {
		out = append(out, e)
	}
	return out
}

func skillKey(e agentSkills.Entry) string {
	return strings.ToLower(strings.TrimSpace(e.Name))
}

func uniqueAbsSkillBaseDirs(entries []agentSkills.Entry) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, e := range entries {
		d := strings.TrimSpace(e.BaseDir)
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out
}

func FilterSkillEntries(entries []agentSkills.Entry, allowed []string) []agentSkills.Entry {
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
		if _, ok := allowSet[strings.ToLower(e.Name)]; ok {
			out = append(out, e)
		}
	}
	return out
}

// BuildSystemPromptSkillsSection loads skills and returns a system-prompt block.
func BuildSystemPromptSkillsSection(projectRoot string, opts Options) string {
	if !opts.EnableSkills {
		return ""
	}
	env := opts.Env
	if env == nil {
		env = os.Getenv
	}
	var entries []agentSkills.Entry
	if strings.TrimSpace(opts.EmployeeID) != "" {
		entries = LoadEmployeeSkillEntries(projectRoot, opts.Config, opts.EmployeeID, env)
	}
	if len(entries) == 0 {
		entries, _ = LoadWorkspaceSkillEntries(projectRoot, opts.Config)
	}
	if opts.SkillFilter != nil {
		entries = FilterSkillEntries(entries, *opts.SkillFilter)
	}
	prompt := agentSkills.BuildPrompt(entries)
	if prompt == "" {
		return ""
	}
	return prompt + "\n\n可通过 skill 工具按名称激活；匹配关键词时也会自动注入技能内容。"
}

// BuildSkillsPrompt is used by tests and callers.
func BuildSkillsPrompt(entries []agentSkills.Entry, cfg *config.OpenOctaConfig) string {
	_ = cfg
	return agentSkills.BuildPrompt(entries)
}

// ApplySkillEnvOverrides applies skill environment variable overrides.
func ApplySkillEnvOverrides(entries []agentSkills.Entry, cfg *config.OpenOctaConfig) func() {
	return agentSkills.ApplyEnvOverrides(entries, cfg)
}
