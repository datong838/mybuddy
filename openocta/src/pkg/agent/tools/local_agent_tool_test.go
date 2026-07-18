package tools

import (
	"context"
	"testing"

	"github.com/openocta/openocta/pkg/agent/tool"
	"github.com/openocta/openocta/pkg/config"
)

func TestLocalAgentToolRequiresAction(t *testing.T) {
	tool := LocalAgentTool{}
	res, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Success {
		t.Fatal("expected failure without action")
	}
}

func TestLocalAgentToolRunRequiresTask(t *testing.T) {
	tool := LocalAgentTool{}
	res, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "run",
		"agent":  "cursor",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Success {
		t.Fatal("expected failure without task")
	}
}

func TestLocalAgentToolUnknownAgent(t *testing.T) {
	tool := LocalAgentTool{}
	res, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "run",
		"agent":  "unknown-agent",
		"task":   "do something",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Success {
		t.Fatal("expected failure for unknown agent")
	}
}

func TestAppendLocalAgentToolDedupes(t *testing.T) {
	cfg := testConfigWithLocalAgentsEnabled()
	base := []tool.Tool{LocalAgentTool{}}
	out := AppendLocalAgentTool(base, cfg, ".", nil)
	if len(out) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(out))
	}
}

func testConfigWithLocalAgentsEnabled() *config.OpenOctaConfig {
	enabled := true
	return &config.OpenOctaConfig{
		LocalAgents: &config.LocalAgentsConfig{Enabled: &enabled},
	}
}
