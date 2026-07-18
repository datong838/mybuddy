package handlers

import (
	"testing"

	"github.com/openocta/openocta/pkg/agent/tools"
)

func TestChatRunDisallowedToolsAlwaysBlocksWebTools(t *testing.T) {
	cases := []ChatRunResources{
		{},
		{Configured: true, WebSearch: false},
		{Configured: true, WebSearch: true},
		{WebSearch: true},
	}
	for _, res := range cases {
		disallowed := chatRunDisallowedTools(res)
		if len(disallowed) != len(tools.WebToolNames) {
			t.Fatalf("expected %d disallowed tools for %#v, got %v", len(tools.WebToolNames), res, disallowed)
		}
	}
}
