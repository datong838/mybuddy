package eino

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestVisibleStreamingTextPreservesNewlineChunk(t *testing.T) {
	got := visibleStreamingText(&schema.Message{Content: "\n"})
	if got != "\n" {
		t.Fatalf("visibleStreamingText(%q) = %q, want %q", "\n", got, "\n")
	}
}

func TestVisibleStreamingTextEmptyContent(t *testing.T) {
	if got := visibleStreamingText(&schema.Message{Content: ""}); got != "" {
		t.Fatalf("visibleStreamingText empty = %q, want empty", got)
	}
	if got := visibleStreamingText(nil); got != "" {
		t.Fatalf("visibleStreamingText nil = %q, want empty", got)
	}
}
