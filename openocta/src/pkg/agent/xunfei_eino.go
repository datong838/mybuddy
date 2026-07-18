package agent

import (
	"context"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/openocta/openocta/pkg/agent/eino"
	octmodel "github.com/openocta/openocta/pkg/agent/model"
	"github.com/openocta/openocta/pkg/config"
)

type xunfeiChatModel struct {
	inner *XunfeiImageModel
	tools []*schema.ToolInfo
}

func newXunfeiChatModel(cfg XunfeiImageConfig) model.ToolCallingChatModel {
	return &xunfeiChatModel{inner: &XunfeiImageModel{config: cfg}}
}

func (m *xunfeiChatModel) Generate(ctx context.Context, input []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	req := schemaMessagesToXunfeiRequest(input)
	resp, err := m.inner.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return schema.AssistantMessage("", nil), nil
	}
	return schema.AssistantMessage(resp.Message.Content, nil), nil
}

func (m *xunfeiChatModel) Stream(ctx context.Context, input []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	req := schemaMessagesToXunfeiRequest(input)
	sr, sw := schema.Pipe[*schema.Message](8)
	go func() {
		defer sw.Close()
		_ = m.inner.CompleteStream(ctx, req, func(chunk octmodel.StreamResult) error {
			if chunk.Delta != "" {
				sw.Send(schema.AssistantMessage(chunk.Delta, nil), nil)
			}
			return nil
		})
	}()
	return sr, nil
}

func (m *xunfeiChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	cp := *m
	cp.tools = tools
	return &cp, nil
}

func schemaMessagesToXunfeiRequest(msgs []*schema.Message) octmodel.Request {
	var out octmodel.Request
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		entry := octmodel.Message{Role: string(msg.Role), Content: msg.Content}
		for _, part := range msg.MultiContent {
			switch part.Type {
			case schema.ChatMessagePartTypeText:
				if part.Text != "" {
					entry.ContentBlocks = append(entry.ContentBlocks, octmodel.ContentBlock{
						Type: octmodel.ContentBlockText,
						Text: part.Text,
					})
				}
			case schema.ChatMessagePartTypeImageURL:
				url := ""
				if part.ImageURL != nil {
					url = part.ImageURL.URL
				}
				if url != "" {
					entry.ContentBlocks = append(entry.ContentBlocks, octmodel.ContentBlock{
						Type: octmodel.ContentBlockImage,
						URL:  url,
					})
				}
			}
		}
		out.Messages = append(out.Messages, entry)
	}
	if len(out.Messages) > 0 {
		last := out.Messages[len(out.Messages)-1]
		if strings.TrimSpace(last.Content) == "" && len(last.ContentBlocks) > 0 {
			for _, b := range last.ContentBlocks {
				if b.Type == octmodel.ContentBlockText && strings.TrimSpace(b.Text) != "" {
					last.Content = b.Text
					break
				}
			}
			out.Messages[len(out.Messages)-1] = last
		}
	}
	return out
}

// CreateXunfeiImageFactory returns an Eino ChatModelFactory for Xunfei image understanding.
func CreateXunfeiImageFactory(cfg *config.OpenOctaConfig) eino.ChatModelFactory {
	env := os.Getenv
	if cfg != nil && cfg.Env != nil && cfg.Env.Vars != nil {
		env = func(k string) string {
			if v, ok := cfg.Env.Vars[k]; ok && v != "" {
				return v
			}
			return os.Getenv(k)
		}
	}
	provider := NewXunfeiImageFactory(env)
	if provider == nil {
		return nil
	}
	xCfg := provider.ImageConfig()
	if strings.TrimSpace(xCfg.AppID) == "" || strings.TrimSpace(xCfg.APIKey) == "" || strings.TrimSpace(xCfg.APISecret) == "" {
		return nil
	}
	return eino.ChatModelFactoryFunc(func(ctx context.Context) (model.ToolCallingChatModel, error) {
		return newXunfeiChatModel(xCfg), nil
	})
}
