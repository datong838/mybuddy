package localagents

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type agentDefinition struct {
	ID          string
	Label       string
	Aliases     []string
	CLINames    []string
	ExtraPaths  func(home string) []string
	MarkerDirs  func(home string) []string
	VersionArgs []string
	InvokeHint  string
}

func allDefinitions() []agentDefinition {
	return []agentDefinition{
		{
			ID:          "openclaw",
			Label:       "OpenClaw",
			Aliases:     []string{"openclaw", "claw"},
			CLINames:    []string{"openclaw"},
			MarkerDirs:  func(home string) []string { return []string{filepath.Join(home, ".openclaw")} },
			VersionArgs: []string{"--version"},
			InvokeHint:  `openclaw agent --local --agent main --message "任务" --json`,
		},
		{
			ID:          "hermes",
			Label:       "Hermes Agent",
			Aliases:     []string{"hermes"},
			CLINames:    []string{"hermes"},
			MarkerDirs:  func(home string) []string { return []string{filepath.Join(home, ".hermes")} },
			VersionArgs: []string{"--version", "-V", "-v"},
			InvokeHint:  `hermes -z "任务"`,
		},
		{
			ID:          "cursor",
			Label:       "Cursor",
			Aliases:     []string{"cursor"},
			CLINames:    []string{"agent", "cursor-agent"},
			ExtraPaths:  cursorExtraPaths,
			MarkerDirs:  func(home string) []string { return []string{filepath.Join(home, ".cursor")} },
			VersionArgs: []string{"--version", "-V", "-v"},
			InvokeHint:  `agent -p --output-format text "任务"`,
		},
		{
			ID:          "codex",
			Label:       "Codex",
			Aliases:     []string{"codex"},
			CLINames:    []string{"codex"},
			MarkerDirs:  func(home string) []string { return []string{filepath.Join(home, ".codex")} },
			VersionArgs: []string{"--version", "-V", "-v"},
			InvokeHint:  `codex exec --output-format text "任务"`,
		},
		{
			ID:       "opencode",
			Label:    "OpenCode",
			Aliases:  []string{"opencode"},
			CLINames: []string{"opencode"},
			MarkerDirs: func(home string) []string {
				return []string{
					filepath.Join(home, ".config", "opencode"),
					filepath.Join(home, ".opencode"),
				}
			},
			VersionArgs: []string{"--version", "-V", "-v"},
			InvokeHint:  `opencode run "任务"`,
		},
		{
			ID:          "trae",
			Label:       "Trae",
			Aliases:     []string{"trae", "trae-cli"},
			CLINames:    []string{"trae-cli", "trae"},
			VersionArgs: []string{"--help", "--version", "-V", "-v"},
			InvokeHint:  `trae-cli run "任务"`,
		},
	}
}

func cursorExtraPaths(home string) []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/Applications/Cursor.app/Contents/Resources/app/bin/cursor",
			"/Applications/Cursor.app/Contents/Resources/app/bin/agent",
			filepath.Join(home, ".local", "bin", "agent"),
			filepath.Join(home, ".local", "bin", "cursor-agent"),
		}
	case "windows":
		localApp := os.Getenv("LOCALAPPDATA")
		if localApp == "" {
			return nil
		}
		return []string{
			filepath.Join(localApp, "Programs", "cursor", "resources", "app", "bin", "agent.cmd"),
			filepath.Join(localApp, "Programs", "cursor", "resources", "app", "bin", "cursor-agent.cmd"),
		}
	default:
		return []string{
			filepath.Join(home, ".local", "bin", "agent"),
			filepath.Join(home, ".local", "bin", "cursor-agent"),
		}
	}
}

func aliasToID() map[string]string {
	out := make(map[string]string)
	for _, def := range allDefinitions() {
		out[def.ID] = def.ID
		for _, alias := range def.Aliases {
			out[normalizeAlias(alias)] = def.ID
		}
	}
	return out
}

func definitionByID(id string) (agentDefinition, bool) {
	for _, def := range allDefinitions() {
		if def.ID == id {
			return def, true
		}
	}
	return agentDefinition{}, false
}

func ResolveAgentID(aliasOrID string) (string, bool) {
	id, ok := aliasToID()[normalizeAlias(aliasOrID)]
	return id, ok
}

func normalizeAlias(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
