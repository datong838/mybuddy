package embeddedmodels

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hybridgroup/yzma/pkg/message"
)

func TestChatRequestWantsTools(t *testing.T) {
	req := chatRequest{
		Tools: []openAIChatTool{{Type: "function", Function: struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
		}{Name: "demo"}}},
	}
	if !chatRequestWantsTools(req) {
		t.Fatal("expected tools enabled by default")
	}
	req.ToolChoice = json.RawMessage(`"none"`)
	if chatRequestWantsTools(req) {
		t.Fatal("expected tool_choice=none to disable tools")
	}
}

func TestChatMessagesToYzmaToolRoles(t *testing.T) {
	msgs := []chatMessage{
		{Role: "user", Content: json.RawMessage(`"hello"`)},
		{
			Role:    "assistant",
			Content: json.RawMessage(`null`),
			ToolCalls: []openAIChatToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{Name: "calc", Arguments: `{"x":"1"}`},
			}},
		},
		{Role: "tool", ToolCallID: "call_1", Content: json.RawMessage(`"2"`)},
	}
	out, err := chatMessagesToYzma(msgs, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(out))
	}
}

func TestSetRuntimeStateAllowsMultipleRunning(t *testing.T) {
	dir := t.TempDir()
	orig := os.Getenv("OPENOCTA_STATE_DIR")
	t.Setenv("OPENOCTA_STATE_DIR", dir)
	defer func() {
		if orig == "" {
			_ = os.Unsetenv("OPENOCTA_STATE_DIR")
		} else {
			t.Setenv("OPENOCTA_STATE_DIR", orig)
		}
	}()

	env := os.Getenv
	manifestPath := filepath.Join(dir, "embedded-models", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(`{
  "models": {
    "llama3.2-1b": {"id":"llama3.2-1b","running":false,"port":18902},
    "qwen3-0.6b": {"id":"qwen3-0.6b","running":false,"port":18903}
  }
}`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := setRuntimeState(env, "llama3.2-1b", true, 18902, ""); err != nil {
		t.Fatal(err)
	}
	if err := setRuntimeState(env, "qwen3-0.6b", true, 18903, ""); err != nil {
		t.Fatal(err)
	}
	m, err := loadManifest(env)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Models["qwen3-0.6b"].Running {
		t.Fatal("expected qwen running")
	}
	if !m.Models["llama3.2-1b"].Running {
		t.Fatal("expected llama still running")
	}
}

func TestYzmaToolCallsToOpenAI(t *testing.T) {
	calls := []message.ToolCall{{
		Type: "function",
		Function: message.ToolFunction{
			Name:      "add",
			Arguments: map[string]string{"a": "1", "b": "2"},
		},
	}}
	out := yzmaToolCallsToOpenAI(calls)
	if len(out) != 1 || out[0].Function.Name != "add" {
		t.Fatalf("unexpected conversion: %+v", out)
	}
}
