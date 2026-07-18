// Package outbound provides delivery logic for channel messages.
package outbound

import (
	"context"
	"fmt"
)

// OutboundAdapter sends messages to a channel.
type OutboundAdapter interface {
	SendText(ctx context.Context, c *OutboundContext) (*DeliveryResult, error)
	SendMedia(ctx context.Context, c *OutboundContext) (*DeliveryResult, error)
}

// AdapterRegistry maps channel IDs to outbound adapters.
type AdapterRegistry struct {
	adapters map[string]OutboundAdapter
}

// NewAdapterRegistry creates a new registry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]OutboundAdapter),
	}
}

// Register adds an adapter for a channel.
func (r *AdapterRegistry) Register(channelID string, a OutboundAdapter) {
	if channelID != "" && a != nil {
		r.adapters[channelID] = a
	}
}

// Get returns the adapter for a channel.
func (r *AdapterRegistry) Get(channelID string) (OutboundAdapter, bool) {
	a, ok := r.adapters[channelID]
	return a, ok
}

// DeliverText sends text to the given channel.
func (r *AdapterRegistry) DeliverText(ctx context.Context, channelID string, c *OutboundContext) (*DeliveryResult, error) {
	a, ok := r.Get(channelID)
	if !ok {
		return nil, fmt.Errorf("outbound not configured for channel: %s", channelID)
	}
	return a.SendText(ctx, c)
}

// DeliverMedia sends media to the given channel.
func (r *AdapterRegistry) DeliverMedia(ctx context.Context, channelID string, c *OutboundContext) (*DeliveryResult, error) {
	a, ok := r.Get(channelID)
	if !ok {
		return nil, fmt.Errorf("outbound not configured for channel: %s", channelID)
	}
	return a.SendMedia(ctx, c)
}
