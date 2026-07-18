package localagents

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const defaultProbeTTL = 5 * time.Minute

var (
	cacheMu      sync.RWMutex
	cachedReport *ProbeReport
	cachedAt     time.Time
	customPaths  map[string]string
)

// SetCustomPaths overrides executable paths per agent id (from config).
func SetCustomPaths(paths map[string]string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	customPaths = paths
}

// InvalidateCache clears the probe cache so the next read re-probes.
func InvalidateCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cachedReport = nil
	cachedAt = time.Time{}
}

// GetCachedReport returns the last probe report without re-running probes.
func GetCachedReport() *ProbeReport {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	if cachedReport == nil {
		return nil
	}
	cp := *cachedReport
	cp.Agents = append([]AgentProbeResult(nil), cachedReport.Agents...)
	return &cp
}

// ProbeAll detects all configured local agents. When force is false, returns cache if fresh.
func ProbeAll(ctx context.Context, force bool) ProbeReport {
	if !force {
		cacheMu.RLock()
		if cachedReport != nil && time.Since(cachedAt) < defaultProbeTTL {
			cp := *cachedReport
			cp.Agents = append([]AgentProbeResult(nil), cachedReport.Agents...)
			cacheMu.RUnlock()
			return cp
		}
		cacheMu.RUnlock()
	}

	home, _ := os.UserHomeDir()
	results := make([]AgentProbeResult, 0, len(allDefinitions()))
	for _, def := range allDefinitions() {
		results = append(results, probeOne(ctx, def, home))
	}
	report := ProbeReport{
		Agents:   results,
		ProbedAt: time.Now().UnixMilli(),
	}

	cacheMu.Lock()
	cachedReport = &report
	cachedAt = time.Now()
	cacheMu.Unlock()

	return report
}

func probeOne(parentCtx context.Context, def agentDefinition, home string) AgentProbeResult {
	ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
	defer cancel()

	res := AgentProbeResult{
		ID:          def.ID,
		Label:       def.Label,
		Aliases:     append([]string(nil), def.Aliases...),
		InvokeHint:  def.InvokeHint,
		ProbeMethod: buildProbeMethodDescription(def),
	}

	path, method := resolveExecutable(ctx, def, home)
	if path == "" {
		res.ProbeMethod = method
		return res
	}

	res.Path = path
	res.Installed = true
	res.ProbeMethod = method

	version := tryVersion(ctx, path, def.VersionArgs)
	if version != "" {
		res.Version = version
	}
	return res
}

func buildProbeMethodDescription(def agentDefinition) string {
	names := strings.Join(def.CLINames, "`, `")
	desc := "在 PATH 中查找 `" + names + "`"
	if def.ExtraPaths != nil {
		desc += "，并检查常见安装路径"
	}
	if def.MarkerDirs != nil {
		desc += "；也可依据配置目录（如 ~/.openclaw）辅助判断"
	}
	if len(def.VersionArgs) > 0 {
		desc += "；找到后执行 `" + strings.Join(def.CLINames, "` 或 `") + " " + def.VersionArgs[0] + "` 获取版本"
	}
	return desc
}

func resolveExecutable(ctx context.Context, def agentDefinition, home string) (path string, method string) {
	cacheMu.RLock()
	if customPaths != nil {
		if p := strings.TrimSpace(customPaths[def.ID]); p != "" {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				cacheMu.RUnlock()
				return p, "使用配置路径 customPaths[" + def.ID + "]: " + p
			}
		}
	}
	cacheMu.RUnlock()

	for _, name := range def.CLINames {
		if p, err := exec.LookPath(name); err == nil && p != "" {
			if def.ID == "cursor" && !looksLikeCursorCLI(ctx, p) {
				continue
			}
			return p, "PATH: `" + name + "` → " + p
		}
	}

	if def.ExtraPaths != nil {
		for _, candidate := range def.ExtraPaths(home) {
			if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
				if def.ID == "cursor" && !looksLikeCursorCLI(ctx, candidate) {
					continue
				}
				return candidate, "常见安装路径: " + candidate
			}
		}
	}

	if def.MarkerDirs != nil {
		for _, dir := range def.MarkerDirs(home) {
			if st, err := os.Stat(dir); err == nil && st.IsDir() {
				for _, name := range def.CLINames {
					if p, err := exec.LookPath(name); err == nil {
						if def.ID == "cursor" && !looksLikeCursorCLI(ctx, p) {
							continue
						}
						return p, "配置目录存在 " + dir + "，且 PATH 中有 `" + name + "`"
					}
				}
				return "", "配置目录存在 " + dir + "，但未在 PATH 中找到 `" + strings.Join(def.CLINames, "`/`") + "`"
			}
		}
	}

	return "", buildProbeMethodDescription(def) + "（未找到）"
}

func looksLikeCursorCLI(ctx context.Context, path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, "cursor") {
		return true
	}
	out := runCmdCombined(ctx, path, []string{"--version"})
	lower := strings.ToLower(out)
	return strings.Contains(lower, "cursor") || strings.Contains(path, "Cursor")
}

func tryVersion(ctx context.Context, path string, flags []string) string {
	for _, flag := range flags {
		out := runCmdCombined(ctx, path, []string{flag})
		if out != "" {
			return firstLineOrTwo(out)
		}
	}
	return ""
}

func runCmdCombined(ctx context.Context, path string, args []string) string {
	cmd := exec.CommandContext(ctx, path, args...)
	applyExecNoWindow(cmd)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	_ = cmd.Run()
	return strings.TrimSpace(buf.String())
}

func firstLineOrTwo(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	var b strings.Builder
	n := 0
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(ln)
		n++
		if n >= 2 {
			break
		}
	}
	out := b.String()
	if len(out) > 500 {
		out = out[:500] + "…"
	}
	return out
}

// InstalledAgent returns probe result for an id from cache or fresh probe.
func InstalledAgent(ctx context.Context, agentID string) (AgentProbeResult, bool) {
	report := ProbeAll(ctx, false)
	for _, a := range report.Agents {
		if a.ID == agentID && a.Installed {
			return a, true
		}
	}
	return AgentProbeResult{}, false
}

// AllInstalledMap returns agent id -> probe result for installed agents.
func AllInstalledMap(ctx context.Context) map[string]AgentProbeResult {
	report := ProbeAll(ctx, false)
	out := make(map[string]AgentProbeResult)
	for _, a := range report.Agents {
		if a.Installed {
			out[a.ID] = a
		}
	}
	return out
}
