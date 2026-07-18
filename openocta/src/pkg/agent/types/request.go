// Package types defines agent runtime request/response types.
package types

import (
	"github.com/cloudwego/eino/schema"

	"github.com/openocta/openocta/pkg/agent/model"
	"github.com/openocta/openocta/pkg/agent/stream"
)

type Request struct {
	Prompt        string
	ContentBlocks []model.ContentBlock
	// SessionMessages, when set, is the full model input history (typically from session transcript).
	SessionMessages []*schema.Message
	// TranscriptPath is the session jsonl used to hydrate SessionMessages when empty at turn execution.
	TranscriptPath string
	// SessionHistoryMaxMessages limits transcript turns loaded into SessionMessages (0 = unlimited).
	SessionHistoryMaxMessages int
	// SessionHistoryRoles filters transcript roles when hydrating history (empty = default roles).
	SessionHistoryRoles []string
	SessionID           string
	RequestID           string
}

type Result struct {
	Output     string
	StopReason string
	Usage      model.Usage
	ToolCalls  []model.ToolCall
}

type Response struct {
	RequestID string
	Result    *Result
}

// StreamEvent is an alias for gateway compatibility.
type StreamEvent = stream.StreamEvent
