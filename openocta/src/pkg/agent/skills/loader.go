// Package skills provides skill loading and management for agents.
package skills

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/openocta/openocta/pkg/config"
	"github.com/openocta/openocta/pkg/paths"
)

// Entry represents a loaded skill entry.
type Entry struct {
	Name            string
	Source          string
	FilePath        string
	BaseDir         string
	Metadata        *Metadata
	Frontmatter     map[string]string
	EmbeddedContent []byte // When non-nil, skill content is embedded (e.g. from binary); Handler should use this instead of reading FilePath.
}

// Metadata holds skill metadata parsed from frontmatter.
type Metadata struct {
	SkillKey   string
	PrimaryEnv string
	Always     *bool
	OS         []string
	Requires   *Requires
	Install    []InstallSpec
	Emoji      string
	Homepage   string
}

// Requires holds skill requirements.
type Requires struct {
	Bins    []string
	AnyBins []string
	Env     []string
	Config  []string
}

// InstallSpec holds skill installation specification.
type InstallSpec struct {
	ID    string
	Kind  string // "brew", "node", "go", "uv", "download"
	Bins  []string
	OS    []string
	Label string
	// Kind-specific fields
	Formula string // for brew
	Package string // for node/uv
	Module  string // for go
	URL     string // for download
}

// LoadOptions configures skill loading.
type LoadOptions struct {
	Config           *config.OpenOctaConfig
	ManagedSkillsDir string
	BundledSkillsDir string
}

// skillMergeKey is the deduplication key in LoadWorkspaceEntries. It follows the on-disk layout
// (<folder>/SKILL.md → folder basename; flat <stem>.md → stem) so sibling directories are not
// collapsed when frontmatter "name" duplicates across folders.
func skillMergeKey(e Entry) string {
	fn := filepath.Base(e.FilePath)
	if strings.EqualFold(fn, "SKILL.md") {
		k := filepath.Base(e.BaseDir)
		if k != "" && k != "." {
			return k
		}
	}
	return strings.TrimSuffix(fn, filepath.Ext(fn))
}

// LoadWorkspaceEntries loads skill entries from a workspace directory.
// Skills are loaded from multiple locations with the following precedence (lowest to highest):
// 1. Extra directories (from config.skills.load.extraDirs)
// 2. Bundled skills (built-in: default ./skills or OPENCLAW_BUNDLED_SKILLS_DIR)
// 3. Managed skills (~/.openclaw/skills)
// 4. Workspace skills (<workspace>/skills) - highest priority
//
// Merge key is skillMergeKey (on-disk identity). Higher-precedence sources overwrite the same key.
func LoadWorkspaceEntries(workspaceDir string, opts *LoadOptions) ([]Entry, error) {
	env := func(k string) string { return os.Getenv(k) }

	// Resolve directories
	managedSkillsDir := opts.ManagedSkillsDir
	if managedSkillsDir == "" {
		stateDir := paths.ResolveStateDir(env)
		managedSkillsDir = filepath.Join(stateDir, "skills")
	}

	bundledSkillsDir := opts.BundledSkillsDir
	if bundledSkillsDir == "" {
		bundledSkillsDir = resolveBundledSkillsDir(env)
	}

	workspaceSkillsDir := filepath.Join(workspaceDir, "skills")

	// Get extra directories from config
	var extraDirs []string
	if opts.Config != nil && opts.Config.Skills != nil && opts.Config.Skills.Load != nil {
		for _, dir := range opts.Config.Skills.Load.ExtraDirs {
			trimmed := strings.TrimSpace(dir)
			if trimmed != "" {
				extraDirs = append(extraDirs, ResolveUserPath(trimmed, env))
			}
		}
	}

	// TODO: Add plugin skill directories when plugin support is available
	// pluginSkillDirs := resolvePluginSkillDirs(workspaceDir, opts.Config)
	// extraDirs = append(extraDirs, pluginSkillDirs...)

	// Load skills from each directory with precedence
	// Priority order (lowest to highest):
	// 1. Extra directories (from config.skills.load.extraDirs)
	// 2. Bundled skills (built-in, shipped with install)
	// 3. Managed skills (~/.openclaw/skills)
	// 4. Workspace skills (<workspace>/skills) - highest
	merged := make(map[string]Entry)

	merge := func(skills []Entry) {
		for _, skill := range skills {
			k := skillMergeKey(skill)
			if k == "" {
				continue
			}
			merged[k] = skill
		}
	}

	// 1. Extra directories
	for _, dir := range extraDirs {
		skills, _ := loadSkillsFromDir(dir, "openclaw-extra")
		merge(skills)
	}

	// 2. Bundled skills (built-in, shipped with install)
	if bundledSkillsDir != "" {
		skills, _ := loadSkillsFromDir(bundledSkillsDir, "openclaw-bundled")
		merge(skills)
	}

	// 3. Managed skills (~/.openclaw/skills)
	skills, _ := loadSkillsFromDir(managedSkillsDir, "openclaw-managed")
	merge(skills)

	// 4. Workspace skills (<workspace>/skills) - highest precedence
	skills, _ = loadSkillsFromDir(workspaceSkillsDir, "openclaw-workspace")
	merge(skills)

	// Convert map to slice
	var result []Entry
	for _, skill := range merged {
		result = append(result, skill)
	}

	return result, nil
}

// LoadEntriesFromDir 是对内部 loadSkillsFromDir 的导出封装，便于其他包（如 employees）
// 在保持统一解析逻辑的前提下，从任意目录加载 SKILL.md。
// 返回空切片表示目录不存在或无可用 skills。
func LoadEntriesFromDir(dir string, source string) ([]Entry, error) {
	return loadSkillsFromDir(dir, source)
}

// loadSkillsFromDir loads skills from a directory.
// Returns empty slice if directory doesn't exist or has no skills.
func loadSkillsFromDir(dir string, source string) ([]Entry, error) {
	if dir == "" {
		return []Entry{}, nil
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}

	if !info.IsDir() {
		return []Entry{}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return []Entry{}, nil
	}

	var skills []Entry
	for _, entry := range entries {
		if entry.IsDir() {
			skillDir := filepath.Join(dir, entry.Name())
			skillFile := filepath.Join(skillDir, "SKILL.md")
			if _, err := os.Stat(skillFile); err == nil {
				skill, err := loadSkillFromDir(skillDir, source)
				if err == nil && skillMergeKey(skill) != "" {
					skills = append(skills, skill)
				}
			}
		} else if strings.HasSuffix(entry.Name(), ".md") {
			// Single file skill
			skillFile := filepath.Join(dir, entry.Name())
			skill, err := loadSkillFromFile(skillFile, source)
			if err == nil && skillMergeKey(skill) != "" {
				skills = append(skills, skill)
			}
		}
	}

	return skills, nil
}

// loadSkillsFromFS loads skills from an fs.FS (e.g. embedded). Stores content in EmbeddedContent.
// Uses path (not filepath) for fs.FS operations - fs.FS requires forward slashes on all platforms including Windows.
func loadSkillsFromFS(fsys fs.FS, basePath string, source string) ([]Entry, error) {
	entries, err := fs.ReadDir(fsys, basePath)
	if err != nil {
		return []Entry{}, nil
	}

	var skills []Entry
	for _, entry := range entries {
		if entry.IsDir() {
			skillDir := entry.Name()
			if basePath != "." {
				skillDir = path.Join(basePath, skillDir)
			}
			skillFile := path.Join(skillDir, "SKILL.md")
			if _, err := fs.Stat(fsys, skillFile); err == nil {
				skill, err := loadSkillFromFSFile(fsys, skillFile, source)
				if err == nil && skillMergeKey(skill) != "" {
					skills = append(skills, skill)
				}
			}
		} else if strings.HasSuffix(entry.Name(), ".md") {
			skillFile := entry.Name()
			if basePath != "." {
				skillFile = path.Join(basePath, skillFile)
			}
			skill, err := loadSkillFromFSFile(fsys, skillFile, source)
			if err == nil && skillMergeKey(skill) != "" {
				skills = append(skills, skill)
			}
		}
	}
	return skills, nil
}

// LoadEntriesFromFS 在给定 fs.FS 下，从 basePath 开始递归加载 skills。
func LoadEntriesFromFS(fsys fs.FS, basePath string, source string) ([]Entry, error) {
	return loadSkillsFromFS(fsys, basePath, source)
}

// loadSkillFromFSFile loads a skill from a file within fs.FS, storing content in EmbeddedContent.
// skillPath uses forward slashes (fs.FS convention, works on all platforms including Windows).
func loadSkillFromFSFile(fsys fs.FS, skillPath string, source string) (Entry, error) {
	data, err := fs.ReadFile(fsys, skillPath)
	if err != nil {
		return Entry{}, err
	}

	dir := path.Dir(skillPath)
	name := path.Base(dir)
	if name == "." || name == "/" {
		name = strings.TrimSuffix(path.Base(skillPath), ".md")
	}

	frontmatter := parseFrontmatter(string(data))
	if displayName, ok := frontmatter["displayName"]; ok && displayName != "" {
		name = displayName
	} else if nameFromFM, ok := frontmatter["name"]; ok && nameFromFM != "" {
		name = nameFromFM
	}

	metadata := parseMetadata(frontmatter)

	return Entry{
		Name:            name,
		Source:          source,
		FilePath:        skillPath,
		BaseDir:         dir,
		Metadata:        metadata,
		Frontmatter:     frontmatter,
		EmbeddedContent: data,
	}, nil
}

// loadSkillFromDir loads a skill from a directory containing SKILL.md.
func loadSkillFromDir(skillDir string, source string) (Entry, error) {
	skillFile := filepath.Join(skillDir, "SKILL.md")
	return loadSkillFromFile(skillFile, source)
}

// loadSkillFromFile loads a skill from a SKILL.md file.
func loadSkillFromFile(skillFile string, source string) (Entry, error) {
	data, err := os.ReadFile(skillFile)
	if err != nil {
		return Entry{}, err
	}

	// Parse frontmatter (simplified - just extract name for now)
	// TODO: Full frontmatter parsing with metadata extraction
	name := filepath.Base(filepath.Dir(skillFile))
	if name == "." || name == "/" {
		name = strings.TrimSuffix(filepath.Base(skillFile), ".md")
	}

	// Try to extract name from frontmatter
	frontmatter := parseFrontmatter(string(data))
	if displayName, ok := frontmatter["displayName"]; ok && displayName != "" {
		name = displayName
	} else if nameFromFM, ok := frontmatter["name"]; ok && nameFromFM != "" {
		name = nameFromFM
	}

	baseDir := filepath.Dir(skillFile)
	if baseDir == "." {
		baseDir = filepath.Dir(skillFile)
	}

	metadata := parseMetadata(frontmatter)

	return Entry{
		Name:        name,
		Source:      source,
		FilePath:    skillFile,
		BaseDir:     baseDir,
		Metadata:    metadata,
		Frontmatter: frontmatter,
	}, nil
}

// parseFrontmatter parses YAML frontmatter from markdown content.
// Simplified implementation - extracts basic key-value pairs.
func parseFrontmatter(content string) map[string]string {
	result := make(map[string]string)

	lines := strings.Split(content, "\n")
	inFrontmatter := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				break
			}
		}
		if inFrontmatter {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Remove quotes if present
				value = strings.Trim(value, `"'`)
				result[key] = value
			}
		}
	}

	return result
}

// parseMetadata parses OpenOcta metadata from frontmatter.
func parseMetadata(frontmatter map[string]string) *Metadata {
	// TODO: Parse JSON5 metadata field
	m := &Metadata{
		Requires: &Requires{},
	}
	if v, ok := frontmatter["skillKey"]; ok {
		m.SkillKey = v
	}
	return m
}

// resolveBundledSkillsDir resolves the bundled skills directory.
// Checks in order:
// 1. Environment override (OPENCLAW_BUNDLED_SKILLS_DIR)
// 2. Current working directory ./skills (built-in default)
// 3. Executable directory ../skills (for compiled binaries)
// 4. Common system locations
func resolveBundledSkillsDir(env func(string) string) string {
	// 1. Check environment override
	override := strings.TrimSpace(env("OPENCLAW_BUNDLED_SKILLS_DIR"))
	if override != "" {
		return override
	}

	// 2. Check current working directory ./skills (built-in default)
	cwd, err := os.Getwd()
	if err == nil {
		localSkills := filepath.Join(cwd, "skills")
		if info, err := os.Stat(localSkills); err == nil && info.IsDir() {
			if looksLikeSkillsDir(localSkills) {
				return localSkills
			}
		}
	}

	// 3. Try to find skills directory relative to executable
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		sibling := filepath.Join(execDir, "skills")
		if info, err := os.Stat(sibling); err == nil && info.IsDir() {
			if looksLikeSkillsDir(sibling) {
				return sibling
			}
		}
	}

	// 4. Try to find in common system locations
	candidates := []string{
		"/usr/local/share/openclaw/skills",
		"/opt/openclaw/skills",
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			if looksLikeSkillsDir(candidate) {
				return candidate
			}
		}
	}

	return ""
}

// looksLikeSkillsDir checks if a directory looks like a skills directory.
// A skills directory should contain at least one .md file or a subdirectory with SKILL.md.
func looksLikeSkillsDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Check for .md files (single-file skills)
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			return true
		}

		// Check for subdirectories with SKILL.md
		if entry.IsDir() {
			skillFile := filepath.Join(dir, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillFile); err == nil {
				return true
			}
		}
	}

	return false
}

// ResolveUserPath resolves a user path (expands ~ and resolves to absolute).
func ResolveUserPath(input string, env func(string) string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return trimmed
	}

	if strings.HasPrefix(trimmed, "~") {
		home := env("HOME")
		if home == "" {
			home = env("USERPROFILE")
		}
		if home == "" {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				home = homeDir
			}
		}
		if home != "" {
			trimmed = strings.Replace(trimmed, "~", home, 1)
		}
	}

	abs, err := filepath.Abs(trimmed)
	if err == nil {
		return abs
	}
	return trimmed
}
