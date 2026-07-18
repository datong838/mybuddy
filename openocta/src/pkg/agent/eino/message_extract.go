package eino

import (
	"strings"

	"github.com/cloudwego/eino/schema"

	"github.com/openocta/openocta/pkg/agent/stream"
)

// isLeakedModelPayload detects raw model/API payloads that must not be shown to users.
func isLeakedModelPayload(text string) bool {
	return stream.IsLeakedAssistantText(text)
}

func visibleAssistantText(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	text := strings.TrimSpace(msg.Content)
	if isLeakedModelPayload(text) {
		text = ""
	}
	if text == "" {
		for _, part := range msg.AssistantGenMultiContent {
			if part.Type == schema.ChatMessagePartTypeText && strings.TrimSpace(part.Text) != "" {
				text = part.Text
				break
			}
		}
	}
	if isLeakedModelPayload(text) {
		return ""
	}
	return text
}

func assistantThinkingText(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	if t := strings.TrimSpace(msg.ReasoningContent); t != "" {
		return t
	}
	for _, part := range msg.AssistantGenMultiContent {
		if part.Type == schema.ChatMessagePartTypeReasoning && part.Reasoning != nil {
			if t := strings.TrimSpace(part.Reasoning.Text); t != "" {
				return t
			}
		}
	}
	return ""
}

func visibleStreamingText(chunk *schema.Message) string {
	if chunk == nil {
		return ""
	}
	text := chunk.Content
	if text == "" && len(chunk.MultiContent) > 0 {
		for _, p := range chunk.MultiContent {
			if p.Type == schema.ChatMessagePartTypeText && p.Text != "" {
				text += p.Text
			}
		}
	}
	if isLeakedModelPayload(strings.TrimSpace(text)) {
		return ""
	}
	return text
}
