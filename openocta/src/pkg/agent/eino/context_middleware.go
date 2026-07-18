package eino

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/dynamictool/toolsearch"
	"github.com/cloudwego/eino/adk/middlewares/reduction"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"

	"github.com/openocta/openocta/pkg/config"
	"github.com/openocta/openocta/pkg/paths"
)

const (
	defaultTruncMaxChars       = 8 * 1024
	defaultContextTokens       = 128000
	defaultClearTokenRatio     = 0.45
	defaultSummarizeTokenRatio = 0.65
	defaultClearRetention      = 4
	defaultClearAtLeastTokens  = 8000
	minToolSearchTools         = 1
)

type contextMiddlewareSettings struct {
	enableReduction     bool
	enableSummarization bool
	enableToolSearch    bool
	maxLengthForTrunc   int
	maxTokensForClear   int64
	clearRetention      int
	clearAtLeastTokens  int64
	summarizeAtTokens   int
	reductionRootDir    string
	transcriptFilePath  string
}

func resolveContextMiddlewareSettings(cfg BuildConfig) contextMiddlewareSettings {
	env := cfg.Env
	if env == nil {
		env = os.Getenv
	}
	stateDir := paths.ResolveStateDir(env)
	reductionRoot := filepath.Join(stateDir, "agents", "reduction")
	transcriptPath := filepath.Join(stateDir, "agents", "transcripts", "conversation.jsonl")

	contextTokens := cfg.TokenLimit
	if contextTokens <= 0 && cfg.Config != nil && cfg.Config.Agents != nil && cfg.Config.Agents.Defaults != nil {
		if v := cfg.Config.Agents.Defaults.ContextTokens; v != nil && *v > 0 {
			contextTokens = *v
		}
	}
	if contextTokens <= 0 {
		contextTokens = defaultContextTokens
	}

	maxTrunc := defaultTruncMaxChars
	if cfg.Config != nil && cfg.Config.Agents != nil && cfg.Config.Agents.Defaults != nil {
		cp := cfg.Config.Agents.Defaults.ContextPruning
		if cp != nil && cp.SoftTrim != nil && cp.SoftTrim.MaxChars != nil && *cp.SoftTrim.MaxChars > 0 {
			maxTrunc = *cp.SoftTrim.MaxChars
		}
	}

	clearRetention := defaultClearRetention
	if cfg.Config != nil && cfg.Config.Agents != nil && cfg.Config.Agents.Defaults != nil {
		if v := cfg.Config.Agents.Defaults.ContextPruning; v != nil && v.KeepLastAssistants != nil && *v.KeepLastAssistants > 0 {
			clearRetention = *v.KeepLastAssistants
		}
	}

	maxClear := int64(float64(contextTokens) * defaultClearTokenRatio)
	summarizeAt := int(float64(contextTokens) * defaultSummarizeTokenRatio)

	if cfg.Config != nil && cfg.Config.Agents != nil && cfg.Config.Agents.Defaults != nil {
		comp := cfg.Config.Agents.Defaults.Compaction
		if comp != nil && comp.ReserveTokensFloor != nil && *comp.ReserveTokensFloor > 0 {
			floor := int64(*comp.ReserveTokensFloor)
			if maxClear > int64(contextTokens)-floor {
				maxClear = int64(contextTokens) - floor
			}
			if summarizeAt > int(int64(contextTokens)-floor) {
				summarizeAt = int(int64(contextTokens) - floor)
			}
		}
	}

	enableReduction := true
	enableSummarization := compactionEnabled(cfg.Config)
	enableToolSearch := true
	if v := strings.TrimSpace(strings.ToLower(env("OPENOCTA_ADK_REDUCTION"))); v == "0" || v == "false" || v == "off" {
		enableReduction = false
	}
	if v := strings.TrimSpace(strings.ToLower(env("OPENOCTA_ADK_SUMMARIZATION"))); v == "0" || v == "false" || v == "off" {
		enableSummarization = false
	}
	if v := strings.TrimSpace(strings.ToLower(env("OPENOCTA_ADK_TOOLSEARCH"))); v == "0" || v == "false" || v == "off" {
		enableToolSearch = false
	}

	return contextMiddlewareSettings{
		enableReduction:     enableReduction,
		enableSummarization: enableSummarization,
		enableToolSearch:    enableToolSearch,
		maxLengthForTrunc:   maxTrunc,
		maxTokensForClear:   maxClear,
		clearRetention:      clearRetention,
		clearAtLeastTokens:  defaultClearAtLeastTokens,
		summarizeAtTokens:   summarizeAt,
		reductionRootDir:    reductionRoot,
		transcriptFilePath:  transcriptPath,
	}
}

func compactionEnabled(cfg *config.OpenOctaConfig) bool {
	if cfg == nil || cfg.Agents == nil || cfg.Agents.Defaults == nil || cfg.Agents.Defaults.Compaction == nil {
		return true
	}
	mode := strings.TrimSpace(strings.ToLower(ptrStr(cfg.Agents.Defaults.Compaction.Mode)))
	return mode != "off"
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func buildAgentMiddlewares(
	ctx context.Context,
	cfg BuildConfig,
	cm einomodel.ToolCallingChatModel,
	fsBackend filesystem.Backend,
	skillsDir string,
	dynamicTools []einotool.BaseTool,
) ([]adk.ChatModelAgentMiddleware, error) {
	settings := resolveContextMiddlewareSettings(cfg)
	var handlers []adk.ChatModelAgentMiddleware

	if settings.enableToolSearch && len(dynamicTools) >= minToolSearchTools {
		tsMW, err := toolsearch.New(ctx, &toolsearch.Config{DynamicTools: dynamicTools})
		if err != nil {
			return nil, fmt.Errorf("toolsearch middleware: %w", err)
		}
		handlers = append(handlers, tsMW)
	}

	if strings.TrimSpace(skillsDir) != "" {
		initCtx := context.WithoutCancel(ctx)
		skillBackend, sbErr := skill.NewBackendFromFilesystem(initCtx, &skill.BackendFromFilesystemConfig{
			Backend: fsBackend,
			BaseDir: skillsDir,
		})
		if sbErr == nil {
			skillMW, smErr := skill.NewMiddleware(initCtx, &skill.Config{Backend: stableSkillBackend{inner: skillBackend}})
			if smErr == nil {
				handlers = append(handlers, skillMW)
			}
		}
	}

	if settings.enableSummarization && cm != nil {
		sumMW, err := summarization.New(ctx, &summarization.Config{
			Model: cm,
			Trigger: &summarization.TriggerCondition{
				ContextTokens: settings.summarizeAtTokens,
			},
			TranscriptFilePath: settings.transcriptFilePath,
		})
		if err != nil {
			return nil, fmt.Errorf("summarization middleware: %w", err)
		}
		handlers = append(handlers, sumMW)
	}

	if settings.enableReduction {
		reductionMW, err := reduction.New(ctx, &reduction.Config{
			Backend:                   fsBackend,
			MaxLengthForTrunc:         settings.maxLengthForTrunc,
			MaxTokensForClear:         settings.maxTokensForClear,
			ClearRetentionSuffixLimit: settings.clearRetention,
			ClearAtLeastTokens:        settings.clearAtLeastTokens,
			ReadFileToolName:          "read_file",
			RootDir:                   settings.reductionRootDir,
		})
		if err != nil {
			return nil, fmt.Errorf("reduction middleware: %w", err)
		}
		handlers = append(handlers, reductionMW)
	}

	return handlers, nil
}
