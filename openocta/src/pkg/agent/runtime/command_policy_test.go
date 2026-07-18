package runtime

import "testing"

func TestEvaluateCommandAccessCompoundCommand(t *testing.T) {
	policy := &ResolvedCommandPolicy{
		Enabled:       true,
		DefaultPolicy: "ask",
		AllowRules:    []CommandRule{{Pattern: "ls", Type: "command"}},
		AskRules:      []CommandRule{{Pattern: "rm", Type: "command"}},
	}

	action, explicit := policy.EvaluateCommandAccess("hostname && uname -a")
	if action != "ask" || explicit {
		t.Fatalf("expected default ask without explicit rule, got action=%q explicit=%v", action, explicit)
	}

	action, explicit = policy.EvaluateCommandAccess("ls -la && pwd")
	if action != "ask" || explicit {
		t.Fatalf("expected default ask when a segment is not allow-listed, got action=%q explicit=%v", action, explicit)
	}

	action, explicit = policy.EvaluateCommandAccess("rm foo")
	if action != "ask" || !explicit {
		t.Fatalf("expected explicit ask for rm, got action=%q explicit=%v", action, explicit)
	}
}
