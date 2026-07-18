package eino

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestAssistantTextFromMessagePreservesNewlines(t *testing.T) {
	t.Parallel()

	msg := &schema.Message{
		Role:    schema.Assistant,
		Content: "# Title\n\nLine one\nLine two",
	}
	got := assistantTextFromMessage(msg)
	want := "# Title\n\nLine one\nLine two"
	if got != want {
		t.Fatalf("assistantTextFromMessage() = %q, want %q", got, want)
	}
}

func TestAssistantTextFromMessageMultiContent(t *testing.T) {
	t.Parallel()

	msg := &schema.Message{
		Role: schema.Assistant,
		MultiContent: []schema.ChatMessagePart{
			{Type: schema.ChatMessagePartTypeText, Text: "part one\n"},
			{Type: schema.ChatMessagePartTypeText, Text: "part two"},
		},
	}
	got := assistantTextFromMessage(msg)
	if got != "part one\n\npart two" {
		t.Fatalf("assistantTextFromMessage() = %q", got)
	}
}
