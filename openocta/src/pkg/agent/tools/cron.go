package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openocta/openocta/pkg/agent/tool"
)

// CronTool exposes cron list/status/add/update/remove/run/runs to the agent via the gateway.
type CronTool struct {
	Invoker GatewayInvoker
}

// Name returns the tool name.
func (CronTool) Name() string {
	return "cron"
}

// Description returns the tool description.
func (CronTool) Description() string {
	return "List, add, update, remove, or run scheduled cron jobs. Actions: status, list, add, update, remove, run, runs, wake."
}

// Schema returns the parameter schema.
func (CronTool) Schema() *tool.JSONSchema {
	return &tool.JSONSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "One of: status, list, add, update, remove, run, runs, wake",
				"enum":        []string{"status", "list", "add", "update", "remove", "run", "runs", "wake"},
			},
			"includeDisabled": map[string]interface{}{"type": "boolean", "description": "For list: include disabled jobs"},
			"jobId":           map[string]interface{}{"type": "string", "description": "Job ID for update/remove/run/runs"},
			"id":              map[string]interface{}{"type": "string", "description": "Alias for jobId"},
			"name":            map[string]interface{}{"type": "string", "description": "Job name (for add)"},
			"schedule":        map[string]interface{}{"type": "object", "description": "Schedule (for add/update)"},
			"payload":         map[string]interface{}{"type": "object", "description": "Payload (for add)"},
			"sessionTarget":   map[string]interface{}{"type": "string", "description": "main or isolated"},
			"wakeMode":        map[string]interface{}{"type": "string", "description": "now or next-heartbeat"},
			"enabled":         map[string]interface{}{"type": "boolean", "description": "Job enabled (for add)"},
			"patch":           map[string]interface{}{"type": "object", "description": "Patch for update"},
			"text":            map[string]interface{}{"type": "string", "description": "Text for wake"},
			"mode":            map[string]interface{}{"type": "string", "description": "For run: due or force"},
			"limit":           map[string]interface{}{"type": "number", "description": "For runs: limit"},
		},
		Required: []string{"action"},
	}
}

// Execute runs the tool.
func (t CronTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	if t.Invoker == nil {
		return &tool.ToolResult{Success: false, Output: "cron: gateway invoker not configured"}, nil
	}
	action, _ := params["action"].(string)
	if action == "" {
		return &tool.ToolResult{Success: false, Output: "action is required"}, nil
	}
	method := "cron." + action
	ok, payload, err := t.Invoker.Invoke(method, params)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: err.Error()}, nil
	}
	if !ok {
		msg := "cron." + action + " failed"
		if payload != nil {
			if m, ok := payload.(map[string]interface{}); ok && m["message"] != nil {
				msg = fmt.Sprint(m["message"])
			}
		}
		return &tool.ToolResult{Success: false, Output: msg}, nil
	}
	out, _ := json.Marshal(payload)
	return &tool.ToolResult{Success: true, Output: string(out)}, nil
}
