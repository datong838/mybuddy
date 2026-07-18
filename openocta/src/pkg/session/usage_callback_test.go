package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSessionCostSummaryTokenUsageWithModel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.jsonl")
	content := `{"type":"session","version":2,"id":"demo","timestamp":"2026-01-01T00:00:00Z"}
{"type":"token_usage","timestamp":"2026-06-01T12:00:00.173Z","model":"gpt-test","provider":"openai","input":100,"output":20,"cacheRead":5,"totalTokens":125}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	summary, err := LoadSessionCostSummary(path, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalTokens != 125 {
		t.Fatalf("total tokens = %d", summary.TotalTokens)
	}
	if len(summary.ModelUsage) != 1 {
		t.Fatalf("model usage len = %d", len(summary.ModelUsage))
	}
	if summary.ModelUsage[0].Model != "gpt-test" {
		t.Fatalf("model = %q", summary.ModelUsage[0].Model)
	}
	if summary.ModelUsage[0].Totals.TotalTokens != 125 {
		t.Fatalf("model tokens = %d", summary.ModelUsage[0].Totals.TotalTokens)
	}
}

func TestLoadSessionCostSummaryToolCallLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.jsonl")
	content := `{"type":"session","version":2,"id":"demo","timestamp":"2026-01-01T00:00:00Z"}
{"type":"tool_call","timestamp":"2026-06-01T12:00:00.173Z","name":"bash"}
{"type":"tool_call","timestamp":"2026-06-01T12:01:00.173Z","name":"bash"}
{"type":"tool_call","timestamp":"2026-06-01T12:02:00.173Z","name":"read"}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	summary, err := LoadSessionCostSummary(path, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ToolUsage == nil {
		t.Fatal("expected tool usage")
	}
	if summary.ToolUsage.TotalCalls != 3 {
		t.Fatalf("total calls = %d", summary.ToolUsage.TotalCalls)
	}
	if summary.ToolUsage.UniqueTools != 2 {
		t.Fatalf("unique tools = %d", summary.ToolUsage.UniqueTools)
	}
	if summary.MessageCounts == nil || summary.MessageCounts.ToolCalls != 3 {
		t.Fatalf("message tool calls = %+v", summary.MessageCounts)
	}
}
