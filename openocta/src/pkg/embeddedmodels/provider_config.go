package embeddedmodels

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/openocta/openocta/pkg/config"
	"github.com/openocta/openocta/pkg/paths"
)

const (
	// EmbeddedChatProviderKey is the models.providers key for embedded chat models.
	EmbeddedChatProviderKey = "openocta-embedded-chat"
	// EmbeddedEmbeddingProviderKey is the models.providers key for embedded embedding models.
	EmbeddedEmbeddingProviderKey = "openocta-embedded-embedding"
)

// IsEmbeddedProvider reports whether provider uses the embedded gateway proxy.
func IsEmbeddedProvider(provider string) bool {
	return provider == EmbeddedChatProviderKey || provider == EmbeddedEmbeddingProviderKey
}

// GatewayProxyBaseURL returns the OpenAI-compatible proxy base URL for embedded models.
func GatewayProxyBaseURL(cfg *config.OpenOctaConfig, env func(string) string) string {
	if env == nil {
		env = os.Getenv
	}
	var portFromConfig *int
	if cfg != nil && cfg.Gateway != nil {
		portFromConfig = cfg.Gateway.Port
	}
	gatewayPort := paths.ResolveGatewayPort(portFromConfig, env)
	return fmt.Sprintf("http://127.0.0.1:%d/api/embedded-models/v1", gatewayPort)
}

// MergedProviderConfig builds provider entries for all running embedded models.
// Chat/embedding requests route through the gateway proxy at /api/embedded-models/v1.
func MergedProviderConfig(env func(string) string, gatewayPort int) map[string]interface{} {
	if env == nil {
		env = os.Getenv
	}
	cfg, _ := config.Load(env)
	proxyBase := GatewayProxyBaseURL(cfg, env)
	if gatewayPort > 0 {
		proxyBase = fmt.Sprintf("http://127.0.0.1:%d/api/embedded-models/v1", gatewayPort)
	}

	chatModels := make([]map[string]interface{}, 0)
	embedModels := make([]map[string]interface{}, 0)

	for _, snap := range listRuntimeSnapshots() {
		entry, ok := ResolveCatalogEntry(snap.ModelID)
		name := snap.ModelID
		kind := snap.Kind
		contextWindow := defaultContextSize
		maxOutput := defaultMaxTokens
		if ok {
			name = entry.Name
			kind = entry.Kind
			contextWindow, maxOutput = catalogLimits(entry)
		}
		capabilities := "chat"
		if kind == ModelKindEmbedding {
			capabilities = "embedding"
		}
		modelEntry := map[string]interface{}{
			"id":            snap.ModelID,
			"name":          name,
			"contextWindow": contextWindow,
			"capabilities":  capabilities,
		}
		if kind == ModelKindChat {
			modelEntry["maxTokens"] = maxOutput
			chatModels = append(chatModels, modelEntry)
		} else if kind == ModelKindEmbedding {
			embedModels = append(embedModels, modelEntry)
		}
	}

	out := map[string]interface{}{}
	if len(chatModels) > 0 {
		out[EmbeddedChatProviderKey] = map[string]interface{}{
			"baseUrl":     proxyBase,
			"apiKey":      "local",
			"displayName": "内嵌对话",
			"models":      chatModels,
		}
	}
	if len(embedModels) > 0 {
		out[EmbeddedEmbeddingProviderKey] = map[string]interface{}{
			"baseUrl":     proxyBase,
			"apiKey":      "local",
			"displayName": "内嵌向量",
			"models":      embedModels,
		}
	}
	return out
}

// PersistMergedProviderConfig writes all running embedded models into openocta.json.
func PersistMergedProviderConfig(env func(string) string, gatewayPort int) error {
	if env == nil {
		env = os.Getenv
	}
	patchMap := MergedProviderConfig(env, gatewayPort)
	cfg, err := config.Load(env)
	if err != nil {
		return err
	}
	if cfg.Models == nil {
		cfg.Models = &config.ModelsConfig{}
	}
	if cfg.Models.Providers == nil {
		cfg.Models.Providers = map[string]config.ModelProvider{}
	}
	if _, ok := patchMap[EmbeddedChatProviderKey]; !ok {
		delete(cfg.Models.Providers, EmbeddedChatProviderKey)
	}
	if _, ok := patchMap[EmbeddedEmbeddingProviderKey]; !ok {
		delete(cfg.Models.Providers, EmbeddedEmbeddingProviderKey)
	}
	for providerKey, raw := range patchMap {
		b, err := json.Marshal(raw)
		if err != nil {
			return err
		}
		var prov config.ModelProvider
		if err := json.Unmarshal(b, &prov); err != nil {
			return err
		}
		cfg.Models.Providers[providerKey] = prov
	}
	stateDir := paths.ResolveStateDir(env)
	configPath := paths.ResolveConfigPath(env, stateDir)
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, out, 0600)
}
