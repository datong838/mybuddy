package feishu

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/channel"
	channeltypes "github.com/larksuite/oapi-sdk-go/v3/channel/types"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/openocta/openocta/pkg/channels"
)

// runtimeLoggerKey is used to prefix log messages; kept minimal to avoid pulling logging deps.
const runtimeLoggerKey = "[feishu-runtime]"

// Runtime 实现 Feishu 的 RuntimeChannel，基于 oapi-sdk-go v3.9.7 Channel 模块收发消息。
type Runtime struct {
	*channels.BaseRuntimeImpl

	appID             string
	appSecret         string
	domain            string
	encryptKey        string
	verificationToken string

	ch         channeltypes.Channel
	wsClient   *larkws.Client
	httpClient *lark.Client

	allowedIDs []string
	botOpenId  string
	botMu      sync.RWMutex
}

// NewRuntime 创建 Feishu Runtime 实例。
func NewRuntime(appID, appSecret, domain, encryptKey, verificationToken string, cfg channels.BaseRuntimeConfig, sink channels.InboundSink) *Runtime {
	base := channels.NewBaseRuntimeImpl("feishu", cfg.AccountID, cfg, sink)

	client := lark.NewClient(
		appID,
		appSecret,
		lark.WithAppType(larkcore.AppTypeSelfBuilt),
		lark.WithOpenBaseUrl(resolveDomain(domain)),
	)

	return &Runtime{
		BaseRuntimeImpl:   base,
		appID:             appID,
		appSecret:         appSecret,
		domain:            domain,
		encryptKey:        encryptKey,
		verificationToken: verificationToken,
		httpClient:        client,
		allowedIDs:        append([]string(nil), cfg.AllowedIDs...),
	}
}

// Start 启动飞书入站 WebSocket。助手出站由网关 handlers.deliverAssistantToIM 在整轮流式结束后调用 Send。
func (r *Runtime) Start(ctx context.Context) error {
	if err := r.BaseRuntimeImpl.Start(ctx); err != nil {
		return err
	}

	eventDispatcher := dispatcher.NewEventDispatcher(r.verificationToken, r.encryptKey)
	r.wsClient = larkws.NewClient(
		r.appID,
		r.appSecret,
		larkws.WithEventHandler(eventDispatcher),
		larkws.WithDomain(resolveDomain(r.domain)),
		larkws.WithLogLevel(larkcore.LogLevelInfo),
	)

	var channelOpts []channeltypes.ChannelOption
	if ids := r.allowedSenderIDs(); len(ids) > 0 {
		channelOpts = append(channelOpts, channeltypes.WithPolicyConfig(channeltypes.PolicyConfig{
			DMMode:      "allowlist",
			DMAllowlist: ids,
		}))
	}

	r.ch = channel.NewChannel(r.httpClient, r.wsClient, channelOpts...)

	r.ch.OnMessage(func(ctx context.Context, msg *channeltypes.NormalizedMessage) error {
		return r.handleNormalizedMessage(ctx, msg)
	})

	r.ch.OnReady(func() {
		if bot := r.ch.GetBotIdentity(context.Background()); bot != nil {
			r.setBotOpenID(bot.OpenID)
		}
		r.BaseRuntimeImpl.MarkConnectionRestored()
	})

	r.ch.OnError(func(err error) {
		fmt.Println(runtimeLoggerKey, "channel error:", err)
		r.BaseRuntimeImpl.MarkConnectionFailed(err)
	})

	r.ch.OnReconnected(func() {
		r.BaseRuntimeImpl.MarkConnectionRestored()
	})

	go func() {
		if err := r.ch.Start(ctx); err != nil {
			fmt.Println(runtimeLoggerKey, "WebSocket error:", err)
			r.BaseRuntimeImpl.MarkConnectionFailed(err)
		}
	}()

	return nil
}

// Stop 停止运行时。
func (r *Runtime) Stop() error {
	if r.ch != nil {
		_ = r.ch.Stop(context.Background())
	}
	return r.BaseRuntimeImpl.Stop()
}

// RuntimeStatus 返回 Feishu 运行时的状态，包含 appId、domain、botOpenId 等平台信息。
func (r *Runtime) RuntimeStatus() channels.RuntimeStatus {
	s := r.BaseRuntimeImpl.RuntimeStatus()
	if s.Extra == nil {
		s.Extra = make(map[string]interface{})
	}
	if r.appID != "" {
		s.Extra["appId"] = r.appID
	}
	if r.domain != "" {
		s.Extra["domain"] = r.domain
	}
	if botOpenID := r.getBotOpenID(); botOpenID != "" {
		s.Extra["botOpenId"] = botOpenID
	}
	if r.appID != "" || r.getBotOpenID() != "" {
		probe := map[string]interface{}{"ok": r.BaseRuntimeImpl.IsRunning()}
		if r.appID != "" {
			probe["appId"] = r.appID
		}
		if botOpenID := r.getBotOpenID(); botOpenID != "" {
			probe["botOpenId"] = botOpenID
		}
		s.Extra["probe"] = probe
	}
	if s.LastStartAt != nil {
		s.Extra["lastProbeAt"] = *s.LastStartAt
	}
	return s
}

// Send 发送一条消息到 Feishu，按 Channel 模块支持的消息类型自动适配。
func (r *Runtime) Send(msg *channels.RuntimeOutboundMessage) error {
	if msg == nil {
		fmt.Println(runtimeLoggerKey, "Send called with nil message")
		return nil
	}
	if r.ch == nil {
		return fmt.Errorf("feishu runtime: channel not started")
	}

	chatID := resolveOutboundChatID(msg)
	if chatID == "" {
		fmt.Println(runtimeLoggerKey, "Send failed: chatId is required")
		return fmt.Errorf("feishu runtime: chatId is required for Send")
	}

	ctx := context.Background()
	var lastErr error
	sent := false

	for _, m := range msg.Media {
		input, err := r.buildMediaSendInput(chatID, m, msg.ReplyToID)
		if err != nil {
			fmt.Println(runtimeLoggerKey, "failed to build media send input:", err)
			lastErr = err
			continue
		}
		if _, err = r.ch.Send(ctx, input); err != nil {
			fmt.Println(runtimeLoggerKey, "failed to send media message:", err)
			lastErr = err
			continue
		}
		sent = true
	}

	if input := r.buildContentSendInput(chatID, msg); input != nil {
		if _, err := r.ch.Send(ctx, input); err != nil {
			fmt.Println(runtimeLoggerKey, "failed to send content message:", err)
			return fmt.Errorf("feishu runtime: failed to send message: %w", err)
		}
		sent = true
	}

	if sent {
		fmt.Printf("%s Feishu message sent successfully, chat_id=%s\n", runtimeLoggerKey, chatID)
	} else if lastErr != nil {
		return lastErr
	} else {
		fmt.Printf("%s Feishu Send: nothing to send (no text or media), chat_id=%s\n", runtimeLoggerKey, chatID)
	}

	return lastErr
}

// SendStream 聚合最终输出片段后调用 Send（避免将思考/中间态发到飞书）。
func (r *Runtime) SendStream(chatID string, stream <-chan *channels.RuntimeStreamChunk) error {
	var buf strings.Builder
	for chunk := range stream {
		if chunk == nil {
			continue
		}
		if chunk.Error != "" {
			return fmt.Errorf("stream error: %s", chunk.Error)
		}
		if chunk.IsComplete {
			buf.WriteString(chunk.Content)
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

func (r *Runtime) handleNormalizedMessage(ctx context.Context, msg *channeltypes.NormalizedMessage) error {
	if msg == nil {
		return nil
	}

	content := strings.TrimSpace(msg.Content)
	media := resourcesToRuntimeMedia(msg.Resources)
	if content == "" && len(media) == 0 {
		return nil
	}

	if msg.UserID != "" && !r.IsAllowed(msg.UserID) {
		return nil
	}

	if msg.MessageID != "" {
		_ = r.addReactionToMessage(msg.MessageID, "Typing")
	}

	ts := time.UnixMilli(msg.CreateTimeMs)
	if msg.CreateTimeMs == 0 {
		ts = time.Now()
	}

	in := &channels.InboundMessage{
		ID:       msg.MessageID,
		SenderID: msg.UserID,
		ChatID:   msg.ChatID,
		ChatType: msg.ChatType,
		Content:  content,
		Media:    media,
		Time:     ts,
		Meta: map[string]interface{}{
			"rawContentType": msg.RawContentType,
			"mentionedBot":   msg.MentionedBot,
			"mentionAll":     msg.MentionAll,
		},
	}

	return r.PublishInbound(ctx, in)
}

func (r *Runtime) buildContentSendInput(chatID string, msg *channels.RuntimeOutboundMessage) *channeltypes.SendInput {
	content := strings.TrimSpace(msg.Content)
	if content == "" && msg.MetadataString("card") == "" {
		return nil
	}

	input := &channeltypes.SendInput{
		ReplyMessageID: msg.ReplyToID,
	}
	applyChatTarget(input, chatID)

	if title := msg.MetadataString("header"); title != "" {
		input.Title = title
	}
	if card := msg.MetadataString("card"); card != "" {
		input.Card = card
		return input
	}
	if post := msg.MetadataString("post"); post != "" {
		input.Post = post
		return input
	}
	if shareChatID := msg.MetadataString("shareChatId"); shareChatID != "" {
		input.ShareChatID = shareChatID
		return input
	}
	if shareUserID := msg.MetadataString("shareUserId"); shareUserID != "" {
		input.ShareUserID = shareUserID
		return input
	}

	msgType := strings.ToLower(strings.TrimSpace(msg.MetadataString("msgType")))
	if msgType == "" {
		msgType = strings.ToLower(strings.TrimSpace(msg.MetadataString("msg_type")))
	}

	switch msgType {
	case "text":
		input.Text = content
	case "post":
		input.Post = content
	case "interactive", "card":
		input.Card = content
	default:
		// Markdown 由 SDK 转为 post，支持表格、代码块等富文本。
		input.Markdown = content
	}

	return input
}

func (r *Runtime) buildMediaSendInput(chatID string, media channels.RuntimeMedia, replyID string) (*channeltypes.SendInput, error) {
	input := &channeltypes.SendInput{
		ReplyMessageID: replyID,
	}
	applyChatTarget(input, chatID)

	mediaType := strings.ToLower(strings.TrimSpace(media.Type))
	if mediaType == "" {
		mediaType = "image"
	}

	if strings.HasPrefix(media.URL, "feishu:") {
		key := strings.TrimPrefix(media.URL, "feishu:")
		switch mediaType {
		case "image":
			input.ImageKey = key
		case "audio":
			input.AudioKey = key
		case "video":
			input.VideoKey = key
		case "sticker":
			input.StickerFileKey = key
		default:
			input.FileKey = key
		}
		return input, nil
	}

	if path := mediaExtraString(media, "path"); path != "" {
		switch mediaType {
		case "image":
			input.ImagePath = path
		case "audio", "video", "file", "sticker":
			input.FilePath = path
		default:
			input.FilePath = path
		}
		return input, nil
	}

	upload := &channeltypes.UploadInput{}
	switch mediaType {
	case "image":
		upload.Kind = channeltypes.MediaKindImage
	case "audio":
		upload.Kind = channeltypes.MediaKindAudio
	case "video":
		upload.Kind = channeltypes.MediaKindVideo
	default:
		upload.Kind = channeltypes.MediaKindFile
	}

	if media.Base64 != "" {
		data, err := base64.StdEncoding.DecodeString(media.Base64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 media: %w", err)
		}
		upload.SourceBytes = data
	} else if media.URL != "" {
		upload.SourceURL = media.URL
	} else {
		return nil, fmt.Errorf("no valid media data (URL, Base64, or path)")
	}

	if name := mediaExtraString(media, "fileName"); name != "" {
		upload.FileName = name
	}
	input.Media = upload
	return input, nil
}

func applyChatTarget(input *channeltypes.SendInput, chatID string) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return
	}
	if strings.HasPrefix(chatID, "oc_") {
		input.ChatID = chatID
		return
	}
	if strings.HasPrefix(chatID, "ou_") {
		input.ReceiveID = chatID
		return
	}
	input.ChatID = chatID
}

func resolveOutboundChatID(msg *channels.RuntimeOutboundMessage) string {
	if msg == nil {
		return ""
	}
	if chatID := strings.TrimSpace(msg.ChatID); chatID != "" {
		return chatID
	}
	if chatID := msg.MetadataString("chatId"); chatID != "" {
		return chatID
	}
	return msg.MetadataString("receiveId")
}

func resourcesToRuntimeMedia(resources []channeltypes.Resource) []channels.RuntimeMedia {
	if len(resources) == 0 {
		return nil
	}
	media := make([]channels.RuntimeMedia, 0, len(resources))
	for _, res := range resources {
		if res.FileKey == "" {
			continue
		}
		item := channels.RuntimeMedia{
			Type: res.Type,
			URL:  "feishu:" + res.FileKey,
			Extra: map[string]interface{}{
				"fileKey": res.FileKey,
			},
		}
		if res.FileName != "" {
			item.Extra["fileName"] = res.FileName
		}
		if res.CoverImageKey != "" {
			item.Extra["coverImageKey"] = res.CoverImageKey
		}
		media = append(media, item)
	}
	return media
}

func mediaExtraString(media channels.RuntimeMedia, key string) string {
	if media.Extra == nil {
		return ""
	}
	if v, ok := media.Extra[key]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func (r *Runtime) allowedSenderIDs() []string {
	return r.allowedIDs
}

func (r *Runtime) setBotOpenID(openID string) {
	r.botMu.Lock()
	r.botOpenId = openID
	r.botMu.Unlock()
}

func (r *Runtime) getBotOpenID() string {
	r.botMu.RLock()
	defer r.botMu.RUnlock()
	return r.botOpenId
}

// addReactionToMessage 在用户消息上添加表情回复（如敲键盘），表示正在处理。
func (r *Runtime) addReactionToMessage(messageID, emojiType string) error {
	if messageID == "" || emojiType == "" {
		return nil
	}
	emoji := emojiType
	req := larkim.NewCreateMessageReactionReqBuilder().
		MessageId(messageID).
		Body(larkim.NewCreateMessageReactionReqBodyBuilder().
			ReactionType(&larkim.Emoji{EmojiType: &emoji}).
			Build()).
		Build()
	resp, err := r.httpClient.Im.MessageReaction.Create(context.Background(), req)
	if err != nil {
		return err
	}
	if !resp.Success() {
		return fmt.Errorf("feishu add reaction: %d %s", resp.Code, resp.Msg)
	}
	return nil
}
