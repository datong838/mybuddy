package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/agent/tool"
	"github.com/openocta/openocta/pkg/config"
	"github.com/openocta/openocta/pkg/localagents"
)

// LocalAgentTool delegates tasks to external CLI agents (Cursor, Codex, etc.).
type LocalAgentTool struct {
	Config  *config.LocalAgentsConfig
	WorkDir string
	Env     func(string) string
}

func (LocalAgentTool) Name() string { return "local_agent" }

func (LocalAgentTool) Description() string {
	return "Delegate tasks to local CLI AI agents installed on this machine (openclaw, hermes, cursor, codex, opencode, trae). " +
		"Always analyze the user request and craft a complete task description before calling. " +
		"Actions: status (probe installed agents), run (execute one agent task), run_many (execute multiple tasks in parallel)."
}

func (LocalAgentTool) Schema() *tool.JSONSchema {
	return &tool.JSONSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "One of: status, run, run_many",
				"enum":        []string{"status", "run", "run_many"},
			},
			"agent": map[string]interface{}{
				"type":        "string",
				"description": "Agent id or alias for run (openclaw, hermes, cursor, codex, opencode, trae)",
			},
			"task": map[string]interface{}{
				"type":        "string",
				"description": "Complete, self-contained task text to pass to the CLI agent",
			},
			"tasks": map[string]interface{}{
				"type":        "array",
				"description": "For run_many: list of {agent, task} objects",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"agent": map[string]interface{}{"type": "string"},
						"task":  map[string]interface{}{"type": "string"},
					},
					"required": []string{"agent", "task"},
				},
			},
		},
		Required: []string{"action"},
	}
}

func (t LocalAgentTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	if t.Config != nil && t.Config.Enabled != nil && !*t.Config.Enabled {
		return &tool.ToolResult{Success: false, Output: "local agents are disabled in config"}, nil
	}

	action, _ := params["action"].(string)
	switch strings.TrimSpace(action) {
	case "status":
		return t.executeStatus(ctx)
	case "run":
		return t.executeRun(ctx, params)
	case "run_many":
		return t.executeRunMany(ctx, params)
	default:
		return &tool.ToolResult{Success: false, Output: "action must be status, run, or run_many"}, nil
	}
}

func (t LocalAgentTool) executeStatus(ctx context.Context) (*tool.ToolResult, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	report := localagents.ProbeAll(probeCtx, false)
	out, err := json.Marshal(report)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: err.Error()}, nil
	}
	return &tool.ToolResult{Success: true, Output: string(out)}, nil
}

func (t LocalAgentTool) executeRun(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	agentRaw, _ := params["agent"].(string)
	task, _ := params["task"].(string)
	agentID, ok := localagents.ResolveAgentID(agentRaw)
	if !ok {
		return &tool.ToolResult{Success: false, Output: fmt.Sprintf("unknown agent %q", strings.TrimSpace(agentRaw))}, nil
	}
	task = strings.TrimSpace(task)
	if task == "" {
		return &tool.ToolResult{Success: false, Output: "task is required for run"}, nil
	}
	return t.runSegments(ctx, []localagents.TaskSegment{{AgentID: agentID, Task: task}})
}

func (t LocalAgentTool) executeRunMany(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	rawTasks, ok := params["tasks"].([]interface{})
	if !ok || len(rawTasks) == 0 {
		return &tool.ToolResult{Success: false, Output: "tasks array is required for run_many"}, nil
	}
	var segments []localagents.TaskSegment
	for _, item := range rawTasks {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		agentRaw, _ := obj["agent"].(string)
		task, _ := obj["task"].(string)
		agentID, ok := localagents.ResolveAgentID(agentRaw)
		if !ok {
			return &tool.ToolResult{Success: false, Output: fmt.Sprintf("unknown agent %q", strings.TrimSpace(agentRaw))}, nil
		}
		task = strings.TrimSpace(task)
		if task == "" {
			return &tool.ToolResult{Success: false, Output: "each task must include non-empty task text"}, nil
		}
		segments = append(segments, localagents.TaskSegment{AgentID: agentID, Task: task})
	}
	if len(segments) == 0 {
		return &tool.ToolResult{Success: false, Output: "no valid tasks in run_many"}, nil
	}
	return t.runSegments(ctx, segments)
}

func (t LocalAgentTool) runSegments(ctx context.Context, segments []localagents.TaskSegment) (*tool.ToolResult, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	installed := localagents.AllInstalledMap(probeCtx)
	missing, ok := localagents.ValidateSegments(segments, installed)
	if !ok {
		return &tool.ToolResult{Success: false, Output: "agents not installed: " + strings.Join(missing, ", ")}, nil
	}

	workDir := strings.TrimSpace(t.WorkDir)
	if workDir == "" {
		workDir = "."
	}
	runOpts := localagents.RunOptions{WorkDir: workDir, Config: t.Config}
	results := localagents.RunSegments(ctx, segments, installed, runOpts)
	output := localagents.FormatCombinedOutput(results)
	success := true
	for _, r := range results {
		if r.Error != "" {
			success = false
			break
		}
	}
	return &tool.ToolResult{Success: success, Output: output}, nil
}

// AppendLocalAgentTool adds the local_agent tool when delegation is enabled.
func AppendLocalAgentTool(base []tool.Tool, cfg *config.OpenOctaConfig, workDir string, env func(string) string) []tool.Tool {
	if cfg == nil || !cfg.LocalAgents.IsEnabled() {
		return base
	}
	for _, t := range base {
		if t != nil && t.Name() == "local_agent" {
			return base
		}
	}
	if env == nil {
		env = func(string) string { return "" }
	}
	return append(base, LocalAgentTool{Config: cfg.LocalAgents, WorkDir: workDir, Env: env})
}
