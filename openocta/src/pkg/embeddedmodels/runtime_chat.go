package embeddedmodels

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hybridgroup/yzma/pkg/message"
)

type openAIChatTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type openAIChatToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatMessage struct {
	Role       string               `json:"role"`
	Content    json.RawMessage      `json:"content"`
	ToolCalls  []openAIChatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
	Name       string               `json:"name,omitempty"`
}

type chatRequest struct {
	Model      string           `json:"model"`
	Messages   []chatMessage    `json:"messages"`
	MaxTokens  int              `json:"max_tokens"`
	Stream     *bool            `json:"stream,omitempty"`
	Thinking   json.RawMessage  `json:"thinking,omitempty"`
	Tools      []openAIChatTool `json:"tools,omitempty"`
	ToolChoice json.RawMessage  `json:"tool_choice,omitempty"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type chatGenerationResult struct {
	Content          string
	ToolCalls        []openAIChatToolCall
	FinishReason     string
	PromptTokens     int
	CompletionTokens int
}

func chatRequestWantsTools(req chatRequest) bool {
	if len(req.Tools) == 0 {
		return false
	}
	if len(req.ToolChoice) == 0 {
		return true
	}
	var asString string
	if err := json.Unmarshal(req.ToolChoice, &asString); err == nil {
		s := strings.ToLower(strings.TrimSpace(asString))
		return s != "none"
	}
	return true
}

func chatMessageContentString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return strings.TrimSpace(string(raw))
}

func openAIToolsToDefinitions(tools []openAIChatTool) []message.ToolDefinition {
	out := make([]message.ToolDefinition, 0, len(tools))
	for _, t := range tools {
		if strings.TrimSpace(t.Type) == "" {
			t.Type = "function"
		}
		params := map[string]interface{}{}
		if len(t.Function.Parameters) > 0 {
			_ = json.Unmarshal(t.Function.Parameters, &params)
		}
		out = append(out, message.ToolDefinition{
			Type: t.Type,
			Function: message.ToolFunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  params,
			},
		})
	}
	return out
}

func formatToolsForPrompt(tools []message.ToolDefinition) string {
	b, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(b)
}

func toolArgumentsMap(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}
	}
	var generic map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &generic); err != nil {
		return map[string]string{"_raw": raw}
	}
	out := make(map[string]string, len(generic))
	for k, v := range generic {
		switch val := v.(type) {
		case string:
			out[k] = val
		default:
			b, _ := json.Marshal(val)
			out[k] = string(b)
		}
	}
	return out
}

func yzmaToolCallsToOpenAI(calls []message.ToolCall) []openAIChatToolCall {
	out := make([]openAIChatToolCall, 0, len(calls))
	for i, call := range calls {
		args := call.Function.Arguments
		if args == nil {
			args = map[string]string{}
		}
		argsJSON, _ := json.Marshal(args)
		typ := strings.TrimSpace(call.Type)
		if typ == "" {
			typ = "function"
		}
		out = append(out, openAIChatToolCall{
			ID:   fmt.Sprintf("call_%d_%d", time.Now().UnixNano(), i),
			Type: typ,
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      call.Function.Name,
				Arguments: string(argsJSON),
			},
		})
	}
	return out
}

func openAIToolCallsToYzma(calls []openAIChatToolCall) []message.ToolCall {
	out := make([]message.ToolCall, 0, len(calls))
	for _, call := range calls {
		typ := strings.TrimSpace(call.Type)
		if typ == "" {
			typ = "function"
		}
		out = append(out, message.ToolCall{
			Type: typ,
			Function: message.ToolFunction{
				Name:      call.Function.Name,
				Arguments: toolArgumentsMap(call.Function.Arguments),
			},
		})
	}
	return out
}

func chatMessagesToYzma(msgs []chatMessage, tools []openAIChatTool, includeTools bool) ([]message.Message, error) {
	out := make([]message.Message, 0, len(msgs)+2)
	var systemParts []string

	if includeTools && len(tools) > 0 {
		defs := openAIToolsToDefinitions(tools)
		systemParts = append(systemParts, fmt.Sprintf(
			"You have access to the following tools:\n\n%s\n\nWhen a tool is needed, respond using the model's tool-call format (for example <tool_call>{\"name\":\"...\",\"arguments\":{...}}</tool_call>).",
			formatToolsForPrompt(defs),
		))
	}

	for _, m := range msgs {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		content := chatMessageContentString(m.Content)
		switch role {
		case "system":
			if content != "" {
				systemParts = append(systemParts, content)
			}
		case "user":
			out = append(out, message.Chat{Role: "user", Content: content})
		case "assistant":
			if len(m.ToolCalls) > 0 {
				out = append(out, message.Tool{
					Role:      "assistant",
					Content:   content,
					ToolCalls: openAIToolCallsToYzma(m.ToolCalls),
				})
			} else {
				out = append(out, message.Chat{Role: "assistant", Content: content})
			}
		case "tool":
			name := strings.TrimSpace(m.Name)
			if name == "" {
				name = strings.TrimSpace(m.ToolCallID)
			}
			out = append(out, message.ToolResponse{
				Role:    "tool",
				Name:    name,
				Content: content,
			})
		default:
			if content != "" {
				out = append(out, message.Chat{Role: role, Content: content})
			}
		}
	}

	if len(systemParts) > 0 {
		out = append([]message.Message{message.Chat{
			Role:    "system",
			Content: strings.Join(systemParts, "\n\n"),
		}}, out...)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("messages 不能为空")
	}
	return out, nil
}
