// Package tool defines the OpenOcta agent tool interface.
package tool

import "context"

type Tool interface {
	Name() string
	Description() string
	Schema() *JSONSchema
	Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}

type OutputRef struct {
	Path      string `json:"path,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

type ToolResult struct {
	Success   bool
	Output    string
	OutputRef *OutputRef
	Data      interface{}
	Error     error
}

type JSONSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required"`
	Enum       []interface{}          `json:"enum,omitempty"`
	Pattern    string                 `json:"pattern,omitempty"`
	Minimum    *float64               `json:"minimum,omitempty"`
	Maximum    *float64               `json:"maximum,omitempty"`
	Items      *JSONSchema            `json:"items,omitempty"`
}
