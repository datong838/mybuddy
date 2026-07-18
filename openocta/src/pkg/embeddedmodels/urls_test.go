package embeddedmodels

import "testing"

func TestRewriteHFMirror(t *testing.T) {
	in := "https://huggingface.co/Qwen/Qwen3-Embedding-0.6B-GGUF/resolve/main/Qwen3-Embedding-0.6B-Q8_0.gguf"
	want := "https://hf-mirror.com/Qwen/Qwen3-Embedding-0.6B-GGUF/resolve/main/Qwen3-Embedding-0.6B-Q8_0.gguf"
	if got := rewriteHFMirror(in, defaultHFMirror); got != want {
		t.Fatalf("rewriteHFMirror() = %q, want %q", got, want)
	}
}

func TestDownloadURLsMirrorFirst(t *testing.T) {
	in := "https://huggingface.co/lmstudio-community/Qwen3-0.6B-GGUF/resolve/main/Qwen3-0.6B-Q4_K_M.gguf"
	urls := DownloadURLs(in, nil)
	if len(urls) != 1 {
		t.Fatalf("expected 1 URL, got %d: %v", len(urls), urls)
	}
	if !stringsHasPrefix(urls[0], "https://hf-mirror.com/") {
		t.Fatalf("expected mirror URL, got %q", urls[0])
	}
}

func TestDownloadURLsMirrorOff(t *testing.T) {
	in := "https://huggingface.co/Qwen/Qwen3-0.6B-GGUF/resolve/main/Qwen3-0.6B-Q8_0.gguf"
	env := func(k string) string {
		if k == "OPENOCTA_HF_MIRROR" {
			return "off"
		}
		return ""
	}
	urls := DownloadURLs(in, env)
	if len(urls) != 1 || urls[0] != in {
		t.Fatalf("expected only catalog URL, got %v", urls)
	}
}

func TestApplyGitHubProxyDefault(t *testing.T) {
	in := "https://github.com/ggml-org/llama.cpp/releases/download/b9934/llama-b9934-bin-ubuntu-x64.tar.gz"
	want := "https://gh-proxy.com/https://github.com/ggml-org/llama.cpp/releases/download/b9934/llama-b9934-bin-ubuntu-x64.tar.gz"
	if got := ApplyGitHubProxy(in, nil); got != want {
		t.Fatalf("ApplyGitHubProxy() = %q, want %q", got, want)
	}
}

func TestGitHubDownloadURLsDefaultFallbacks(t *testing.T) {
	in := "https://github.com/ggml-org/llama.cpp/releases/download/b9934/llama-b9934-bin-ubuntu-x64.tar.gz"
	urls := GitHubDownloadURLs(in, nil)
	if len(urls) != 1 {
		t.Fatalf("expected 1 proxy URL, got %d: %v", len(urls), urls)
	}
	if urls[0] != "https://gh-proxy.com/"+in {
		t.Fatalf("unexpected URL: %q", urls[0])
	}
}

func TestGitHubDownloadURLsOff(t *testing.T) {
	in := "https://github.com/ggml-org/llama.cpp/releases/download/b9934/llama-b9934-bin-ubuntu-x64.tar.gz"
	env := func(k string) string {
		if k == "OPENOCTA_GITHUB_PROXY" {
			return "off"
		}
		return ""
	}
	urls := GitHubDownloadURLs(in, env)
	if len(urls) != 1 || urls[0] != in {
		t.Fatalf("expected only direct URL, got %v", urls)
	}
}

func TestGitHubDownloadURLsCustomList(t *testing.T) {
	in := "https://github.com/hybridgroup/llama-cpp-builder/releases/download/b9934/llama-b9934-bin-ubuntu-cuda-13-x64.tar.gz"
	env := func(k string) string {
		if k == "OPENOCTA_GITHUB_PROXY" {
			return "https://mirror.example.com"
		}
		return ""
	}
	urls := GitHubDownloadURLs(in, env)
	if len(urls) != 1 {
		t.Fatalf("expected 1 custom proxy, got %v", urls)
	}
	if urls[0] != "https://mirror.example.com/"+in {
		t.Fatalf("unexpected URL: %q", urls[0])
	}
}

func TestApplyGitHubProxyOff(t *testing.T) {
	in := "https://github.com/ggml-org/llama.cpp/releases/download/b9934/llama-b9934-bin-ubuntu-x64.tar.gz"
	env := func(k string) string {
		if k == "OPENOCTA_GITHUB_PROXY" {
			return "off"
		}
		return ""
	}
	if got := ApplyGitHubProxy(in, env); got != in {
		t.Fatalf("expected original URL, got %q", got)
	}
}

func TestApplyGitHubProxyCustom(t *testing.T) {
	in := "https://github.com/hybridgroup/llama-cpp-builder/releases/download/b9934/llama-b9934-bin-ubuntu-cuda-13-x64.tar.gz"
	env := func(k string) string {
		if k == "OPENOCTA_GITHUB_PROXY" {
			return "https://mirror.ghproxy.com"
		}
		return ""
	}
	want := "https://mirror.ghproxy.com/" + in
	if got := ApplyGitHubProxy(in, env); got != want {
		t.Fatalf("ApplyGitHubProxy() = %q, want %q", got, want)
	}
}

func TestApplyGitHubProxySkipsNonGitHub(t *testing.T) {
	in := "https://hybridgroup.github.io/llama-cpp-builder/version.json"
	if got := ApplyGitHubProxy(in, nil); got != in {
		t.Fatalf("expected non-GitHub URL unchanged, got %q", got)
	}
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
