package channels

import (
	"sort"
	"strings"
	"sync"
)

// ChatChannelOrder defines the default channel display order.
var ChatChannelOrder = []string{
	"telegram", "whatsapp", "discord", "irc",
	"googlechat", "slack", "signal", "imessage",
}

// Registry holds channel plugins.
type Registry struct {
	mu      sync.RWMutex
	plugins map[ChannelId]ChannelPlugin
}

// NewRegistry creates a new registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[ChannelId]ChannelPlugin),
	}
}

// Register adds a plugin.
func (r *Registry) Register(p ChannelPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := strings.TrimSpace(p.ID())
	if id != "" {
		r.plugins[id] = p
	}
}

// Get returns a plugin by ID.
func (r *Registry) Get(id ChannelId) (ChannelPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, false
	}
	p, ok := r.plugins[id]
	return p, ok
}

// List returns all plugins, sorted by order.
func (r *Registry) List() []ChannelPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []ChannelPlugin
	seen := make(map[string]bool)
	for _, p := range r.plugins {
		id := p.ID()
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		oi := orderIndex(out[i].ID())
		oj := orderIndex(out[j].ID())
		if oi != oj {
			return oi < oj
		}
		return out[i].ID() < out[j].ID()
	})
	return out
}

// GatewayMethods returns all unique gateway methods from registered plugins.
func (r *Registry) GatewayMethods() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]bool)
	for _, p := range r.plugins {
		for _, m := range p.GatewayMethods() {
			if m != "" {
				seen[m] = true
			}
		}
	}
	var out []string
	for m := range seen {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

func orderIndex(id string) int {
	for i, c := range ChatChannelOrder {
		if c == id {
			return i
		}
	}
	return 999
}

// NormalizeChannelId returns trimmed non-empty id or empty.
func NormalizeChannelId(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	return s
}
