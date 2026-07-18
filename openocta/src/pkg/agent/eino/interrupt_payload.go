package eino

import (
	"encoding/json"

	"github.com/cloudwego/eino/adk"

	"github.com/openocta/openocta/pkg/agent/stream"
)

func interruptPayloadFromInfo(sessionID, runID string, info *adk.InterruptInfo) *stream.InterruptPayload {
	if info == nil {
		return nil
	}
	payload := &stream.InterruptPayload{
		SessionID:  sessionID,
		RunID:      runID,
		StopReason: "interrupt",
	}
	for _, ctx := range flattenInterruptContexts(info.InterruptContexts) {
		ic := stream.InterruptContext{
			ID:          ctx.ID,
			IsRootCause: ctx.IsRootCause,
		}
		if ap := approvalInfoFromAny(ctx.Info); ap != nil {
			ic.ToolName = ap.ToolName
			ic.Arguments = ap.ArgumentsInJSON
			if payload.ToolName == "" && ap.ToolName != "" {
				payload.ToolName = ap.ToolName
				payload.Arguments = ap.ArgumentsInJSON
			}
		}
		payload.Contexts = append(payload.Contexts, ic)
	}
	if payload.ToolName == "" {
		if ap := approvalInfoFromAny(info.Data); ap != nil {
			payload.ToolName = ap.ToolName
			payload.Arguments = ap.ArgumentsInJSON
		}
	}
	return payload
}

func flattenInterruptContexts(contexts []*adk.InterruptCtx) []*adk.InterruptCtx {
	if len(contexts) == 0 {
		return nil
	}
	out := make([]*adk.InterruptCtx, 0, len(contexts))
	var walk func(*adk.InterruptCtx)
	walk = func(c *adk.InterruptCtx) {
		if c == nil {
			return
		}
		out = append(out, c)
		walk(c.Parent)
	}
	for _, c := range contexts {
		walk(c)
	}
	return out
}

func approvalInfoFromAny(v any) *ApprovalInfo {
	switch x := v.(type) {
	case *ApprovalInfo:
		return x
	case ApprovalInfo:
		return &x
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var ap ApprovalInfo
		if json.Unmarshal(b, &ap) != nil {
			return nil
		}
		if ap.ToolName == "" && ap.ArgumentsInJSON == "" {
			return nil
		}
		return &ap
	}
}

func resumeTargetsFromPayload(contexts []stream.InterruptContext, approved bool, reason string) map[string]any {
	targets := make(map[string]any)
	var disapprove *string
	if reason != "" && !approved {
		disapprove = &reason
	}
	result := &ApprovalResult{Approved: approved, DisapproveReason: disapprove}
	for _, ctx := range contexts {
		if ctx.ID == "" {
			continue
		}
		if ctx.IsRootCause || len(contexts) == 1 {
			targets[ctx.ID] = result
		}
	}
	if len(targets) == 0 {
		for _, ctx := range contexts {
			if ctx.ID != "" {
				targets[ctx.ID] = result
			}
		}
	}
	return targets
}
