package handlers

import (
	"errors"
	"strings"
	"sync"

	"github.com/openocta/openocta/pkg/agent/stream"
	"github.com/openocta/openocta/pkg/session"
)

// OpenAPIRunResult carries the synchronous HTTP completion payload.
type OpenAPIRunResult struct {
	Content          string
	PromptTokens     int64
	CompletionTokens int64
	Err              error
}

var openAPIRunWaiters sync.Map // map[string]chan OpenAPIRunResult

func registerOpenAPIRunWaiter(runID string) chan OpenAPIRunResult {
	ch := make(chan OpenAPIRunResult, 1)
	openAPIRunWaiters.Store(runID, ch)
	return ch
}

func unregisterOpenAPIRunWaiter(runID string) {
	openAPIRunWaiters.Delete(runID)
}

func tryCompleteOpenAPIRun(runID, plainText string, streamUsage *stream.Usage, sessionUsage *session.Usage) bool {
	runID = strings.TrimSpace(runID)
	text := strings.TrimSpace(plainText)
	if runID == "" || text == "" {
		return false
	}
	v, ok := openAPIRunWaiters.Load(runID)
	if !ok {
		return false
	}
	ch, ok := v.(chan OpenAPIRunResult)
	if !ok {
		return false
	}
	prompt, completion := tokensFromUsage(streamUsage, sessionUsage)
	result := OpenAPIRunResult{
		Content:          text,
		PromptTokens:     prompt,
		CompletionTokens: completion,
	}
	select {
	case ch <- result:
		openAPIRunWaiters.Delete(runID)
		return true
	default:
		return false
	}
}

func failOpenAPIRun(runID string, err error) {
	runID = strings.TrimSpace(runID)
	if runID == "" || err == nil {
		return
	}
	v, ok := openAPIRunWaiters.Load(runID)
	if !ok {
		return
	}
	ch, ok := v.(chan OpenAPIRunResult)
	if !ok {
		return
	}
	select {
	case ch <- OpenAPIRunResult{Err: err}:
	default:
	}
	openAPIRunWaiters.Delete(runID)
}

func tokensFromUsage(streamUsage *stream.Usage, sessionUsage *session.Usage) (prompt, completion int64) {
	if streamUsage != nil {
		return int64(streamUsage.InputTokens), int64(streamUsage.OutputTokens)
	}
	if sessionUsage != nil {
		prompt = int64(sessionUsage.Input + sessionUsage.CacheRead + sessionUsage.CacheWrite)
		completion = int64(sessionUsage.Output)
		if total := int64(sessionUsage.TotalTokens); total > 0 && prompt+completion == 0 {
			completion = total
		}
	}
	return prompt, completion
}

var errOpenAPIRunNoReply = errors.New("agent run ended without delivering a reply")

func deliverAssistantReplies(
	ctx *Context,
	runID string,
	deliver *DeliverContext,
	plainText string,
	streamUsage *stream.Usage,
	sessionUsage *session.Usage,
) bool {
	text := strings.TrimSpace(plainText)
	if text == "" {
		return false
	}
	deliverAssistantToIM(ctx, deliver, text)
	return tryCompleteOpenAPIRun(runID, text, streamUsage, sessionUsage)
}
