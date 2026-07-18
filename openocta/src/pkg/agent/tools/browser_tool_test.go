package tools_test

import (
	"testing"

	"github.com/openocta/openocta/pkg/agent/tool"
	"github.com/openocta/openocta/pkg/agent/tools"
	"github.com/openocta/openocta/pkg/config"
)

func TestBrowserToolsFromConfigDefaultEnabled(t *testing.T) {
	got := tools.BrowserToolsFromConfig(nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 browser tool, got %d", len(got))
	}
	if got[0].Name() != "browser" {
		t.Fatalf("unexpected tool name: %s", got[0].Name())
	}
}

func TestFilterOutBrowserTools(t *testing.T) {
	all := []tool.Tool{
		tools.BrowserToolViaInvoker{},
		&tools.BrowserTool{},
	}
	filtered := tools.FilterOutBrowserTools(all)
	if len(filtered) != 0 {
		t.Fatalf("expected no browser tools, got %d", len(filtered))
	}
}

func TestBrowserToolsFromConfigDisabled(t *testing.T) {
	enabled := false
	cfg := &config.OpenOctaConfig{
		Browser: &config.BrowserConfig{Enabled: &enabled},
	}
	if len(tools.BrowserToolsFromConfig(cfg)) != 0 {
		t.Fatal("expected no tools when browser.enabled=false")
	}
}
