package runtime

import (
	"testing"

	"github.com/openocta/openocta/pkg/agent/tool"
	agenttools "github.com/openocta/openocta/pkg/agent/tools"
)

func TestShouldRegisterWebToolsDisabled(t *testing.T) {
	if shouldRegisterWebTools(Options{}) {
		t.Fatal("web tools should be disabled")
	}
	on := true
	if shouldRegisterWebTools(Options{EnableWebTools: &on}) {
		t.Fatal("web tools should stay disabled even when EnableWebTools=true")
	}
}

func TestFilterOutBrowserToolsFromExtraTools(t *testing.T) {
	opts := Options{
		Tools: []tool.Tool{
			&agenttools.BrowserToolViaInvoker{},
			&agenttools.GatewayTool{},
		},
	}
	filtered := agenttools.FilterOutBrowserTools(opts.Tools)
	if len(filtered) != 1 || filtered[0].Name() != "gateway_config" {
		t.Fatalf("expected only gateway_config, got %#v", filtered)
	}
}

func TestFilterOutWebToolsFromExtraTools(t *testing.T) {
	opts := Options{
		Tools: []tool.Tool{
			&agenttools.WebSearchTool{},
			&agenttools.WebFetchTool{},
		},
	}
	filtered := agenttools.FilterOutWebTools(opts.Tools)
	if len(filtered) != 0 {
		t.Fatalf("expected all web tools filtered, got %d", len(filtered))
	}
}
