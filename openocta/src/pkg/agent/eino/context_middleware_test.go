package eino

import (
	"testing"

	"github.com/openocta/openocta/pkg/config"
)

func TestResolveContextMiddlewareSettingsFromConfig(t *testing.T) {
	modeOff := "off"
	contextTokens := 100000
	maxChars := 4096
	keepLast := 6
	reserve := 20000
	cfg := &config.OpenOctaConfig{
		Agents: &config.AgentsConfig{
			Defaults: &config.AgentDefaultsConfig{
				ContextTokens: &contextTokens,
				ContextPruning: &config.AgentContextPruningConfig{
					Mode:               &modeOff,
					KeepLastAssistants: &keepLast,
					SoftTrim: &config.AgentContextPruningSoftTrimConfig{
						MaxChars: &maxChars,
					},
				},
				Compaction: &config.AgentCompactionConfig{
					ReserveTokensFloor: &reserve,
				},
			},
		},
	}

	got := resolveContextMiddlewareSettings(BuildConfig{Config: cfg, TokenLimit: 80000})
	if !got.enableReduction {
		t.Fatal("expected reduction enabled by default")
	}
	if !got.enableSummarization {
		t.Fatal("expected summarization enabled by default")
	}
	if got.maxLengthForTrunc != maxChars {
		t.Fatalf("max trunc = %d, want %d", got.maxLengthForTrunc, maxChars)
	}
	if got.clearRetention != keepLast {
		t.Fatalf("clear retention = %d, want %d", got.clearRetention, keepLast)
	}
	if got.maxTokensForClear != 36000 {
		t.Fatalf("max clear = %d, want 36000", got.maxTokensForClear)
	}
	if got.summarizeAtTokens != 52000 {
		t.Fatalf("summarize at = %d, want 52000", got.summarizeAtTokens)
	}
}
