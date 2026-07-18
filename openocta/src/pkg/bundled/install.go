package bundled

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/openocta/openocta/pkg/gateway/handlers"
	"github.com/openocta/openocta/pkg/installmetadata"
	"github.com/openocta/openocta/pkg/paths"
)

const (
	builtinType = "内置"
	builtinFrom = "openocta-bundled"
)

type mcpSiteConfig map[string]mcpSiteEntry

type mcpSiteEntry struct {
	Enabled *bool             `json:"enabled"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// InstallAll installs bundled peekaboo (macOS/Windows) and inner_skills on first run.
// Linux is skipped.
func InstallAll(env func(string) string) error {
	if runtime.GOOS == "linux" {
		return nil
	}
	if env == nil {
		env = os.Getenv
	}
	assetsDir := ResolveAssetsDir(env)
	if assetsDir == "" {
		slog.Debug("bundled: assets dir not found, skipping builtin install")
		return nil
	}

	if err := installInnerSkills(assetsDir, env); err != nil {
		slog.Warn("bundled: inner skills install failed", "error", err)
	}

	switch runtime.GOOS {
	case "darwin":
		if err := installPeekabooMac(assetsDir, env); err != nil {
			slog.Warn("bundled: peekaboo mac install failed", "error", err)
		}
	case "windows":
		if err := installPeekabooWin(assetsDir, env); err != nil {
			slog.Warn("bundled: peekaboo win install failed", "error", err)
		}
	}
	return nil
}

func installInnerSkills(assetsDir string, env func(string) string) error {
	dir := innerSkillsDir(assetsDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	installed := installmetadata.SkillInstallSet(env)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".zip") {
			continue
		}
		remoteID := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if _, ok := installed[remoteID]; ok {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			slog.Warn("bundled: read inner skill zip failed", "file", entry.Name(), "error", err)
			continue
		}
		folder, err := installSkillZip(data, remoteID, env)
		if err != nil {
			slog.Warn("bundled: install inner skill failed", "id", remoteID, "error", err)
			continue
		}
		_ = installmetadata.Append(env, "skill", remoteID, folder, builtinType)
		slog.Info("bundled: installed inner skill", "id", remoteID, "folder", folder)
	}
	return nil
}

func installPeekabooMac(assetsDir string, env func(string) string) error {
	srcDir := peekaboPlatformDir(assetsDir)
	binarySrc := filepath.Join(srcDir, "peekaboo")
	skillSrc := filepath.Join(srcDir, "SKILL.md")
	if _, err := os.Stat(binarySrc); err != nil {
		return fmt.Errorf("peekaboo binary missing: %w", err)
	}

	stateDir := paths.ResolveStateDir(env)
	toolsDir := filepath.Join(stateDir, "tools", "peekaboo")
	binaryDest := filepath.Join(toolsDir, "peekaboo")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return err
	}
	if err := copyFile(binarySrc, binaryDest, 0755); err != nil {
		return err
	}

	managedDir := handlers.ResolveManagedSkillsDir(env)
	skillDestDir := filepath.Join(managedDir, "peekaboo")
	if err := os.MkdirAll(skillDestDir, 0755); err != nil {
		return err
	}
	if _, err := os.Stat(skillSrc); err == nil {
		if err := copyFile(skillSrc, filepath.Join(skillDestDir, "SKILL.md"), 0644); err != nil {
			return err
		}
		writeSkillMeta(skillDestDir)
		_ = installmetadata.Append(env, "skill", "peekaboo", "peekaboo", builtinType)
	}

	if err := mergeMCPServer(env, "peekaboo", binaryDest, []string{"mcp"}, map[string]string{}); err != nil {
		return err
	}
	_ = installmetadata.Append(env, "mcp", "peekaboo", "peekaboo", builtinType)
	slog.Info("bundled: installed peekaboo (macOS)", "binary", binaryDest)
	return nil
}

func installPeekabooWin(assetsDir string, env func(string) string) error {
	srcDir := peekaboPlatformDir(assetsDir)
	stateDir := paths.ResolveStateDir(env)
	toolsDir := filepath.Join(stateDir, "tools", "peekaboo-win")

	zipPath := ""
	for _, name := range []string{"peekaboo-win.zip", "PeekabooWin.zip"} {
		candidate := filepath.Join(srcDir, name)
		if _, err := os.Stat(candidate); err == nil {
			zipPath = candidate
			break
		}
	}
	if zipPath != "" {
		if err := extractZipToDir(zipPath, toolsDir); err != nil {
			return fmt.Errorf("extract peekaboo-win zip: %w", err)
		}
	} else if _, err := os.Stat(toolsDir); os.IsNotExist(err) {
		slog.Warn("bundled: peekaboo-win zip not found, skipping binary extract", "dir", srcDir)
	}

	skillSrc := filepath.Join(srcDir, "SKILL.md")
	managedDir := handlers.ResolveManagedSkillsDir(env)
	skillDestDir := filepath.Join(managedDir, "peekaboo-win")
	if err := os.MkdirAll(skillDestDir, 0755); err != nil {
		return err
	}
	if _, err := os.Stat(skillSrc); err == nil {
		if err := copyFile(skillSrc, filepath.Join(skillDestDir, "SKILL.md"), 0644); err != nil {
			return err
		}
		writeSkillMeta(skillDestDir)
		_ = installmetadata.Append(env, "skill", "peekaboo-win", "peekaboo-win", builtinType)
	}

	configPath := filepath.Join(srcDir, "config.json")
	if data, err := os.ReadFile(configPath); err == nil && len(data) > 0 {
		if err := installMcpFromConfig(data, toolsDir, env); err != nil {
			return err
		}
	}
	slog.Info("bundled: installed peekaboo-win", "dir", toolsDir)
	return nil
}

func installMcpFromConfig(configData []byte, toolsDir string, env func(string) string) error {
	var siteCfg mcpSiteConfig
	if err := json.Unmarshal(configData, &siteCfg); err != nil {
		return fmt.Errorf("peekaboo win config.json: %w", err)
	}
	for key, entry := range siteCfg {
		command := strings.TrimSpace(entry.Command)
		if command == "" {
			continue
		}
		args := make([]string, len(entry.Args))
		copy(args, entry.Args)
		for i, arg := range args {
			args[i] = strings.ReplaceAll(arg, "${PEEKABOO_WIN_DIR}", toolsDir)
		}
		envVars := entry.Env
		if envVars == nil {
			envVars = map[string]string{}
		}
		if err := mergeMCPServer(env, key, command, args, envVars); err != nil {
			return err
		}
		_ = installmetadata.Append(env, "mcp", key, key, builtinType)
	}
	return nil
}

func mergeMCPServer(env func(string) string, key, command string, args []string, envVars map[string]string) error {
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveConfigPath(env, stateDir)
	snap, err := handlers.LoadConfigSnapshot(env)
	if err != nil {
		return err
	}
	cfgMap := handlers.ConfigSnapshotToMap(snap)
	if cfgMap == nil {
		cfgMap = map[string]interface{}{}
	}
	mcpRaw, _ := cfgMap["mcp"].(map[string]interface{})
	if mcpRaw == nil {
		mcpRaw = map[string]interface{}{}
		cfgMap["mcp"] = mcpRaw
	}
	serversRaw, _ := mcpRaw["servers"].(map[string]interface{})
	if serversRaw == nil {
		serversRaw = map[string]interface{}{}
		mcpRaw["servers"] = serversRaw
	}
	if _, exists := serversRaw[key]; exists {
		return nil
	}
	enabled := true
	serversRaw[key] = map[string]interface{}{
		"enabled": enabled,
		"command": command,
		"args":    args,
		"env":     envVars,
	}
	return handlers.WriteConfigMap(configPath, cfgMap)
}

func installSkillZip(zipData []byte, zipName string, env func(string) string) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return "", err
	}

	prefix := ""
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		clean := filepath.ToSlash(filepath.Clean(f.Name))
		clean = strings.TrimPrefix(clean, "/")
		if strings.Contains(clean, "..") {
			continue
		}
		if strings.ToLower(filepath.Base(clean)) == "skill.md" {
			dir := path.Dir(clean)
			if dir != "." {
				prefix = dir + "/"
			}
			break
		}
	}

	skillName := ""
	if prefix != "" {
		parts := strings.Split(strings.TrimSuffix(prefix, "/"), "/")
		if len(parts) > 0 {
			skillName = parts[len(parts)-1]
		}
	}
	if skillName == "" {
		skillName = strings.TrimSpace(zipName)
		skillName = strings.TrimSuffix(strings.TrimSuffix(skillName, ".zip"), ".ZIP")
	}
	if skillName == "" {
		skillName = "skill"
	}

	managedDir := handlers.ResolveManagedSkillsDir(env)
	targetDir := filepath.Join(managedDir, skillName)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		clean := filepath.ToSlash(filepath.Clean(f.Name))
		clean = strings.TrimPrefix(clean, "/")
		if strings.Contains(clean, "..") {
			continue
		}
		rel := clean
		if prefix != "" && strings.HasPrefix(clean, prefix) {
			rel = strings.TrimPrefix(clean, prefix)
		}
		if rel == "" || (prefix != "" && rel == clean) {
			continue
		}
		dest := filepath.Join(targetDir, filepath.FromSlash(rel))
		_ = os.MkdirAll(filepath.Dir(dest), 0755)
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, _ := io.ReadAll(io.LimitReader(rc, 1<<20))
		rc.Close()
		_ = os.WriteFile(dest, data, 0644)
	}
	writeSkillMeta(targetDir)
	return skillName, nil
}

func writeSkillMeta(skillDir string) {
	meta := map[string]string{"type": builtinType, "from": builtinFrom}
	metaData, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(filepath.Join(skillDir, ".openocta-meta.json"), metaData, 0644)
}

func copyFile(src, dest string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dest, data, mode); err != nil {
		return err
	}
	return nil
}

func extractZipToDir(zipPath, destDir string) error {
	data, err := os.ReadFile(zipPath)
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	_ = os.RemoveAll(destDir)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		clean := filepath.ToSlash(filepath.Clean(f.Name))
		clean = strings.TrimPrefix(clean, "/")
		if strings.Contains(clean, "..") {
			continue
		}
		out := filepath.Join(destDir, filepath.FromSlash(clean))
		_ = os.MkdirAll(filepath.Dir(out), 0755)
		rc, err := f.Open()
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(rc, 64<<20))
		rc.Close()
		mode := os.FileMode(0644)
		if strings.HasSuffix(clean, ".exe") || strings.HasSuffix(clean, ".js") {
			mode = 0755
		}
		_ = os.WriteFile(out, body, mode)
	}
	return nil
}
