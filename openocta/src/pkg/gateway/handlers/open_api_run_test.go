package handlers

import (
	"testing"

	"github.com/openocta/openocta/pkg/agent/stream"
)

func TestTryCompleteOpenAPIRun(t *testing.T) {
	ch := registerOpenAPIRunWaiter("run-1")
	defer unregisterOpenAPIRunWaiter("run-1")

	if !tryCompleteOpenAPIRun("run-1", "hello", &stream.Usage{InputTokens: 3, OutputTokens: 5}, nil) {
		t.Fatal("tryCompleteOpenAPIRun() = false, want true")
	}

	select {
	case result := <-ch:
		if result.Content != "hello" {
			t.Fatalf("content = %q, want hello", result.Content)
		}
		if result.PromptTokens != 3 || result.CompletionTokens != 5 {
			t.Fatalf("tokens = %d/%d, want 3/5", result.PromptTokens, result.CompletionTokens)
		}
	default:
		t.Fatal("expected result on waiter channel")
	}

	if tryCompleteOpenAPIRun("run-1", "again", nil, nil) {
		t.Fatal("second tryCompleteOpenAPIRun() should be false")
	}
}
