package eino

import (
	"strings"

	"github.com/cloudwego/eino/adk"

	"github.com/openocta/openocta/pkg/agent/types"
)

// Session value keys for usage callback (align with legacy token middleware).
const (
	usageKeySessionID      = "session_id"
	usageKeyRequestID      = "request_id"
	usageKeyAgentID        = "agent_id"
	usageKeyTokenTracking  = "token_tracking"
	usageKeyRecordToolCall = "usage_record_tool_calls"
)

func usageRunOptions(req types.Request, agentID string, tokenTracking bool) []adk.AgentRunOption {
	opts := []adk.AgentRunOption{}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return opts
	}
	values := map[string]any{
		usageKeySessionID:     sessionID,
		usageKeyRequestID:     strings.TrimSpace(req.RequestID),
		usageKeyAgentID:       strings.TrimSpace(agentID),
		usageKeyTokenTracking: tokenTracking,
	}
	opts = append(opts, adk.WithSessionValues(values))
	return opts
}
