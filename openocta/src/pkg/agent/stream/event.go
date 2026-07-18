// Package stream defines Anthropic-compatible SSE events for Gateway/UI consumption.
package stream

import "encoding/json"

const (
	EventMessageStart        = "message_start"
	EventContentBlockStart   = "content_block_start"
	EventContentBlockDelta   = "content_block_delta"
	EventContentBlockStop    = "content_block_stop"
	EventMessageDelta        = "message_delta"
	EventMessageStop         = "message_stop"
	EventPing                = "ping"
	EventAgentStart          = "agent_start"
	EventAgentStop           = "agent_stop"
	EventIterationStart      = "iteration_start"
	EventIterationStop       = "iteration_stop"
	EventToolExecutionStart  = "tool_execution_start"
	EventToolExecutionOutput = "tool_execution_output"
	EventToolExecutionResult = "tool_execution_result"
	EventError               = "error"
	EventA2UI                = "a2ui"
	EventRunInterrupted      = "run_interrupted"
)

// InterruptContext describes one resumable interrupt point (Eino InterruptCtx).
type InterruptContext struct {
	ID          string `json:"id,omitempty"`
	ToolName    string `json:"toolName,omitempty"`
	Arguments   string `json:"arguments,omitempty"`
	IsRootCause bool   `json:"isRootCause,omitempty"`
}

// InterruptPayload is emitted when agent execution pauses for human approval.
type InterruptPayload struct {
	SessionID  string             `json:"session_id,omitempty"`
	RunID      string             `json:"run_id,omitempty"`
	Contexts   []InterruptContext `json:"contexts,omitempty"`
	ToolName   string             `json:"toolName,omitempty"`
	Arguments  string             `json:"arguments,omitempty"`
	StopReason string             `json:"stop_reason,omitempty"`
}

type StreamEvent struct {
	Type         string            `json:"type"`
	Message      *Message          `json:"message,omitempty"`
	Index        *int              `json:"index,omitempty"`
	ContentBlock *ContentBlock     `json:"content_block,omitempty"`
	Delta        *Delta            `json:"delta,omitempty"`
	Usage        *Usage            `json:"usage,omitempty"`
	ToolUseID    string            `json:"tool_use_id,omitempty"`
	Name         string            `json:"name,omitempty"`
	Output       interface{}       `json:"output,omitempty"`
	IsStderr     *bool             `json:"is_stderr,omitempty"`
	IsError      *bool             `json:"is_error,omitempty"`
	SessionID    string            `json:"session_id,omitempty"`
	Iteration    *int              `json:"iteration,omitempty"`
	TotalIter    *int              `json:"total_iterations,omitempty"`
	A2UI         json.RawMessage   `json:"a2ui,omitempty"`
	Interrupt    *InterruptPayload `json:"interrupt,omitempty"`
}

type Message struct {
	ID    string `json:"id,omitempty"`
	Type  string `json:"type,omitempty"`
	Role  string `json:"role,omitempty"`
	Model string `json:"model,omitempty"`
	Usage *Usage `json:"usage,omitempty"`
}

type ContentBlock struct {
	Type     string          `json:"type,omitempty"`
	Text     string          `json:"text,omitempty"`
	Thinking string          `json:"thinking,omitempty"`
	ID       string          `json:"id,omitempty"`
	Name     string          `json:"name,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
}

type Delta struct {
	Type        string          `json:"type,omitempty"`
	Text        string          `json:"text,omitempty"`
	Thinking    string          `json:"thinking,omitempty"`
	PartialJSON json.RawMessage `json:"partial_json,omitempty"`
	StopReason  string          `json:"stop_reason,omitempty"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
}
