package embeddedmodels

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsInstalledEmptyCatalogFiles(t *testing.T) {
	env := func(key string) string {
		if key == "OPENOCTA_STATE_DIR" {
			return t.TempDir()
		}
		return os.Getenv(key)
	}
	if IsInstalled(env, "llama3.2-1b") {
		t.Fatal("expected not installed when catalog has no files and nothing on disk")
	}
}

func TestIsInstalledUsesManifestFiles(t *testing.T) {
	root := t.TempDir()
	env := func(key string) string {
		if key == "OPENOCTA_STATE_DIR" {
			return root
		}
		return os.Getenv(key)
	}
	modelID := "llama3.2-1b"
	dir := ModelDir(env, modelID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	const fileName = "Llama-3.2-1B-Instruct-Q4_K_M.gguf"
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte("gguf"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := upsertInstalled(env, modelID, []string{fileName}); err != nil {
		t.Fatal(err)
	}
	if !IsInstalled(env, modelID) {
		t.Fatal("expected installed when manifest records downloaded files on disk")
	}
}
