// Package outbound provides stub adapters for channels without native send.
package outbound

import (
	"context"
	"fmt"
)

// StubAdapter returns "not implemented" for all sends.
type StubAdapter struct {
	ChannelID string
}

// SendText returns not implemented.
func (s *StubAdapter) SendText(ctx context.Context, c *OutboundContext) (*DeliveryResult, error) {
	_ = ctx
	_ = c
	return nil, fmt.Errorf("send not implemented for channel: %s", s.ChannelID)
}

// SendMedia returns not implemented.
func (s *StubAdapter) SendMedia(ctx context.Context, c *OutboundContext) (*DeliveryResult, error) {
	_ = ctx
	_ = c
	return nil, fmt.Errorf("send not implemented for channel: %s", s.ChannelID)
}
