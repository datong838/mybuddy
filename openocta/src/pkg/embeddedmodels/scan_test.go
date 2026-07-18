package embeddedmodels

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanSideloadedModels(t *testing.T) {
	root := t.TempDir()
	env := func(key string) string {
		if key == "OPENOCTA_STATE_DIR" {
			return root
		}
		return os.Getenv(key)
	}

	modelID := "my-local-qwen"
	dir := ModelDir(env, modelID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	const fileName = "My-Local-Qwen-Q4_K_M.gguf"
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte("gguf"), 0644); err != nil {
		t.Fatal(err)
	}

	RefreshSideloadCatalog(env)
	entry, ok := ResolveCatalogEntry(modelID)
	if !ok {
		t.Fatal("expected sideloaded entry")
	}
	if !entry.Sideloaded {
		t.Fatal("expected sideloaded=true")
	}
	if entry.Kind != ModelKindChat {
		t.Fatalf("kind = %q, want chat", entry.Kind)
	}
	if !IsInstalled(env, modelID) {
		t.Fatal("expected installed after sideload scan")
	}
}

func TestScanSideloadedEmbeddingKind(t *testing.T) {
	root := t.TempDir()
	env := func(key string) string {
		if key == "OPENOCTA_STATE_DIR" {
			return root
		}
		return os.Getenv(key)
	}

	modelID := "bge-small-embed"
	dir := ModelDir(env, modelID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bge-small-embed-q8_0.gguf"), []byte("gguf"), 0644); err != nil {
		t.Fatal(err)
	}

	RefreshSideloadCatalog(env)
	entry, ok := ResolveCatalogEntry(modelID)
	if !ok {
		t.Fatal("expected sideloaded entry")
	}
	if entry.Kind != ModelKindEmbedding {
		t.Fatalf("kind = %q, want embedding", entry.Kind)
	}
}

func TestListCatalogIncludesSideloadedOnly(t *testing.T) {
	root := t.TempDir()
	env := func(key string) string {
		if key == "OPENOCTA_STATE_DIR" {
			return root
		}
		return os.Getenv(key)
	}

	modelID := "custom-chat-model"
	dir := ModelDir(env, modelID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "weights.gguf"), []byte("gguf"), 0644); err != nil {
		t.Fatal(err)
	}

	items := ListCatalog(env)
	found := false
	for _, item := range items {
		if item["id"] == modelID {
			found = true
			if sideloaded, _ := item["sideloaded"].(bool); !sideloaded {
				t.Fatal("expected sideloaded flag in catalog response")
			}
			if installed, _ := item["installed"].(bool); !installed {
				t.Fatal("expected installed=true")
			}
		}
	}
	if !found {
		t.Fatal("sideloaded model missing from ListCatalog")
	}
}
