package handlers

import (
	"context"
	"os"
	"time"

	"github.com/openocta/openocta/pkg/agent"
	octool "github.com/openocta/openocta/pkg/agent/tool"
	"github.com/openocta/openocta/pkg/agent/tools"
	"github.com/openocta/openocta/pkg/config"
	"github.com/openocta/openocta/pkg/localagents"
)

func applyLocalAgentsConfig(cfg *config.OpenOctaConfig) {
	if cfg == nil || cfg.LocalAgents == nil || len(cfg.LocalAgents.CustomPaths) == 0 {
		return
	}
	localagents.SetCustomPaths(cfg.LocalAgents.CustomPaths)
}

func localAgentsConfigFromContext(ctx *Context) *config.LocalAgentsConfig {
	cfg := loadConfigFromContext(ctx)
	if cfg == nil {
		return nil
	}
	return cfg.LocalAgents
}

// AppendLocalAgentToolsForSession adds local_agent when delegation is enabled.
func AppendLocalAgentToolsForSession(base []octool.Tool, cfg *config.OpenOctaConfig, agentID string, env func(string) string) []octool.Tool {
	if env == nil {
		env = os.Getenv
	}
	workDir := "."
	if cfg != nil {
		workDir = agent.ResolveAgentWorkspaceDir(cfg, agentID, env)
		if workDir == "" {
			workDir = "."
		}
	}
	return tools.AppendLocalAgentTool(base, cfg, workDir, env)
}

// LocalAgentsProbeHandler handles localAgents.probe.
func LocalAgentsProbeHandler(opts HandlerOpts) error {
	force := false
	if f, ok := opts.Params["force"].(bool); ok {
		force = f
	}
	applyLocalAgentsConfig(loadConfigFromContext(opts.Context))
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	report := localagents.ProbeAll(ctx, force)
	opts.Respond(true, report, nil, nil)
	return nil
}

// LocalAgentsStatusHandler handles localAgents.status (cached snapshot).
func LocalAgentsStatusHandler(opts HandlerOpts) error {
	report := localagents.GetCachedReport()
	if report == nil {
		applyLocalAgentsConfig(loadConfigFromContext(opts.Context))
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		r := localagents.ProbeAll(ctx, true)
		report = &r
	}
	opts.Respond(true, report, nil, map[string]interface{}{"cached": true})
	return nil
}
