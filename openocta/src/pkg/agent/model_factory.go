// Package agent provides agent configuration and model factory creation.
package agent

import (
	"strings"

	"github.com/openocta/openocta/pkg/agent/eino"
	"github.com/openocta/openocta/pkg/config"
)

// ModelFactory is an alias for Eino chat model factory.
type ModelFactory = eino.ChatModelFactory

// ResolveSessionAgentID resolves agent ID from sessionKey.
func ResolveSessionAgentID(sessionKey string) string {
	if sessionKey == "" {
		return "main"
	}
	parts := strings.SplitN(sessionKey, ":", 3)
	if len(parts) >= 3 {
		return strings.TrimSpace(strings.ToLower(parts[1]))
	}
	return "main"
}

func resolveAgentConfig(cfg *config.OpenOctaConfig, agentID string) *config.AgentConfig {
	if cfg == nil || cfg.Agents == nil || len(cfg.Agents.List) == 0 {
		return nil
	}
	for i := range cfg.Agents.List {
		agent := &cfg.Agents.List[i]
		if strings.EqualFold(agent.ID, agentID) {
			return agent
		}
	}
	for i := range cfg.Agents.List {
		agent := &cfg.Agents.List[i]
		if agent.Default != nil && *agent.Default {
			return agent
		}
	}
	if len(cfg.Agents.List) > 0 {
		return &cfg.Agents.List[0]
	}
	return nil
}

func resolveModelFromConfig(modelRef string) (provider string, modelID string) {
	if modelRef == "" {
		return "", ""
	}
	parts := strings.SplitN(strings.TrimSpace(modelRef), "/", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "anthropic", strings.TrimSpace(modelRef)
}

func resolveAgentModelRef(cfg *config.OpenOctaConfig, agentID string) string {
	agentCfg := resolveAgentConfig(cfg, agentID)
	if agentCfg != nil && agentCfg.Model != nil {
		if modelStr, ok := agentCfg.Model.(string); ok && modelStr != "" {
			return strings.TrimSpace(modelStr)
		}
		if modelMap, ok := agentCfg.Model.(map[string]interface{}); ok {
			if primary, ok := modelMap["primary"].(string); ok && primary != "" {
				return strings.TrimSpace(primary)
			}
		}
	}
	if cfg != nil && cfg.Agents != nil && cfg.Agents.Defaults != nil && cfg.Agents.Defaults.Model != nil {
		if cfg.Agents.Defaults.Model.Primary != nil && *cfg.Agents.Defaults.Model.Primary != "" {
			return strings.TrimSpace(*cfg.Agents.Defaults.Model.Primary)
		}
	}
	return ""
}

// ResolveAgentModelRef returns the primary model reference from agent config or defaults.
func ResolveAgentModelRef(cfg *config.OpenOctaConfig, agentID string) string {
	return resolveAgentModelRef(cfg, agentID)
}

// CreateModelFactoryFromConfig creates a ModelFactory from config.
func CreateModelFactoryFromConfig(cfg *config.OpenOctaConfig, agentID string) (ModelFactory, error) {
	modelRef := resolveAgentModelRef(cfg, agentID)
	return eino.CreateModelFactoryFromConfig(cfg, modelRef)
}

// CreateModelFactoryForModelRef creates a ModelFactory for an explicit model reference.
func CreateModelFactoryForModelRef(cfg *config.OpenOctaConfig, modelRef string) (ModelFactory, error) {
	return eino.CreateModelFactoryForModelRef(cfg, modelRef)
}

func modelDefFromProviderCfg(prov config.ModelProvider, resolvedModelID string) *config.ModelDefinition {
	foundModelID := resolvedModelID
	if foundModelID == "" && len(prov.Models) > 0 {
		foundModelID = prov.Models[0].ID
	}
	for i := range prov.Models {
		if prov.Models[i].ID == foundModelID {
			return &prov.Models[i]
		}
	}
	return nil
}

// TokenLimitForSessionHistory returns conversation history trim budget from config.
func TokenLimitForSessionHistory(cfg *config.OpenOctaConfig, agentID string, modelRefOverride string) int {
	if cfg == nil {
		return 0
	}
	ref := strings.TrimSpace(modelRefOverride)
	if ref == "" {
		ref = resolveAgentModelRef(cfg, agentID)
	}
	if ref == "" || cfg.Models == nil || cfg.Models.Providers == nil {
		return 0
	}
	provider, mid := resolveModelFromConfig(ref)
	prov, ok := cfg.Models.Providers[provider]
	if !ok {
		return 0
	}
	resolved := mid
	if resolved == "" && len(prov.Models) > 0 {
		resolved = prov.Models[0].ID
	}
	def := modelDefFromProviderCfg(prov, resolved)
	if def != nil && def.ContextWindow != nil && *def.ContextWindow > 0 {
		return *def.ContextWindow
	}
	return 0
}

// CreateXunfeiImageFactory is implemented in xunfei_eino.go.
