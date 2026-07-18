package hooks

import (
	"sync"
)

// Registry holds hook handlers by event key.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewRegistry creates a new hook registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string][]Handler),
	}
}

// Register adds a handler for an event key (e.g. "command:new", "session:start").
func (r *Registry) Register(eventKey string, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.handlers[eventKey] == nil {
		r.handlers[eventKey] = []Handler{}
	}
	r.handlers[eventKey] = append(r.handlers[eventKey], handler)
}

// Trigger invokes all handlers for the event key and any wildcard prefix.
func (r *Registry) Trigger(event *Event) error {
	r.mu.RLock()
	eventKey := string(event.Type) + ":" + event.Action
	eventKeyWild := string(event.Type)
	handlers := make([]Handler, 0)
	if h := r.handlers[eventKey]; len(h) > 0 {
		handlers = append(handlers, h...)
	}
	if eventKey != eventKeyWild && len(r.handlers[eventKeyWild]) > 0 {
		handlers = append(handlers, r.handlers[eventKeyWild]...)
	}
	r.mu.RUnlock()
	for _, h := range handlers {
		if err := h(event); err != nil {
			return err
		}
	}
	return nil
}

// Clear removes all handlers for an event key.
func (r *Registry) Clear(eventKey string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, eventKey)
}

// ClearAll removes all handlers.
func (r *Registry) ClearAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = make(map[string][]Handler)
}

// EventKeys returns all registered event keys.
func (r *Registry) EventKeys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.handlers))
	for k := range r.handlers {
		keys = append(keys, k)
	}
	return keys
}
