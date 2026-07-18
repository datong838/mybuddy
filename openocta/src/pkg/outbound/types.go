// Package outbound provides types and infrastructure for channel message delivery.
// Mirrors src/infra/outbound/.
package outbound

// DeliveryResult is the result of sending a message to a channel.
type DeliveryResult struct {
	Channel        string `json:"channel"`
	MessageID      string `json:"messageId"`
	ChatID         string `json:"chatId,omitempty"`
	ChannelID      string `json:"channelId,omitempty"`
	RoomID         string `json:"roomId,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
	Timestamp      int64  `json:"timestamp,omitempty"`
	ToJID          string `json:"toJid,omitempty"`
}

// OutboundContext is the context for sending a message.
type OutboundContext struct {
	To        string
	Text      string
	MediaURL  string
	AccountID string
	ReplyToID string
	ThreadID  string
}
