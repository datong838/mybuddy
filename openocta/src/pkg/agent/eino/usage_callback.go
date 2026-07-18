package eino

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	cbutils "github.com/cloudwego/eino/utils/callbacks"

	"github.com/openocta/openocta/pkg/logging"
	"github.com/openocta/openocta/pkg/session"
)

var usageCallbackOnce sync.Once

// SetupUsageCallback registers a global Eino callback that records token usage and
// tool calls into session transcript JSONL (token_usage / tool_call lines).
// Safe to call multiple times; only the first registration takes effect.
func SetupUsageCallback() {
	usageCallbackOnce.Do(func() {
		handler := newUsageCallbackHandler()
		callbacks.AppendGlobalHandlers(handler)
	})
}

func newUsageCallbackHandler() callbacks.Handler {
	recordModel := func(ctx context.Context, info *callbacks.RunInfo, usage *model.TokenUsage, cfg *model.Config) {
		if usage == nil || !usageTrackingEnabled(ctx) {
			return
		}
		modelName := modelNameFromConfig(cfg)
		if modelName == "" && info != nil {
			modelName = strings.TrimSpace(info.Name)
		}
		recordTokenUsage(ctx, usage, modelName, "")
	}

	return cbutils.NewHandlerHelper().
		ChatModel(&cbutils.ModelCallbackHandler{
			OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
				if output == nil {
					return ctx
				}
				recordModel(ctx, info, output.TokenUsage, output.Config)
				return ctx
			},
			OnEndWithStreamOutput: func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[*model.CallbackOutput]) context.Context {
				if output == nil {
					return ctx
				}
				defer output.Close()
				var last *model.CallbackOutput
				for {
					chunk, err := output.Recv()
					if err != nil {
						break
					}
					if chunk != nil {
						last = chunk
					}
				}
				if last != nil {
					recordModel(ctx, info, last.TokenUsage, last.Config)
				}
				return ctx
			},
		}).
		AgenticModel(&cbutils.AgenticModelCallbackHandler{
			OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.AgenticCallbackOutput) context.Context {
				if output == nil {
					return ctx
				}
				modelName := ""
				if output.Config != nil {
					modelName = strings.TrimSpace(output.Config.Model)
				}
				if modelName == "" && info != nil {
					modelName = strings.TrimSpace(info.Name)
				}
				recordTokenUsage(ctx, output.TokenUsage, modelName, "")
				return ctx
			},
			OnEndWithStreamOutput: func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[*model.AgenticCallbackOutput]) context.Context {
				if output == nil {
					return ctx
				}
				defer output.Close()
				var last *model.AgenticCallbackOutput
				for {
					chunk, err := output.Recv()
					if err != nil {
						break
					}
					if chunk != nil {
						last = chunk
					}
				}
				if last != nil {
					modelName := ""
					if last.Config != nil {
						modelName = strings.TrimSpace(last.Config.Model)
					}
					if modelName == "" && info != nil {
						modelName = strings.TrimSpace(info.Name)
					}
					recordTokenUsage(ctx, last.TokenUsage, modelName, "")
				}
				return ctx
			},
		}).
		Tool(&cbutils.ToolCallbackHandler{
			OnEnd: func(ctx context.Context, info *callbacks.RunInfo, _ *tool.CallbackOutput) context.Context {
				if !usageTrackingEnabled(ctx) || !recordToolCallsEnabled(ctx) {
					return ctx
				}
				name := toolNameFromRunInfo(info)
				if name == "" {
					return ctx
				}
				recordToolCall(ctx, name)
				return ctx
			},
		}).
		Handler()
}

func usageTrackingEnabled(ctx context.Context) bool {
	values := adk.GetSessionValues(ctx)
	if len(values) == 0 {
		return false
	}
	enabled, _ := values[usageKeyTokenTracking].(bool)
	return enabled && strings.TrimSpace(sessionIDFromValues(values)) != ""
}

func recordToolCallsEnabled(ctx context.Context) bool {
	values := adk.GetSessionValues(ctx)
	if len(values) == 0 {
		return false
	}
	enabled, _ := values[usageKeyRecordToolCall].(bool)
	return enabled
}

func sessionIDFromValues(values map[string]any) string {
	if values == nil {
		return ""
	}
	sid, _ := values[usageKeySessionID].(string)
	return strings.TrimSpace(sid)
}

func agentIDFromValues(values map[string]any) string {
	if values == nil {
		return session.DefaultAgentID
	}
	aid, _ := values[usageKeyAgentID].(string)
	aid = strings.TrimSpace(aid)
	if aid == "" {
		return session.DefaultAgentID
	}
	return aid
}

func requestIDFromValues(values map[string]any) string {
	if values == nil {
		return ""
	}
	rid, _ := values[usageKeyRequestID].(string)
	return strings.TrimSpace(rid)
}

func transcriptPathForValues(values map[string]any) string {
	sid := sessionIDFromValues(values)
	if sid == "" {
		return ""
	}
	agentID := agentIDFromValues(values)
	dir := session.ResolveAgentSessionsDir(agentID, os.Getenv)
	return filepath.Join(dir, sid+".jsonl")
}

func modelNameFromConfig(cfg *model.Config) string {
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Model)
}

func toolNameFromRunInfo(info *callbacks.RunInfo) string {
	if info == nil {
		return ""
	}
	return strings.TrimSpace(info.Name)
}

func recordTokenUsage(ctx context.Context, usage *model.TokenUsage, modelName, provider string) {
	if usage == nil {
		return
	}
	values := adk.GetSessionValues(ctx)
	path := transcriptPathForValues(values)
	if path == "" {
		return
	}
	input := int64(usage.PromptTokens)
	output := int64(usage.CompletionTokens)
	cacheRead := int64(usage.PromptTokenDetails.CachedTokens)
	total := int64(usage.TotalTokens)
	if total == 0 {
		total = input + output + cacheRead
	}
	if input == 0 && output == 0 && cacheRead == 0 && total == 0 {
		return
	}
	line := logging.TokenUsageSessionLine{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		SessionID:   sessionIDFromValues(values),
		RequestID:   requestIDFromValues(values),
		Model:       modelName,
		Provider:    provider,
		Input:       input,
		Output:      output,
		CacheRead:   cacheRead,
		TotalTokens: total,
	}
	if err := logging.AppendTokenUsageToSession(path, line); err != nil {
		slog.Warn("usage callback: append token_usage failed", "path", path, "error", err)
	}
}

func recordToolCall(ctx context.Context, toolName string) {
	values := adk.GetSessionValues(ctx)
	path := transcriptPathForValues(values)
	if path == "" {
		return
	}
	line := map[string]string{
		"type":      "tool_call",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"sessionId": sessionIDFromValues(values),
		"requestId": requestIDFromValues(values),
		"name":      toolName,
	}
	payload, err := json.Marshal(line)
	if err != nil {
		return
	}
	if err := appendJSONL(path, payload); err != nil {
		slog.Warn("usage callback: append tool_call failed", "path", path, "error", err)
	}
}

func appendJSONL(path string, payload []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, werr := f.Write(append(payload, '\n'))
	_ = f.Close()
	return werr
}
