package eino

import (
	"github.com/openocta/openocta/pkg/a2ui"
	"github.com/openocta/openocta/pkg/agent/stream"
)

func emitA2UIMessages(out chan<- stream.StreamEvent, sessionID string, msgs ...*a2ui.ServerMessage) {
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		raw, err := msg.RawJSON()
		if err != nil || len(raw) == 0 {
			continue
		}
		out <- stream.StreamEvent{
			Type:      stream.EventA2UI,
			SessionID: sessionID,
			A2UI:      raw,
		}
	}
}

func emitAssistantTextDelta(out chan<- stream.StreamEvent, sessionID string, textStream *a2ui.AssistantTextStream, delta string) {
	if textStream == nil || delta == "" {
		return
	}
	emitA2UIMessages(out, sessionID, textStream.AppendDelta(delta)...)
}
