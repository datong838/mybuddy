package tools

import (
	"testing"

	"github.com/openocta/openocta/pkg/agent/tool"
)

func TestWebToolsFromConfigDisabled(t *testing.T) {
	got := WebToolsFromConfig(nil, ".")
	if len(got) != 0 {
		t.Fatalf("expected no web tools while disabled, got %d", len(got))
	}
}

func TestFilterOutWebTools(t *testing.T) {
	all := []tool.Tool{
		&WebSearchTool{},
		&WebFetchTool{},
		&BrowserTool{},
	}
	filtered := FilterOutWebTools(all)
	if len(filtered) != 1 || filtered[0].Name() != "browser" {
		t.Fatalf("expected only browser tool, got %#v", filtered)
	}
}
