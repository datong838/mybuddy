package handlers

import (
	"strings"
	"testing"
)

func TestBuildLocalAgentSystemHintEmpty(t *testing.T) {
	if got := BuildLocalAgentSystemHint("hello", nil); got != "" {
		t.Fatalf("expected empty hint, got %q", got)
	}
}

func TestBuildLocalAgentSystemHintWithMention(t *testing.T) {
	got := BuildLocalAgentSystemHint("@cursor 实现登录", nil)
	if got == "" {
		t.Fatal("expected non-empty hint")
	}
	if !strings.Contains(got, "local_agent") {
		t.Fatalf("hint should mention local_agent tool: %q", got)
	}
	if !strings.Contains(got, "@cursor") {
		t.Fatalf("hint should include parsed mention: %q", got)
	}
}
