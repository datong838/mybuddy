package eino

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/cloudwego/eino/adk"
)

const turnLoopCheckpointPrefix = "turnloop_"

// TurnLoopCheckpointID returns the checkpoint key used by TurnLoop (separate from Runner).
func TurnLoopCheckpointID(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return turnLoopCheckpointPrefix + "default"
	}
	return turnLoopCheckpointPrefix + sanitizeCheckpointID(sessionID)
}

func deleteCheckpoint(store adk.CheckPointStore, id string) {
	if store == nil || strings.TrimSpace(id) == "" {
		return
	}
	if deleter, ok := store.(interface {
		Delete(context.Context, string) error
	}); ok {
		_ = deleter.Delete(context.Background(), id)
	}
}

// ClearPersistedSessionCheckpoints removes Runner and TurnLoop checkpoints for a session.
func ClearPersistedSessionCheckpoints(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	store := SharedCheckPointStore(nil)
	deleteCheckpoint(store, sessionID)
	deleteCheckpoint(store, TurnLoopCheckpointID(sessionID))
}

var (
	checkpointOnce  sync.Once
	checkpointStore adk.CheckPointStore
)

// SharedCheckPointStore returns a process-wide persistent checkpoint store.
func SharedCheckPointStore(env func(string) string) adk.CheckPointStore {
	checkpointOnce.Do(func() {
		if env == nil {
			env = os.Getenv
		}
		store, err := NewFileCheckPointStore(env)
		if err != nil {
			checkpointStore = NewMemoryCheckPointStore()
			return
		}
		checkpointStore = store
	})
	return checkpointStore
}
