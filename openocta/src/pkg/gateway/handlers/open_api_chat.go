package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openocta/openocta/pkg/agent/runtime"
	"github.com/openocta/openocta/pkg/apikeys"
	"github.com/openocta/openocta/pkg/gateway/protocol"
	"github.com/openocta/openocta/pkg/session"
)

// OpenAPICompletionInput is the normalized open completion request.
type OpenAPICompletionInput struct {
	Model          string
	Messages       []map[string]interface{}
	Stream         bool
	ConversationID string
}

// OpenAPICompletionOutput is the assistant reply for HTTP mapping.
type OpenAPICompletionOutput struct {
	Content          string
	Model            string
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
}

// ExecuteOpenAPICompletion runs one turn through the unified chat.send pipeline.
func ExecuteOpenAPICompletion(ctx *Context, httpCtx context.Context, rec *apikeys.Record, in OpenAPICompletionInput) (*OpenAPICompletionOutput, int, string) {
	if ctx == nil || rec == nil {
		return nil, http.StatusServiceUnavailable, "open api not configured"
	}
	if in.Stream {
		return nil, http.StatusNotImplemented, "streaming is not supported yet"
	}
	message, err := extractOpenAPILastUserMessage(in.Messages)
	if err != nil {
		return nil, http.StatusBadRequest, err.Error()
	}
	sessionKey, err := openAPISessionKey(rec, in.ConversationID)
	if err != nil {
		return nil, http.StatusBadRequest, err.Error()
	}
	sessionKey = strings.TrimSpace(strings.ToLower(sessionKey))

	if strings.Contains(sessionKey, ":employee:") {
		if ctx.InvokeMethod != nil {
			_, _, _ = ctx.InvokeMethod("sessions.ensure", map[string]interface{}{"key": sessionKey})
		}
	}

	timeout := openAPIRunTimeout(ctx)
	if deadline, ok := httpCtx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	runID := uuid.New().String()
	resultCh := registerOpenAPIRunWaiter(runID)

	params := ChatSendParams{
		SessionKey:     sessionKey,
		Message:        message,
		IdempotencyKey: runID,
		ModelRef:       strings.TrimSpace(in.Model),
		TimeoutMs:      int(timeout / time.Millisecond),
		Resources:      openAPIChatResources(rec),
	}

	sentRunID, ok, errShape := InvokeChatSendWithError(ctx, params)
	if !ok {
		unregisterOpenAPIRunWaiter(runID)
		return nil, openAPIErrorStatus(errShape), openAPIErrorMessage(errShape, "chat.send failed")
	}
	if sentRunID != "" && sentRunID != runID {
		unregisterOpenAPIRunWaiter(runID)
		runID = sentRunID
		resultCh = registerOpenAPIRunWaiter(runID)
	}
	defer unregisterOpenAPIRunWaiter(runID)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var result OpenAPIRunResult
	var got bool
	select {
	case result = <-resultCh:
		got = true
	case <-httpCtx.Done():
		return nil, http.StatusGatewayTimeout, httpCtx.Err().Error()
	case <-timer.C:
	}

	if !got {
		if waitErr := waitForChatRunCompletion(httpCtx, runID, 5*time.Second); waitErr == nil {
			content, promptTokens, completionTokens, fetchErr := fetchLastAssistantFromSession(ctx, sessionKey)
			if fetchErr == nil && strings.TrimSpace(content) != "" {
				result = OpenAPIRunResult{
					Content:          content,
					PromptTokens:     promptTokens,
					CompletionTokens: completionTokens,
				}
				got = true
			}
		}
	}
	if !got {
		return nil, http.StatusGatewayTimeout, "agent run timed out"
	}

	if result.Err != nil {
		return nil, http.StatusInternalServerError, result.Err.Error()
	}
	if strings.TrimSpace(result.Content) == "" {
		return nil, http.StatusInternalServerError, "agent returned empty response"
	}

	total := result.PromptTokens + result.CompletionTokens
	if ctx.ApiKeyService != nil && total > 0 {
		_ = ctx.ApiKeyService.RecordUsage(rec.ID, total)
	}

	return &OpenAPICompletionOutput{
		Content:          result.Content,
		Model:            strings.TrimSpace(in.Model),
		PromptTokens:     result.PromptTokens,
		CompletionTokens: result.CompletionTokens,
		TotalTokens:      total,
	}, http.StatusOK, ""
}

func openAPIRunTimeout(ctx *Context) time.Duration {
	cfg := loadConfigFromContext(ctx)
	if d := runtime.DefaultAgentRunDuration(os.Getenv, cfg); d > 0 {
		return d
	}
	return 10 * time.Minute
}

func openAPISessionKey(rec *apikeys.Record, conversationID string) (string, error) {
	if rec == nil {
		return "", fmt.Errorf("invalid api key")
	}
	conv := sanitizeOpenAPIConversationID(conversationID)
	if rec.BindingMode == apikeys.BindingModeEmployee {
		empID := strings.TrimSpace(rec.DigitalEmployeeID)
		if empID == "" {
			return "", fmt.Errorf("api key employee binding is not configured")
		}
		return fmt.Sprintf("agent:main:employee:%s:run:open-api:%s:%s", empID, rec.ID, conv), nil
	}
	return fmt.Sprintf("agent:main:open-api:%s:%s", rec.ID, conv), nil
}

func sanitizeOpenAPIConversationID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "default"
	}
	if len(out) > 64 {
		return out[:64]
	}
	return out
}

func openAPIChatResources(rec *apikeys.Record) map[string]interface{} {
	if rec == nil || rec.BindingMode == apikeys.BindingModeEmployee {
		return nil
	}
	return map[string]interface{}{
		"configured": true,
		"skillKeys":  append([]string(nil), rec.SkillKeys...),
		"mcpServers": append([]string(nil), rec.McpServers...),
		"webSearch":  false,
	}
}

func extractOpenAPILastUserMessage(messages []map[string]interface{}) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("messages required")
	}
	for i := len(messages) - 1; i >= 0; i-- {
		role, _ := messages[i]["role"].(string)
		if !strings.EqualFold(strings.TrimSpace(role), "user") {
			continue
		}
		text := extractOpenAPIMessageContent(messages[i])
		if text == "" {
			return "", fmt.Errorf("user message content required")
		}
		return text, nil
	}
	return "", fmt.Errorf("messages must include a user message")
}

func extractOpenAPIMessageContent(msg map[string]interface{}) string {
	if msg == nil {
		return ""
	}
	switch content := msg["content"].(type) {
	case string:
		return strings.TrimSpace(content)
	case []interface{}:
		var parts []string
		for _, item := range content {
			block, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if text, ok := block["text"].(string); ok && strings.TrimSpace(text) != "" {
				parts = append(parts, text)
				continue
			}
			if text, ok := block["content"].(string); ok && strings.TrimSpace(text) != "" {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	}
	return ""
}

func waitForChatRunCompletion(ctx context.Context, runID string, timeout time.Duration) error {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return fmt.Errorf("missing run id")
	}
	deadline := time.Now().Add(timeout)
	appeared := false
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if _, ok := chatAbortControllers.Load(runID); ok {
			appeared = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !appeared {
		return fmt.Errorf("agent run did not start")
	}
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if _, ok := chatAbortControllers.Load(runID); !ok {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("agent run timed out")
}

func fetchLastAssistantFromSession(ctx *Context, sessionKey string) (string, int64, int64, error) {
	params := map[string]interface{}{"sessionKey": sessionKey}
	sessionID, sessionFile, storePath, err := ResolveChatSessionID(params, ctx)
	if err != nil {
		return "", 0, 0, err
	}
	env := func(k string) string { return os.Getenv(k) }
	cfg := loadConfigFromContext(ctx)
	resolveKey := sessionKey
	if resolveKey == "" {
		resolveKey = "main"
	}
	target := resolveGatewaySessionStoreTarget(cfg, resolveKey, env)
	transcriptPath := resolveSessionTranscriptPath(sessionID, storePath, sessionFile, target.agentID, env)
	msgs, err := session.ReadTranscriptMessages(transcriptPath, 0)
	if err != nil {
		for _, alt := range resolveSessionTranscriptCandidates(sessionID, storePath, sessionFile, target.agentID, env) {
			if alt == transcriptPath {
				continue
			}
			if m2, e2 := session.ReadTranscriptMessages(alt, 0); e2 == nil {
				msgs, err = m2, nil
				break
			}
		}
	}
	if err != nil && len(msgs) == 0 {
		return "", 0, 0, err
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if !strings.EqualFold(strings.TrimSpace(m.Role), "assistant") {
			continue
		}
		msgMap := transcriptMessageToClientFormat(m)
		text := extractAssistantTextForIMDelivery(msgMap)
		if text == "" {
			text = extractAssistantTextFromMessage(msgMap)
		}
		var promptTokens, completionTokens int64
		if m.Usage != nil {
			promptTokens = int64(m.Usage.Input + m.Usage.CacheRead + m.Usage.CacheWrite)
			completionTokens = int64(m.Usage.Output)
			if total := int64(m.Usage.TotalTokens); total > 0 && promptTokens+completionTokens == 0 {
				completionTokens = total
			}
		}
		return text, promptTokens, completionTokens, nil
	}
	return "", 0, 0, fmt.Errorf("assistant reply not found")
}

func openAPIErrorStatus(errShape *protocol.ErrorShape) int {
	if errShape == nil {
		return http.StatusInternalServerError
	}
	switch errShape.Code {
	case protocol.ErrCodeInvalidRequest:
		return http.StatusBadRequest
	case protocol.ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case protocol.ErrCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func openAPIErrorMessage(errShape *protocol.ErrorShape, fallback string) string {
	if errShape != nil && strings.TrimSpace(errShape.Message) != "" {
		return strings.TrimSpace(errShape.Message)
	}
	return fallback
}
