package runtime

import (
	"sync"
	"time"

	"github.com/openocta/openocta/pkg/logging"
)

var poolLog = logging.Sub("runtime-pool")

// Pool reuses agent Runtime instances (and associated MCP connections) per session
// to avoid cold-start cost on every chat message.
type Pool struct {
	mu      sync.Mutex
	entries map[string]*poolEntry
}

type poolEntry struct {
	rt          *Runtime
	onEvict     func()
	fingerprint string
	lastUsed    time.Time
	mu          sync.Mutex
}

var defaultPool = NewPool()

// NewPool creates an empty runtime pool.
func NewPool() *Pool {
	return &Pool{entries: make(map[string]*poolEntry)}
}

// DefaultPool returns the process-wide chat runtime pool.
func DefaultPool() *Pool {
	return defaultPool
}

// EvictSessionRuntime closes and removes the pooled runtime for sessionKey (/new, session delete).
func EvictSessionRuntime(sessionKey string) {
	defaultPool.Evict(sessionKey)
}

// WaitUntilAvailable reports whether sessionKey's pooled runtime can be acquired now.
// It returns true when there is no pooled entry or the entry mutex is not held by an active run.
func WaitUntilAvailable(sessionKey string, timeout time.Duration) bool {
	return defaultPool.WaitUntilAvailable(sessionKey, timeout)
}

// PooledSessionCount returns how many session runtimes are currently cached.
func PooledSessionCount() int {
	return defaultPool.Len()
}

// Acquire returns a pooled Runtime for sessionKey when fingerprint matches; otherwise it
// builds a new one via build. release must be called when the run finishes (unlocks the entry).
func (p *Pool) Acquire(sessionKey, fingerprint string, build func() (*Runtime, func(), error)) (*Runtime, func(), error) {
	if p == nil {
		rt, onEvict, err := build()
		if err != nil {
			return nil, nil, err
		}
		return rt, func() {
			if onEvict != nil {
				onEvict()
			}
			if rt != nil {
				rt.Close()
			}
		}, nil
	}

	release := func(ent *poolEntry, acquiredAt time.Time) func() {
		return func() {
			heldMs := time.Since(acquiredAt).Milliseconds()
			ent.lastUsed = time.Now()
			ent.mu.Unlock()
			if heldMs >= 1000 {
				poolLog.Info("runtime pool released sessionKey=%s heldMs=%d poolSize=%d",
					sessionKey, heldMs, p.Len())
			}
		}
	}

	p.mu.Lock()
	ent, ok := p.entries[sessionKey]
	if ok && ent.fingerprint != fingerprint {
		ent.close()
		delete(p.entries, sessionKey)
		ok = false
	}
	if ok {
		p.mu.Unlock()
		acquiredAt := time.Now()
		ent.mu.Lock()
		if waited := time.Since(acquiredAt); waited >= 50*time.Millisecond {
			poolLog.Info("runtime pool acquire waited sessionKey=%s waitedMs=%d fingerprint=%s poolSize=%d",
				sessionKey, waited.Milliseconds(), fingerprint, p.Len())
		}
		return ent.rt, release(ent, acquiredAt), nil
	}
	p.mu.Unlock()

	rt, onEvict, err := build()
	if err != nil {
		return nil, nil, err
	}

	p.mu.Lock()
	if existing, exists := p.entries[sessionKey]; exists {
		if existing.fingerprint == fingerprint {
			if onEvict != nil {
				onEvict()
			}
			if rt != nil {
				rt.Close()
			}
			p.mu.Unlock()
			acquiredAt := time.Now()
			existing.mu.Lock()
			if waited := time.Since(acquiredAt); waited >= 50*time.Millisecond {
				poolLog.Info("runtime pool acquire waited sessionKey=%s waitedMs=%d fingerprint=%s poolSize=%d",
					sessionKey, waited.Milliseconds(), fingerprint, p.Len())
			}
			return existing.rt, release(existing, acquiredAt), nil
		}
		existing.close()
		delete(p.entries, sessionKey)
	}
	ent = &poolEntry{rt: rt, onEvict: onEvict, fingerprint: fingerprint}
	p.entries[sessionKey] = ent
	p.mu.Unlock()

	acquiredAt := time.Now()
	ent.mu.Lock()
	poolLog.Info("runtime pool created sessionKey=%s fingerprint=%s poolSize=%d",
		sessionKey, fingerprint, p.Len())
	return ent.rt, release(ent, acquiredAt), nil
}

// Len returns the number of pooled session runtimes.
func (p *Pool) Len() int {
	if p == nil {
		return 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.entries)
}

// WaitUntilAvailable waits until sessionKey has no pooled entry or its entry lock is free.
func (p *Pool) WaitUntilAvailable(sessionKey string, timeout time.Duration) bool {
	if p == nil {
		return true
	}
	if timeout <= 0 {
		timeout = time.Second
	}
	start := time.Now()
	deadline := start.Add(timeout)
	hadEntry := false
	for {
		p.mu.Lock()
		ent, ok := p.entries[sessionKey]
		poolSize := len(p.entries)
		p.mu.Unlock()
		if !ok {
			if waited := time.Since(start); hadEntry && waited >= 50*time.Millisecond {
				poolLog.Info("runtime pool wait ok sessionKey=%s waitedMs=%d poolSize=%d reason=entry_removed",
					sessionKey, waited.Milliseconds(), poolSize)
			}
			return true
		}
		hadEntry = true
		if ent.mu.TryLock() {
			ent.mu.Unlock()
			if waited := time.Since(start); waited >= 50*time.Millisecond {
				poolLog.Info("runtime pool wait ok sessionKey=%s waitedMs=%d poolSize=%d reason=entry_unlocked",
					sessionKey, waited.Milliseconds(), poolSize)
			}
			return true
		}
		if time.Now().After(deadline) {
			poolLog.Warn("runtime pool wait timed out sessionKey=%s waitedMs=%d timeoutMs=%d poolSize=%d",
				sessionKey, time.Since(start).Milliseconds(), timeout.Milliseconds(), poolSize)
			return false
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// Evict closes and removes the entry for sessionKey.
func (p *Pool) Evict(sessionKey string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	ent := p.entries[sessionKey]
	delete(p.entries, sessionKey)
	poolSize := len(p.entries)
	p.mu.Unlock()
	if ent != nil {
		asyncClose := !ent.mu.TryLock()
		if !asyncClose {
			ent.mu.Unlock()
		}
		poolLog.Info("runtime pool evict sessionKey=%s asyncClose=%v poolSize=%d",
			sessionKey, asyncClose, poolSize)
		ent.close()
	}
}

// EvictAll closes and removes every pooled runtime (e.g. after knowledge index rebuild).
func (p *Pool) EvictAll() {
	if p == nil {
		return
	}
	p.mu.Lock()
	entries := make([]*poolEntry, 0, len(p.entries))
	for _, ent := range p.entries {
		if ent != nil {
			entries = append(entries, ent)
		}
	}
	p.entries = make(map[string]*poolEntry)
	p.mu.Unlock()
	for _, ent := range entries {
		ent.close()
	}
}

// EvictAllSessionRuntimes evicts all pooled chat runtimes in this process.
func EvictAllSessionRuntimes() {
	defaultPool.EvictAll()
}

func (e *poolEntry) close() {
	if e == nil {
		return
	}
	if !e.mu.TryLock() {
		ent := e
		go func() {
			ent.mu.Lock()
			defer ent.mu.Unlock()
			ent.closeLocked()
		}()
		return
	}
	defer e.mu.Unlock()
	e.closeLocked()
}

func (e *poolEntry) closeLocked() {
	if e.onEvict != nil {
		e.onEvict()
		e.onEvict = nil
	}
	if e.rt != nil {
		e.rt.Close()
		e.rt = nil
	}
}
