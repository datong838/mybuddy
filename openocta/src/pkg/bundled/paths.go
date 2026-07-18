package bundled

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const bundledAssetsDirName = "bundled"

// ResolveAssetsDir returns the directory containing shipped peekaboo / inner_skills assets.
// Resolution order:
//  1. OPENOCTA_BUNDLED_ASSETS_DIR
//  2. <executable>/bundled
//  3. macOS .app: ../Resources/bundled
//  4. Dev: <cwd>/deploy (repo layout)
func ResolveAssetsDir(env func(string) string) string {
	if env == nil {
		env = os.Getenv
	}
	if override := strings.TrimSpace(env("OPENOCTA_BUNDLED_ASSETS_DIR")); override != "" {
		if info, err := os.Stat(override); err == nil && info.IsDir() {
			return override
		}
	}
	for _, dir := range assetDirCandidates(env) {
		if looksLikeAssetsDir(dir) {
			return dir
		}
	}
	return ""
}

func assetDirCandidates(env func(string) string) []string {
	var out []string
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		out = append(out,
			filepath.Join(execDir, bundledAssetsDirName),
			filepath.Join(execDir, "resources", bundledAssetsDirName),
		)
		if runtime.GOOS == "darwin" && strings.Contains(execDir, ".app/Contents/MacOS") {
			out = append(out, filepath.Clean(filepath.Join(execDir, "..", "Resources", bundledAssetsDirName)))
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		out = append(out,
			filepath.Join(cwd, "deploy"),
			filepath.Join(cwd, "..", "deploy"),
			filepath.Join(cwd, "bundled"),
		)
	}
	_ = env
	return out
}

func looksLikeAssetsDir(dir string) bool {
	if dir == "" {
		return false
	}
	peekabo := filepath.Join(dir, "peekabo")
	inner := filepath.Join(dir, "inner_skills")
	if info, err := os.Stat(peekabo); err == nil && info.IsDir() {
		return true
	}
	if info, err := os.Stat(inner); err == nil && info.IsDir() {
		return true
	}
	return false
}

func peekaboPlatformDir(assetsDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(assetsDir, "peekabo", "macos")
	case "windows":
		return filepath.Join(assetsDir, "peekabo", "win")
	default:
		return ""
	}
}

func innerSkillsDir(assetsDir string) string {
	return filepath.Join(assetsDir, "inner_skills")
}
