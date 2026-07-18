package localagents

import "testing"

func TestResolveAgentID(t *testing.T) {
	id, ok := ResolveAgentID("cursor")
	if !ok || id != "cursor" {
		t.Fatalf("cursor: got %q ok=%v", id, ok)
	}
	id, ok = ResolveAgentID("claw")
	if !ok || id != "openclaw" {
		t.Fatalf("claw alias: got %q ok=%v", id, ok)
	}
	_, ok = ResolveAgentID("unknown")
	if ok {
		t.Fatal("expected unknown agent to fail")
	}
}
