package embeddedmodels

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeDownloadDestFlattensNestedDir(t *testing.T) {
	root := t.TempDir()
	nestedDir := filepath.Join(root, "model.gguf")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte("gguf-bytes")
	if err := os.WriteFile(filepath.Join(nestedDir, "model.gguf"), content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := normalizeDownloadDest(nestedDir); err != nil {
		t.Fatalf("normalizeDownloadDest: %v", err)
	}
	info, err := os.Stat(nestedDir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("expected regular file at %s", nestedDir)
	}
	got, err := os.ReadFile(nestedDir)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("unexpected file content: %q", got)
	}
}

func TestResolveCatalogFilePathNestedLayout(t *testing.T) {
	root := t.TempDir()
	nestedDir := filepath.Join(root, "Qwen3-0.6B-Q4_K_M.gguf")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(nestedDir, "Qwen3-0.6B-Q4_K_M.gguf")
	if err := os.WriteFile(filePath, []byte("gguf"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveCatalogFilePath(root, "Qwen3-0.6B-Q4_K_M.gguf")
	if err != nil {
		t.Fatalf("resolveCatalogFilePath: %v", err)
	}
	if got != filePath {
		t.Fatalf("got %q, want %q", got, filePath)
	}
}
