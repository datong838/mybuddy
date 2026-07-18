// Package tools bridges OpenOcta tools to the agent tool interface.
package tools

import (
	"context"

	"github.com/openocta/openocta/pkg/agent/tool"
)

// Registry holds custom tools for the agent runtime.
type Registry struct {
	tools []tool.Tool
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: nil}
}

// Register adds a tool.
func (r *Registry) Register(t tool.Tool) {
	if t != nil {
		r.tools = append(r.tools, t)
	}
}

// Tools returns all registered tools.
func (r *Registry) Tools() []tool.Tool {
	if r.tools == nil {
		return []tool.Tool{}
	}
	return r.tools
}

// EchoTool is a minimal demo tool.
type EchoTool struct{}

func (EchoTool) Name() string        { return "echo" }
func (EchoTool) Description() string { return "Echo back the input text (demo tool)" }

func (EchoTool) Schema() *tool.JSONSchema {
	return &tool.JSONSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Text to echo back",
			},
		},
		Required: []string{"text"},
	}
}

func (EchoTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	text, _ := params["text"].(string)
	if text == "" {
		return &tool.ToolResult{Success: false, Output: "text is required"}, nil
	}
	return &tool.ToolResult{Success: true, Output: text}, nil
}

// DefaultTools returns the default tool set.
func DefaultTools() []tool.Tool {
	return []tool.Tool{}
}

// DefaultToolsWithInvoker returns tools including gateway invoker when available.
func DefaultToolsWithInvoker(invoker GatewayInvoker) []tool.Tool {
	tools := DefaultTools()
	if invoker != nil {
		tools = append(tools, GatewayTool{Invoker: invoker})
	}
	return tools
}
