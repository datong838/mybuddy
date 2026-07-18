package runtime

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolReusesRuntimeForSameFingerprint(t *testing.T) {
	pool := NewPool()
	var builds atomic.Int32

	rt1, release1, err := pool.Acquire("sess-a", "fp1", func() (*Runtime, func(), error) {
		builds.Add(1)
		return &Runtime{}, func() {}, nil
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	release1()

	rt2, release2, err := pool.Acquire("sess-a", "fp1", func() (*Runtime, func(), error) {
		builds.Add(1)
		return &Runtime{}, func() {}, nil
	})
	if err != nil {
		t.Fatalf("re-acquire: %v", err)
	}
	defer release2()

	if builds.Load() != 1 {
		t.Fatalf("expected one build, got %d", builds.Load())
	}
	if rt1 != rt2 {
		t.Fatalf("expected same runtime pointer")
	}
}

func TestPoolEvictForcesRebuild(t *testing.T) {
	pool := NewPool()
	var builds atomic.Int32

	_, release, err := pool.Acquire("sess-b", "fp1", func() (*Runtime, func(), error) {
		builds.Add(1)
		return &Runtime{}, func() {}, nil
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	release()

	pool.Evict("sess-b")

	_, release2, err := pool.Acquire("sess-b", "fp1", func() (*Runtime, func(), error) {
		builds.Add(1)
		return &Runtime{}, func() {}, nil
	})
	if err != nil {
		t.Fatalf("re-acquire: %v", err)
	}
	release2()

	if builds.Load() != 2 {
		t.Fatalf("expected two builds after evict, got %d", builds.Load())
	}
}

func TestWaitUntilAvailableWhenNoEntry(t *testing.T) {
	pool := NewPool()
	if !pool.WaitUntilAvailable("missing", time.Millisecond) {
		t.Fatal("expected available when entry is absent")
	}
}

func TestWaitUntilAvailableReturnsAfterRelease(t *testing.T) {
	pool := NewPool()
	_, release, err := pool.Acquire("sess-wait", "fp1", func() (*Runtime, func(), error) {
		return &Runtime{}, func() {}, nil
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if !pool.WaitUntilAvailable("sess-wait", time.Second) {
			t.Error("expected pool to become available after release")
		}
	}()

	time.Sleep(50 * time.Millisecond)
	release()
	<-done
}

func TestWaitUntilAvailableFalseWhileHeld(t *testing.T) {
	pool := NewPool()
	_, release, err := pool.Acquire("sess-busy", "fp1", func() (*Runtime, func(), error) {
		return &Runtime{}, func() {}, nil
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer release()

	if pool.WaitUntilAvailable("sess-busy", 50*time.Millisecond) {
		t.Fatal("expected wait to time out while entry is held")
	}
}
