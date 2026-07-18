package eino

import (
	"context"
	"sync"
)

type memoryCheckPointStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMemoryCheckPointStore() *memoryCheckPointStore {
	return &memoryCheckPointStore{data: make(map[string][]byte)}
}

func (s *memoryCheckPointStore) Get(_ context.Context, id string) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.data[id]
	if !ok {
		return nil, false, nil
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp, true, nil
}

func (s *memoryCheckPointStore) Set(_ context.Context, id string, checkpoint []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]byte, len(checkpoint))
	copy(cp, checkpoint)
	s.data[id] = cp
	return nil
}

func (s *memoryCheckPointStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

func (s *memoryCheckPointStore) ClearSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, sessionID)
}
