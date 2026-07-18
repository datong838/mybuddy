package embeddedmodels

import "testing"

func TestExtractHFRepo(t *testing.T) {
	repo, err := ExtractHFRepo("https://huggingface.co/lmstudio-community/Qwen3.5-9B-GGUF")
	if err != nil {
		t.Fatal(err)
	}
	if repo != "lmstudio-community/Qwen3.5-9B-GGUF" {
		t.Fatalf("repo = %q", repo)
	}
}

func TestMatchQuantFile(t *testing.T) {
	files := []ggufCandidate{
		{name: "Model-Q6_K.gguf", size: 100},
		{name: "Model-Q4_K_M.gguf", size: 80},
	}
	chosen, ok := matchQuantFile(files, "Q4_K_M")
	if !ok || chosen.name != "Model-Q4_K_M.gguf" {
		t.Fatalf("chosen = %+v ok=%v", chosen, ok)
	}
}

func TestIsMainModelGGUF(t *testing.T) {
	if !isMainModelGGUF("Qwen3-0.6B-Q4_K_M.gguf") {
		t.Fatal("expected main gguf")
	}
	if isMainModelGGUF("mmproj-Qwen2.5-VL-3B-Instruct-Q4_K_M.gguf") {
		t.Fatal("expected mmproj to be excluded")
	}
}
