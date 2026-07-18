// Package handlers: 统一 chat.send 调用入口，供 feishu、dingding、webhooks、cron、employee 等渠道复用。
package handlers

import (
	"github.com/openocta/openocta/pkg/gateway/protocol"
)

// ChatSendParams 是 chat.send 的统一参数，供各渠道构建后调用。
type ChatSendParams struct {
	SessionKey     string
	Message        string
	SessionID      string
	IdempotencyKey string
	Channel        string
	To             string
	ChatType       string
	MessageID      string
	Header         string
	Metadata       map[string]interface{}
	Thinking       string
	TimeoutMs      int
	Attachments    []interface{}
	ModelRef       string
	Resources      map[string]interface{}
}

// InvokeChatSend 通过 ctx.InvokeMethod 调用 chat.send，返回 runId 与是否成功。
// 供 HooksAgent、RunCronChat、employee 等渠道统一使用。
func InvokeChatSend(ctx *Context, p ChatSendParams) (runID string, ok bool) {
	if ctx == nil || ctx.InvokeMethod == nil {
		return "", false
	}
	params := map[string]interface{}{
		"sessionKey":     p.SessionKey,
		"message":        p.Message,
		"idempotencyKey": p.IdempotencyKey,
	}
	if p.SessionID != "" {
		params["sessionId"] = p.SessionID
	}
	if p.Channel != "" {
		params["channel"] = p.Channel
	}
	if p.To != "" {
		params["to"] = p.To
	}
	if p.ChatType != "" {
		params["chatType"] = p.ChatType
	}
	if p.MessageID != "" {
		params["messageId"] = p.MessageID
	}
	if p.Header != "" {
		params["header"] = p.Header
	}
	if len(p.Metadata) > 0 {
		params["deliverMetadata"] = p.Metadata
	}
	if p.Thinking != "" {
		params["thinking"] = p.Thinking
	}
	if p.TimeoutMs > 0 {
		params["timeoutMs"] = p.TimeoutMs
	}
	if len(p.Attachments) > 0 {
		params["attachments"] = p.Attachments
	}
	if p.ModelRef != "" {
		params["modelRef"] = p.ModelRef
	}
	if len(p.Resources) > 0 {
		params["resources"] = p.Resources
	}
	ok, payload, _ := ctx.InvokeMethod("chat.send", params)
	if !ok || payload == nil {
		return "", false
	}
	m, ok := payload.(map[string]interface{})
	if !ok {
		return "", false
	}
	rid, _ := m["runId"].(string)
	return rid, rid != ""
}

// InvokeChatSendWithError 与 InvokeChatSend 相同，但返回协议错误供调用方处理。
func InvokeChatSendWithError(ctx *Context, p ChatSendParams) (runID string, ok bool, err *protocol.ErrorShape) {
	if ctx == nil || ctx.InvokeMethod == nil {
		return "", false, nil
	}
	params := map[string]interface{}{
		"sessionKey":     p.SessionKey,
		"message":        p.Message,
		"idempotencyKey": p.IdempotencyKey,
	}
	if p.SessionID != "" {
		params["sessionId"] = p.SessionID
	}
	if p.Channel != "" {
		params["channel"] = p.Channel
	}
	if p.To != "" {
		params["to"] = p.To
	}
	if p.ChatType != "" {
		params["chatType"] = p.ChatType
	}
	if p.MessageID != "" {
		params["messageId"] = p.MessageID
	}
	if p.Header != "" {
		params["header"] = p.Header
	}
	if len(p.Metadata) > 0 {
		params["deliverMetadata"] = p.Metadata
	}
	if p.Thinking != "" {
		params["thinking"] = p.Thinking
	}
	if p.TimeoutMs > 0 {
		params["timeoutMs"] = p.TimeoutMs
	}
	if len(p.Attachments) > 0 {
		params["attachments"] = p.Attachments
	}
	if p.ModelRef != "" {
		params["modelRef"] = p.ModelRef
	}
	if len(p.Resources) > 0 {
		params["resources"] = p.Resources
	}
	ok, payload, errShape := ctx.InvokeMethod("chat.send", params)
	if !ok {
		return "", false, errShape
	}
	if payload == nil {
		return "", false, errShape
	}
	m, ok := payload.(map[string]interface{})
	if !ok {
		return "", false, errShape
	}
	rid, _ := m["runId"].(string)
	return rid, rid != "", errShape
}
