package weixin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	ilink "github.com/openilink/openilink-sdk-go"
	"github.com/openocta/openocta/pkg/channels"
)

// Runtime 使用微信 iLink Bot HTTPS 长轮询（openilink-sdk-go）收发个人微信消息。
type Runtime struct {
	*channels.BaseRuntimeImpl

	botToken  string
	botID     string
	baseURL   string
	botType   string
	routeTag  string
	syncBuf   string
	loginUser string

	mu            sync.Mutex
	client        *ilink.Client
	monitorDown   bool
	lastInboundMs int64
}

// NewRuntime 使用 botToken + botId 创建个人微信 iLink 运行时。
func NewRuntime(botToken, botID, baseURL, botType, routeTag, syncBuf, loginUser string, cfg channels.BaseRuntimeConfig, sink channels.InboundSink) *Runtime {
	base := channels.NewBaseRuntimeImpl(channelID, cfg.AccountID, cfg, sink)
	return &Runtime{
		BaseRuntimeImpl: base,
		botToken:        strings.TrimSpace(botToken),
		botID:           strings.TrimSpace(botID),
		baseURL:         strings.TrimSpace(baseURL),
		botType:         strings.TrimSpace(botType),
		routeTag:        strings.TrimSpace(routeTag),
		syncBuf:         strings.TrimSpace(syncBuf),
		loginUser:       strings.TrimSpace(loginUser),
	}
}

func (r *Runtime) ilinkOptions() []ilink.Option {
	var opts []ilink.Option
	if r.baseURL != "" {
		opts = append(opts, ilink.WithBaseURL(r.baseURL))
	}
	if r.botType != "" {
		opts = append(opts, ilink.WithBotType(r.botType))
	}
	if r.routeTag != "" {
		opts = append(opts, ilink.WithRouteTag(r.routeTag))
	}
	return opts
}

// Start 启动 iLink getUpdates 长轮询；出站由网关在流式结束后调用 Send。
func (r *Runtime) Start(ctx context.Context) error {
	if err := r.BaseRuntimeImpl.Start(ctx); err != nil {
		return err
	}
	if !r.BaseRuntimeImpl.IsRunning() {
		return nil
	}
	r.BaseRuntimeImpl.MarkConnectionFailed(nil)
	if r.botToken == "" || r.botID == "" {
		r.SetLastError(fmt.Errorf("weixin: botToken and botId are required when enabled"))
		return nil
	}
	go r.runMonitor(ctx)
	return nil
}

func (r *Runtime) runMonitor(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	go func() {
		select {
		case <-r.WaitForStop():
			cancel()
		case <-parent.Done():
		}
	}()

	cl := ilink.NewClient(r.botToken, r.ilinkOptions()...)
	r.mu.Lock()
	r.client = cl
	r.monitorDown = false
	r.mu.Unlock()

	initial := r.syncBuf
	err := cl.Monitor(ctx, func(msg ilink.WeixinMessage) {
		if msg.MessageType != ilink.MsgTypeUser {
			return
		}
		text := ilink.ExtractText(&msg)
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		sender := strings.TrimSpace(msg.FromUserID)
		if sender == "" {
			return
		}
		if !r.IsAllowed(sender) {
			return
		}
		chatID := sender
		var ts time.Time
		if msg.CreateTimeMs > 0 {
			ts = time.UnixMilli(msg.CreateTimeMs)
		} else {
			ts = time.Now()
		}
		atomic.StoreInt64(&r.lastInboundMs, time.Now().UnixMilli())
		msgID := ""
		if msg.MessageID != 0 {
			msgID = strconv.FormatInt(msg.MessageID, 10)
		}
		in := &channels.InboundMessage{
			ID:       msgID,
			SenderID: sender,
			ChatID:   chatID,
			ChatType: "dm",
			Content:  text,
			Time:     ts,
			Meta: map[string]interface{}{
				"msgtype":       "text",
				"ilink":         true,
				"context_token": msg.ContextToken,
			},
		}
		_ = r.PublishInbound(context.Background(), in)
	}, &ilink.MonitorOptions{
		InitialBuf: initial,
		OnBufUpdate: func(buf string) {
			r.mu.Lock()
			r.syncBuf = buf
			r.mu.Unlock()
		},
		OnError: func(err error) {
			if err != nil {
				r.SetLastError(err)
			}
		},
		OnSessionExpired: func() {
			r.mu.Lock()
			r.monitorDown = true
			r.mu.Unlock()
			r.SetLastError(fmt.Errorf("weixin: iLink session expired; re-login with QR"))
			r.BaseRuntimeImpl.MarkConnectionFailed(fmt.Errorf("weixin: session expired"))
		},
	})

	r.mu.Lock()
	r.client = nil
	r.mu.Unlock()

	if err != nil && ctx.Err() == nil {
		r.BaseRuntimeImpl.MarkConnectionFailed(fmt.Errorf("weixin monitor: %w", err))
	}
}

// Stop 停止长轮询。
func (r *Runtime) Stop() error {
	return r.BaseRuntimeImpl.Stop()
}

// Send 使用缓存的 context_token 回复文本（需用户先发过消息）。
func (r *Runtime) Send(msg *channels.RuntimeOutboundMessage) error {
	if msg == nil {
		return nil
	}
	r.mu.Lock()
	cl := r.client
	r.mu.Unlock()
	if cl == nil {
		return fmt.Errorf("weixin runtime: client not ready")
	}
	to := strings.TrimSpace(msg.ChatID)
	if to == "" {
		to = strings.TrimSpace(msg.MetadataString("chat_id"))
	}
	if to == "" {
		return fmt.Errorf("weixin runtime: chatID (peer user id) is required for Send")
	}
	token, ok := cl.GetContextToken(to)
	if !ok {
		token = strings.TrimSpace(msg.MetadataString("context_token"))
	}
	if token == "" {
		return fmt.Errorf("weixin runtime: no context_token for peer; user must message the bot first")
	}
	plain := stripSimpleMarkdown(msg.Content)
	_, err := cl.SendText(context.Background(), to, plain, token)
	if err != nil {
		return fmt.Errorf("weixin runtime: send text: %w", err)
	}
	return nil
}

// SendStream 聚合最终输出后一次性发送。
func (r *Runtime) SendStream(chatID string, stream <-chan *channels.RuntimeStreamChunk) error {
	var buf strings.Builder
	for chunk := range stream {
		if chunk == nil {
			continue
		}
		if chunk.Error != "" {
			return fmt.Errorf("weixin runtime stream error: %s", chunk.Error)
		}
		if !chunk.IsThinking && chunk.IsFinal {
			buf.WriteString(chunk.Content)
		}
		if chunk.IsComplete {
			break
		}
	}
	if buf.Len() == 0 {
		return nil
	}
	return r.Send(&channels.RuntimeOutboundMessage{
		ChatID:  chatID,
		Content: buf.String(),
	})
}

// RuntimeStatus 反映长轮询是否正常工作。
func (r *Runtime) RuntimeStatus() channels.RuntimeStatus {
	s := r.BaseRuntimeImpl.RuntimeStatus()
	r.mu.Lock()
	down := r.monitorDown
	cl := r.client
	r.mu.Unlock()
	connected := cl != nil && !down
	s.Running = r.Enabled() && connected
	if s.Extra == nil {
		s.Extra = map[string]interface{}{}
	}
	s.Extra["transport"] = "weixin_ilink_poll"
	s.Extra["botIdMasked"] = maskID(r.botID)
	if r.loginUser != "" {
		s.Extra["ilinkUserIdMasked"] = maskID(r.loginUser)
	}
	if ts := atomic.LoadInt64(&r.lastInboundMs); ts > 0 {
		s.Extra["lastInboundAt"] = ts
	}
	return s
}

func maskID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 6 {
		if id == "" {
			return ""
		}
		return "****"
	}
	return id[:3] + "…" + id[len(id)-3:]
}

// 与 formatChannelReply 产生的轻量 Markdown 相比，个微文本通道做极简剥离。
func stripSimpleMarkdown(s string) string {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "**") && !strings.Contains(s, "`") {
		return s
	}
	out := strings.ReplaceAll(s, "**", "")
	out = strings.ReplaceAll(out, "`", "")
	return out
}
