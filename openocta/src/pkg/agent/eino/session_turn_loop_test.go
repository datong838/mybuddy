package eino

import (
	"testing"

	"github.com/openocta/openocta/pkg/agent/stream"
)

func TestMessageStopEndsAgentEventStream(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reason string
		want   bool
	}{
		{reason: "tool_use", want: false},
		{reason: "", want: false},
		{reason: "end_turn", want: true},
		{reason: "interrupt", want: true},
		{reason: "preempted", want: true},
		{reason: "aborted", want: true},
	}
	for _, tc := range tests {
		evt := stream.StreamEvent{
			Type:  stream.EventMessageStop,
			Delta: &stream.Delta{StopReason: tc.reason},
		}
		if got := messageStopEndsAgentEventStream(evt); got != tc.want {
			t.Fatalf("reason=%q got=%v want=%v", tc.reason, got, tc.want)
		}
	}
}
