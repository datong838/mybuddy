package embeddedmodels

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindModelsToRestore(t *testing.T) {
	root := t.TempDir()
	env := func(k string) string {
		if k == "OPENOCTA_STATE_DIR" {
			return root
		}
		return ""
	}
	modelDir := filepath.Join(root, "embedded-models", "qwen3-0.6b")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, "Qwen3-0.6B-Q4_K_M.gguf"), []byte("gguf"), 0644); err != nil {
		t.Fatal(err)
	}

	manifestMu.Lock()
	m, err := loadManifest(env)
	if err != nil {
		manifestMu.Unlock()
		t.Fatal(err)
	}
	m.Models["qwen3-0.6b"] = InstalledState{
		ID:        "qwen3-0.6b",
		Running:   true,
		Port:      18902,
		StartedAt: time.Now().UTC(),
		Files:     []string{"Qwen3-0.6B-Q4_K_M.gguf"},
	}
	m.Models["llama3.2-1b"] = InstalledState{
		ID:      "llama3.2-1b",
		Running: true,
		Port:    18903,
	}
	if err := saveManifest(env, m); err != nil {
		manifestMu.Unlock()
		t.Fatal(err)
	}
	manifestMu.Unlock()

	got := findModelsToRestore(env)
	if len(got) != 2 {
		t.Fatalf("expected 2 models to restore, got %d", len(got))
	}
	ids := map[string]int{}
	for _, item := range got {
		ids[item.modelID] = item.port
	}
	if ids["qwen3-0.6b"] != 18902 {
		t.Fatalf("qwen port=%d", ids["qwen3-0.6b"])
	}
	if ids["llama3.2-1b"] != 18903 {
		t.Fatalf("llama port=%d", ids["llama3.2-1b"])
	}
}

func TestListenPortPrefersSavedPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	preferred := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	port, gotLn, err := listenPortExcluding(preferred, 18902, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer gotLn.Close()
	if port != preferred {
		t.Fatalf("got port %d, want %d", port, preferred)
	}
}

func TestListenPortExcludesUsedPorts(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	blocked := ln.Addr().(*net.TCPAddr).Port
	defer ln.Close()

	port, gotLn, err := listenPortExcluding(blocked, blocked, map[int]struct{}{blocked: {}})
	if err != nil {
		t.Fatal(err)
	}
	defer gotLn.Close()
	if port == blocked {
		t.Fatalf("expected different port, got blocked port %d", port)
	}
}
