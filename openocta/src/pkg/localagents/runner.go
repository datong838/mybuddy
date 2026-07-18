package localagents

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/openocta/openocta/pkg/config"
)

// RunOptions configures external agent execution.
type RunOptions struct {
	Timeout time.Duration
	WorkDir string
	Config  *config.LocalAgentsConfig
}

func defaultRunOptions(opts RunOptions) RunOptions {
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Minute
	}
	if opts.Config != nil && opts.Config.TimeoutSeconds != nil && *opts.Config.TimeoutSeconds > 0 {
		opts.Timeout = time.Duration(*opts.Config.TimeoutSeconds) * time.Second
	}
	return opts
}

func isAgentAllowed(id string, cfg *config.LocalAgentsConfig) bool {
	if cfg != nil && cfg.Enabled != nil && !*cfg.Enabled {
		return false
	}
	if cfg == nil || len(cfg.Allowed) == 0 {
		return true
	}
	for _, a := range cfg.Allowed {
		if strings.EqualFold(strings.TrimSpace(a), id) {
			return true
		}
	}
	return false
}

// RunSegments executes task segments in parallel and returns combined results in order.
func RunSegments(ctx context.Context, segments []TaskSegment, installed map[string]AgentProbeResult, opts RunOptions) []SegmentResult {
	opts = defaultRunOptions(opts)
	results := make([]SegmentResult, len(segments))
	var wg sync.WaitGroup
	for i, seg := range segments {
		wg.Add(1)
		go func(idx int, s TaskSegment) {
			defer wg.Done()
			results[idx] = runOne(ctx, s, installed, opts)
		}(i, seg)
	}
	wg.Wait()
	return results
}

func runOne(parentCtx context.Context, seg TaskSegment, installed map[string]AgentProbeResult, opts RunOptions) SegmentResult {
	probe, ok := installed[seg.AgentID]
	if !ok {
		return SegmentResult{
			AgentID: seg.AgentID,
			Label:   seg.AgentID,
			Error:   "agent not installed on this machine",
		}
	}
	if !isAgentAllowed(seg.AgentID, opts.Config) {
		return SegmentResult{
			AgentID: seg.AgentID,
			Label:   probe.Label,
			Error:   "agent not allowed by localAgents.allowed config",
		}
	}

	ctx, cancel := context.WithTimeout(parentCtx, opts.Timeout)
	defer cancel()

	path := probe.Path
	args, err := buildInvokeArgs(seg.AgentID, seg.Task)
	if err != nil {
		return SegmentResult{AgentID: seg.AgentID, Label: probe.Label, Error: err.Error()}
	}

	cmd := exec.CommandContext(ctx, path, args...)
	applyExecNoWindow(cmd)
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if out == "" {
		out = strings.TrimSpace(stderr.String())
	}
	if runErr != nil {
		errMsg := runErr.Error()
		if out != "" {
			errMsg = out + "\n(" + errMsg + ")"
		}
		return SegmentResult{AgentID: seg.AgentID, Label: probe.Label, Output: out, Error: errMsg}
	}
	return SegmentResult{AgentID: seg.AgentID, Label: probe.Label, Output: out}
}

func buildInvokeArgs(agentID, task string) ([]string, error) {
	task = strings.TrimSpace(task)
	if task == "" {
		return nil, fmt.Errorf("empty task")
	}
	switch agentID {
	case "openclaw":
		return []string{"agent", "--local", "--agent", "main", "--message", task, "--json"}, nil
	case "hermes":
		return []string{"-z", task}, nil
	case "cursor":
		return []string{"-p", "--output-format", "text", task}, nil
	case "codex":
		return []string{"exec", "--output-format", "text", task}, nil
	case "opencode":
		return []string{"run", task}, nil
	case "trae":
		return []string{"run", task}, nil
	default:
		return nil, fmt.Errorf("unknown agent %q", agentID)
	}
}

// FormatCombinedOutput renders segment results for chat display.
func FormatCombinedOutput(results []SegmentResult) string {
	var b strings.Builder
	for i, r := range results {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("## ")
		b.WriteString(r.Label)
		b.WriteString("\n\n")
		if r.Error != "" {
			b.WriteString("**Error:** ")
			b.WriteString(r.Error)
			if r.Output != "" && r.Output != r.Error {
				b.WriteString("\n\n")
				b.WriteString(r.Output)
			}
			continue
		}
		if r.Output == "" {
			b.WriteString("_(no output)_")
		} else {
			b.WriteString(r.Output)
		}
	}
	return b.String()
}
