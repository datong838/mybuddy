package tools

import "testing"

func TestStripOpenOctaAttachmentsMarker(t *testing.T) {
	input := "summary\n@@OPENOCTA_ATTACHMENTS@@\n[{\"type\":\"file\"}]"
	got := StripOpenOctaAttachmentsMarker(input)
	if got != "summary" {
		t.Fatalf("got %q want summary", got)
	}
	if StripOpenOctaAttachmentsMarker("plain text") != "plain text" {
		t.Fatal("expected plain text unchanged")
	}
}
