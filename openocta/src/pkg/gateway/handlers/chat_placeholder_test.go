package handlers

import "testing"

func TestShouldSkipAssistantTextBlock(t *testing.T) {
	t.Parallel()

	toolBlocks := []map[string]interface{}{
		{"type": "toolCall", "id": "bash:1", "name": "bash"},
	}

	if !shouldSkipAssistantTextBlock(".", toolBlocks) {
		t.Fatal("expected placeholder text with tool calls to be skipped")
	}
	if shouldSkipAssistantTextBlock("checking chrome", toolBlocks) {
		t.Fatal("expected real text with tool calls to be kept")
	}
	if !shouldSkipAssistantTextBlock(" . ", toolBlocks) {
		t.Fatal("expected trimmed placeholder text to be skipped")
	}
	if shouldSkipAssistantTextBlock(".", nil) {
		t.Fatal("expected placeholder without tool calls to be kept")
	}
}

func TestShouldSuppressAssistantTextForA2UI(t *testing.T) {
	t.Parallel()

	a2uiBlocks := []map[string]interface{}{
		{"type": "a2ui", "a2ui": map[string]interface{}{"createSurface": map[string]interface{}{}}},
	}
	if !shouldSuppressAssistantTextForA2UI("hello", a2uiBlocks, false) {
		t.Fatal("expected text suppressed when content already has a2ui")
	}
	if !shouldSuppressAssistantTextForA2UI("hello", nil, true) {
		t.Fatal("expected text suppressed when turn has a2ui")
	}
	if shouldSuppressAssistantTextForA2UI("hello", nil, false) {
		t.Fatal("expected plain text kept when no a2ui")
	}
}

func TestNormalizeAssistantContentForA2UI(t *testing.T) {
	t.Parallel()

	withBoth := []map[string]interface{}{
		{"type": "text", "text": "hello"},
		{"type": "a2ui", "a2ui": map[string]interface{}{"createSurface": map[string]interface{}{"surfaceId": "main"}}},
	}
	out := normalizeAssistantContentForA2UI(withBoth, true, nil)
	if combinedAssistantText(out) != "" {
		t.Fatalf("expected text stripped when a2ui present, got %q", combinedAssistantText(out))
	}
	if !assistantContentHasA2UI(out) {
		t.Fatal("expected a2ui block preserved")
	}

	textOnly := []map[string]interface{}{
		{"type": "text", "text": "# Title\n\nLine one\nLine two"},
	}
	out = normalizeAssistantContentForA2UI(textOnly, true, nil)
	if got := combinedAssistantText(out); got != "# Title\n\nLine one\nLine two" {
		t.Fatalf("expected markdown text preserved on final turn, got %q", got)
	}
	if assistantContentHasA2UI(out) {
		t.Fatal("expected no auto a2ui conversion for plain text")
	}
}

func TestExtractAssistantTextForIMDeliverySkipsPlaceholder(t *testing.T) {
	t.Parallel()

	msg := map[string]interface{}{
		"role": "assistant",
		"content": []map[string]interface{}{
			{"type": "text", "text": "."},
			{"type": "toolCall", "id": "bash:1", "name": "bash"},
		},
	}
	if got := extractAssistantTextForIMDelivery(msg); got != "" {
		t.Fatalf("extractAssistantTextForIMDelivery() = %q, want empty", got)
	}
}

func TestImPlainFromTurnUsesTextBufWhenA2UIStripsText(t *testing.T) {
	t.Parallel()

	a2uiOnly := map[string]interface{}{
		"role": "assistant",
		"content": []map[string]interface{}{
			{"type": "a2ui", "a2ui": map[string]interface{}{"updateDataModel": map[string]interface{}{"path": "/content", "value": "hello"}}},
		},
	}
	if got := imPlainFromTurn(a2uiOnly, "hello from model", "end_turn"); got != "hello from model" {
		t.Fatalf("imPlainFromTurn() = %q, want hello from model", got)
	}
	if got := imPlainFromTurn(a2uiOnly, "pre-tool", "tool_use"); got != "" {
		t.Fatalf("imPlainFromTurn() on tool_use = %q, want empty", got)
	}
	if got := imPlainFromTurn(a2uiOnly, "final answer", "end_turn"); got != "final answer" {
		t.Fatalf("imPlainFromTurn() on end_turn = %q, want final answer", got)
	}
}
