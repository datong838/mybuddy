package eino

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ApprovalInfo is user-facing interrupt metadata for tool approval (Eino interrupt/resume).
type ApprovalInfo struct {
	ToolName        string `json:"toolName"`
	ArgumentsInJSON string `json:"argumentsInJSON"`
}

// ApprovalResult is resume data after the user approves or rejects a tool call.
type ApprovalResult struct {
	Approved         bool    `json:"approved"`
	DisapproveReason *string `json:"disapproveReason,omitempty"`
}

// approvalMiddleware intercepts sensitive DeepAgent tools (execute) and triggers interrupt/resume.
type approvalMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	toolNames map[string]struct{}
}

func newApprovalMiddleware(toolNames ...string) adk.ChatModelAgentMiddleware {
	set := make(map[string]struct{}, len(toolNames))
	for _, name := range toolNames {
		name = strings.TrimSpace(name)
		if name != "" {
			set[name] = struct{}{}
		}
	}
	return &approvalMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		toolNames:                    set,
	}
}

func (m *approvalMiddleware) needsApproval(toolName string) bool {
	_, ok := m.toolNames[strings.TrimSpace(toolName)]
	return ok
}

func (m *approvalMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	if !m.needsApproval(tCtx.Name) {
		return endpoint, nil
	}
	return func(ctx context.Context, args string, opts ...einotool.Option) (string, error) {
		wasInterrupted, _, storedArgs := einotool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return "", einotool.StatefulInterrupt(ctx, &ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: args,
			}, args)
		}
		isTarget, hasData, data := einotool.GetResumeContext[*ApprovalResult](ctx)
		if isTarget && hasData && data != nil {
			if data.Approved {
				return endpoint(ctx, storedArgs, opts...)
			}
			if data.DisapproveReason != nil && strings.TrimSpace(*data.DisapproveReason) != "" {
				return fmt.Sprintf("tool %q disapproved: %s", tCtx.Name, *data.DisapproveReason), nil
			}
			return fmt.Sprintf("tool %q disapproved", tCtx.Name), nil
		}
		isTarget, _, _ = einotool.GetResumeContext[any](ctx)
		if !isTarget {
			return "", einotool.StatefulInterrupt(ctx, &ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: storedArgs,
			}, storedArgs)
		}
		return endpoint(ctx, storedArgs, opts...)
	}, nil
}

func (m *approvalMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	if !m.needsApproval(tCtx.Name) {
		return endpoint, nil
	}
	return func(ctx context.Context, args string, opts ...einotool.Option) (*schema.StreamReader[string], error) {
		wasInterrupted, _, storedArgs := einotool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return nil, einotool.StatefulInterrupt(ctx, &ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: args,
			}, args)
		}
		isTarget, hasData, data := einotool.GetResumeContext[*ApprovalResult](ctx)
		if isTarget && hasData && data != nil {
			if data.Approved {
				return endpoint(ctx, storedArgs, opts...)
			}
			msg := fmt.Sprintf("tool %q disapproved", tCtx.Name)
			if data.DisapproveReason != nil && strings.TrimSpace(*data.DisapproveReason) != "" {
				msg = fmt.Sprintf("tool %q disapproved: %s", tCtx.Name, *data.DisapproveReason)
			}
			return singleChunkStream(msg), nil
		}
		isTarget, _, _ = einotool.GetResumeContext[any](ctx)
		if !isTarget {
			return nil, einotool.StatefulInterrupt(ctx, &ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: storedArgs,
			}, storedArgs)
		}
		return endpoint(ctx, storedArgs, opts...)
	}, nil
}

func singleChunkStream(text string) *schema.StreamReader[string] {
	return schema.StreamReaderFromArray([]string{text})
}
