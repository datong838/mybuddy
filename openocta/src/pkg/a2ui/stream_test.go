package a2ui

import "testing"

func TestAssistantTextStreamAppendDelta(t *testing.T) {
	stream := NewAssistantTextStream("main")
	msgs := stream.AppendDelta("hello")
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].CreateSurface == nil || msgs[0].CreateSurface.SurfaceID != "main" {
		t.Fatalf("expected createSurface for main, got %#v", msgs[0])
	}
	if msgs[1].UpdateComponents == nil {
		t.Fatalf("expected updateComponents layout")
	}
	if msgs[2].UpdateDataModel == nil || msgs[2].UpdateDataModel.Value != "hello" {
		t.Fatalf("expected updateDataModel value hello, got %#v", msgs[2].UpdateDataModel)
	}

	msgs = stream.AppendDelta(" world")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 follow-up message, got %d", len(msgs))
	}
	if msgs[0].UpdateDataModel == nil || msgs[0].UpdateDataModel.Value != "hello world" {
		t.Fatalf("expected accumulated text, got %#v", msgs[0].UpdateDataModel)
	}
}

func TestAssistantTextStreamAppendDeltaPreservesNewlineChunk(t *testing.T) {
	stream := NewAssistantTextStream("main")
	msgs := stream.AppendDelta("\n")
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages for newline delta, got %d", len(msgs))
	}
	if msgs[2].UpdateDataModel == nil || msgs[2].UpdateDataModel.Value != "\n" {
		t.Fatalf("expected newline value, got %#v", msgs[2].UpdateDataModel)
	}
}
