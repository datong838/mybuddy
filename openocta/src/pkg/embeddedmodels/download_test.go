package embeddedmodels

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPercentTrackerUsesExpectedSizeWhenHTTPTotalUnknown(t *testing.T) {
	var lastDone, lastTotal int64
	tracker := newPercentTracker(10, 80, 1000, func(_ int, done, total int64) {
		lastDone = done
		lastTotal = total
	})
	body := tracker.TrackProgress("test.gguf", 0, -1, io.NopCloser(strings.NewReader(strings.Repeat("x", 600))))
	buf := make([]byte, 600)
	if _, err := body.Read(buf); err != nil && err != io.EOF {
		t.Fatal(err)
	}
	_ = body.Close()
	if lastDone <= 0 {
		t.Fatal("expected bytes done when HTTP total size is unknown")
	}
	if lastTotal != 1000 {
		t.Fatalf("expected total 1000, got %d", lastTotal)
	}
}

func TestCleanupCancelledDownload(t *testing.T) {
	root := t.TempDir()
	env := func(key string) string {
		if key == "OPENOCTA_STATE_DIR" {
			return root
		}
		return os.Getenv(key)
	}
	modelID := "test-cancel-cleanup"
	dir := ModelDir(env, modelID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "partial.gguf"), []byte("gguf"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := upsertInstalled(env, modelID, []string{"partial.gguf"}); err != nil {
		t.Fatal(err)
	}

	cleanupCancelledDownload(env, modelID)

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected model dir removed, stat err=%v", err)
	}
	st, ok := getInstalledState(env, modelID)
	if ok && len(st.Files) > 0 {
		t.Fatalf("expected manifest entry removed, got %+v", st)
	}
}
