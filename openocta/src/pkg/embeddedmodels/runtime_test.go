package embeddedmodels

import (
	"encoding/json"
	"testing"
)

func TestChatRequestEnableThinking(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "omitted", raw: "", want: true},
		{name: "true bool", raw: "true", want: true},
		{name: "false bool", raw: "false", want: false},
		{name: "medium string", raw: `"medium"`, want: true},
		{name: "off string", raw: `"off"`, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw json.RawMessage
			if tt.raw != "" {
				raw = json.RawMessage(tt.raw)
			}
			if got := chatRequestEnableThinking(raw); got != tt.want {
				t.Fatalf("chatRequestEnableThinking(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestDefaultMaxOutputTokensFromCatalog(t *testing.T) {
	inst := &runtimeInstance{modelID: "qwen3-0.6b"}
	entry, ok := FindCatalogEntry(inst.modelID)
	if !ok {
		t.Fatal("expected qwen3-0.6b in catalog")
	}
	_, want := catalogLimits(entry)
	if got := inst.defaultMaxOutputTokens(); got != want {
		t.Fatalf("defaultMaxOutputTokens() = %d, want %d", got, want)
	}
}

func TestChatRequestWantsStream(t *testing.T) {
	falseVal := false
	trueVal := true
	if chatRequestWantsStream(nil) {
		t.Fatal("expected false when stream omitted")
	}
	if chatRequestWantsStream(&falseVal) {
		t.Fatal("expected false when stream=false")
	}
	if !chatRequestWantsStream(&trueVal) {
		t.Fatal("expected true when stream=true")
	}
}

func TestCatalogLimits(t *testing.T) {
	t.Run("uses catalog contextLength", func(t *testing.T) {
		ctx, max := catalogLimits(CatalogEntry{ContextLength: 131072})
		if ctx != 131072 {
			t.Fatalf("contextWindow = %d, want 131072", ctx)
		}
		if max != 65536 {
			t.Fatalf("maxOutput = %d, want 65536", max)
		}
	})
	t.Run("falls back when missing", func(t *testing.T) {
		ctx, max := catalogLimits(CatalogEntry{})
		if ctx != defaultContextSize {
			t.Fatalf("contextWindow = %d, want %d", ctx, defaultContextSize)
		}
		if max != defaultContextSize/2 {
			t.Fatalf("maxOutput = %d, want %d", max, defaultContextSize/2)
		}
	})
}

func TestProviderConfigMaxTokens(t *testing.T) {
	entry, ok := FindCatalogEntry("qwen3-0.6b")
	if !ok {
		t.Fatal("expected qwen3-0.6b in catalog")
	}
	wantCtx, wantMax := catalogLimits(entry)

	chat := ProviderConfig(18902, "qwen3-0.6b")
	chatProv, ok := chat["openocta-embedded-chat"].(map[string]interface{})
	if !ok {
		t.Fatal("expected openocta-embedded-chat provider")
	}
	chatModels, ok := chatProv["models"].([]map[string]interface{})
	if !ok || len(chatModels) == 0 {
		t.Fatal("expected chat models")
	}
	if got := chatModels[0]["maxTokens"]; got != wantMax {
		t.Fatalf("chat maxTokens = %v, want %d", got, wantMax)
	}
	if got := chatModels[0]["contextWindow"]; got != wantCtx {
		t.Fatalf("chat contextWindow = %v, want %d", got, wantCtx)
	}

	embedEntry, ok := FindCatalogEntry("qwen3-embedding-0.6b")
	if !ok {
		t.Fatal("expected qwen3-embedding-0.6b in catalog")
	}
	wantEmbedCtx, _ := catalogLimits(embedEntry)

	embed := ProviderConfig(18903, "qwen3-embedding-0.6b")
	embedProv, ok := embed["openocta-embedded-embedding"].(map[string]interface{})
	if !ok {
		t.Fatal("expected openocta-embedded-embedding provider")
	}
	embedModels, ok := embedProv["models"].([]map[string]interface{})
	if !ok || len(embedModels) == 0 {
		t.Fatal("expected embedding models")
	}
	if got := embedModels[0]["contextWindow"]; got != wantEmbedCtx {
		t.Fatalf("embedding contextWindow = %v, want %d", got, wantEmbedCtx)
	}
	if _, has := embedModels[0]["maxTokens"]; has {
		t.Fatal("embedding model should not set maxTokens")
	}
}
