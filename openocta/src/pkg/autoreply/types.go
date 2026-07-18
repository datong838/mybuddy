// Package autoreply provides auto-reply queue and dispatch (Phase 6d skeleton).
package autoreply

// ReplyPayload is a single reply payload to be delivered.
type ReplyPayload struct {
	Text       string   `json:"text,omitempty"`
	MediaURL   string   `json:"mediaUrl,omitempty"`
	MediaURLs  []string `json:"mediaUrls,omitempty"`
	ResponseID string   `json:"responseId,omitempty"`
}

// ReplyDispatchKind is the type of reply.
type ReplyDispatchKind string

const (
	ReplyKindTool  ReplyDispatchKind = "tool"
	ReplyKindBlock ReplyDispatchKind = "block"
	ReplyKindFinal ReplyDispatchKind = "final"
)

// DispatcherOptions configures the reply dispatcher.
type DispatcherOptions struct {
	Deliver func(payload ReplyPayload, kind ReplyDispatchKind) error
	OnIdle  func()
	OnError func(err error, kind ReplyDispatchKind)
}
