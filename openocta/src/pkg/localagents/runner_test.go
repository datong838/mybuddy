package localagents

import "testing"

func TestBuildInvokeArgs(t *testing.T) {
	args, err := buildInvokeArgs("cursor", "hello")
	if err != nil || len(args) < 2 {
		t.Fatalf("cursor args: %v err=%v", args, err)
	}
	if args[0] != "-p" {
		t.Fatalf("unexpected cursor args: %v", args)
	}

	_, err = buildInvokeArgs("unknown", "x")
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestIsAgentAllowedDefault(t *testing.T) {
	if !isAgentAllowed("cursor", nil) {
		t.Fatal("expected allowed by default")
	}
}
