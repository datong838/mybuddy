package eino

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/openocta/openocta/pkg/paths"
)

// FileCheckPointStore persists Eino checkpoints under ~/.openocta/checkpoints/.
type FileCheckPointStore struct {
	dir string
	mu  sync.RWMutex
}

func NewFileCheckPointStore(env func(string) string) (*FileCheckPointStore, error) {
	if env == nil {
		env = os.Getenv
	}
	stateDir := paths.ResolveStateDir(env)
	dir := filepath.Join(stateDir, "checkpoints")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &FileCheckPointStore{dir: dir}, nil
}

func (s *FileCheckPointStore) path(id string) string {
	safe := sanitizeCheckpointID(id)
	return filepath.Join(s.dir, safe+".cp")
}

func (s *FileCheckPointStore) Get(_ context.Context, id string) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, true, nil
}

func (s *FileCheckPointStore) Set(_ context.Context, id string, checkpoint []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]byte, len(checkpoint))
	copy(cp, checkpoint)
	tmp := s.path(id) + ".tmp"
	if err := os.WriteFile(tmp, cp, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path(id))
}

func (s *FileCheckPointStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.path(id))
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

func sanitizeCheckpointID(id string) string {
	var b []byte
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
			b = append(b, c)
		default:
			b = append(b, '_')
		}
	}
	if len(b) == 0 {
		return "default"
	}
	return string(b)
}
