package eino

import (
	"context"
	"strings"

	ccb "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

type cozeLoopParser struct {
	inner ccb.CallbackDataParser
}

func newCozeLoopParser() *cozeLoopParser {
	return &cozeLoopParser{inner: ccb.NewDefaultDataParser(true)}
}

func (p *cozeLoopParser) ParseInput(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) map[string]any {
	return p.inner.ParseInput(ctx, info, input)
}

func (p *cozeLoopParser) ParseOutput(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) map[string]any {
	if info != nil && info.Component == adk.ComponentOfAgent {
		if tags := p.parseAgentOutput(ctx, output); len(tags) > 0 {
			return tags
		}
	}
	return p.inner.ParseOutput(ctx, info, output)
}

func (p *cozeLoopParser) ParseStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) map[string]any {
	return p.inner.ParseStreamInput(ctx, info, input)
}

func (p *cozeLoopParser) ParseStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) map[string]any {
	return p.inner.ParseStreamOutput(ctx, info, output)
}

func (p *cozeLoopParser) parseAgentOutput(ctx context.Context, output callbacks.CallbackOutput) map[string]any {
	agentOutput := adk.ConvAgentCallbackOutput(output)
	if agentOutput == nil || agentOutput.Events == nil {
		return nil
	}

	var events []map[string]any
	var assistantTexts []string

	for {
		event, ok := agentOutput.Events.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if evtData := serializeAgentEventForTrace(event); evtData != nil {
			events = append(events, evtData)
		}
		if text := assistantTextFromAgentEvent(event); text != "" {
			assistantTexts = append(assistantTexts, text)
		}
	}

	tags := make(map[string]any, 2)
	if len(events) > 0 {
		tags[string(tracespec.Output)] = events
	}
	if len(assistantTexts) > 0 {
		tags["assistant_text"] = strings.Join(assistantTexts, "\n\n")
	}
	return tags
}

func serializeAgentEventForTrace(event *adk.AgentEvent) map[string]any {
	if event == nil {
		return nil
	}

	result := make(map[string]any)
	if event.AgentName != "" {
		result["agent_name"] = event.AgentName
	}
	if len(event.RunPath) > 0 {
		parts := make([]string, 0, len(event.RunPath))
		for _, step := range event.RunPath {
			parts = append(parts, step.String())
		}
		result["run_path"] = strings.Join(parts, " -> ")
	}
	if event.Output != nil && event.Output.MessageOutput != nil {
		msg, _, err := adk.GetMessage(event)
		if err != nil {
			result["message_error"] = err.Error()
		} else if msg != nil {
			result["message"] = map[string]any{
				"role":    string(msg.Role),
				"content": assistantTextFromMessage(msg),
			}
		}
	}
	if event.Err != nil {
		result["error"] = event.Err.Error()
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func assistantTextFromAgentEvent(event *adk.AgentEvent) string {
	if event == nil || event.Output == nil || event.Output.MessageOutput == nil {
		return ""
	}
	msg, _, err := adk.GetMessage(event)
	if err != nil || msg == nil || msg.Role != schema.Assistant {
		return ""
	}
	return assistantTextFromMessage(msg)
}

func assistantTextFromMessage(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	if text := strings.TrimSpace(msg.Content); text != "" {
		return msg.Content
	}
	var parts []string
	for _, part := range msg.AssistantGenMultiContent {
		if part.Type == schema.ChatMessagePartTypeText && strings.TrimSpace(part.Text) != "" {
			parts = append(parts, part.Text)
		}
	}
	if len(parts) == 0 {
		for _, part := range msg.MultiContent {
			if part.Type == schema.ChatMessagePartTypeText && strings.TrimSpace(part.Text) != "" {
				parts = append(parts, part.Text)
			}
		}
	}
	return strings.Join(parts, "\n")
}
