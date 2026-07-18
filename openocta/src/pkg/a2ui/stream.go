package a2ui

import "strings"

const defaultStreamSurfaceID = "main"

// AssistantTextStream emits v0.9 A2UI messages for streaming assistant markdown/text
// using createSurface, a Text component bound to /content, and updateDataModel deltas.
type AssistantTextStream struct {
	SurfaceID string
	started   bool
	layout    bool
	content   strings.Builder
}

// NewAssistantTextStream returns a stream helper for the given surface id.
func NewAssistantTextStream(surfaceID string) *AssistantTextStream {
	sid := strings.TrimSpace(surfaceID)
	if sid == "" {
		sid = defaultStreamSurfaceID
	}
	return &AssistantTextStream{SurfaceID: sid}
}

// AppendDelta appends text and returns the A2UI messages required to render it.
func (s *AssistantTextStream) AppendDelta(delta string) []*ServerMessage {
	if delta == "" {
		return nil
	}
	s.content.WriteString(delta)
	var out []*ServerMessage
	if !s.started {
		s.started = true
		out = append(out, &ServerMessage{
			Version: Version,
			CreateSurface: &CreateSurface{
				SurfaceID: s.SurfaceID,
				CatalogID: BasicCatalogID,
			},
		})
	}
	if !s.layout {
		s.layout = true
		out = append(out, &ServerMessage{
			Version: Version,
			UpdateComponents: &UpdateComponents{
				SurfaceID: s.SurfaceID,
				Components: []map[string]any{
					{
						"id":        "root",
						"component": "Column",
						"children":  []any{"body"},
					},
					{
						"id":        "body",
						"component": "Text",
						"text":      map[string]any{"path": "/content"},
					},
				},
			},
		})
	}
	out = append(out, &ServerMessage{
		Version: Version,
		UpdateDataModel: &UpdateDataModel{
			SurfaceID: s.SurfaceID,
			Path:      "/content",
			Value:     s.content.String(),
		},
	})
	return out
}

// Started reports whether any A2UI output has been produced.
func (s *AssistantTextStream) Started() bool {
	return s.started
}

// Reset clears accumulated text so the next model turn starts a fresh surface payload.
func (s *AssistantTextStream) Reset() {
	if s == nil {
		return
	}
	s.started = false
	s.layout = false
	s.content.Reset()
}
