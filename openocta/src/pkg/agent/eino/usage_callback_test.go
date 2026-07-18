package eino

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"

	cbutils "github.com/cloudwego/eino/utils/callbacks"
)

func TestUsageCallbackHandlerBuilds(t *testing.T) {
	handler := newUsageCallbackHandler()
	if handler == nil {
		t.Fatal("handler is nil")
	}
	_ = cbutils.NewHandlerHelper().ChatModel(&cbutils.ModelCallbackHandler{}).Handler()
}

func TestModelAndToolNameHelpers(t *testing.T) {
	if got := modelNameFromConfig(&model.Config{Model: " gpt-4 "}); got != "gpt-4" {
		t.Fatalf("model = %q", got)
	}
	if got := toolNameFromRunInfo(&callbacks.RunInfo{Name: " bash "}); got != "bash" {
		t.Fatalf("tool = %q", got)
	}
}

func TestAppendJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sess.jsonl")
	if err := appendJSONL(path, []byte(`{"type":"tool_call","name":"read"}`)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected data")
	}
}

func TestUsageTrackingDisabledWithoutSession(t *testing.T) {
	if usageTrackingEnabled(t.Context()) {
		t.Fatal("expected tracking disabled without adk session context")
	}
}
