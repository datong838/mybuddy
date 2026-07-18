package eino

import (
	"context"

	einomodel "github.com/cloudwego/eino/components/model"
)

// ChatModelFactory resolves a ToolCallingChatModel for an agent run.
type ChatModelFactory interface {
	ChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error)
}

type chatModelFactoryFunc func(context.Context) (einomodel.ToolCallingChatModel, error)

func (fn chatModelFactoryFunc) ChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	if fn == nil {
		return nil, ErrMissingModel
	}
	return fn(ctx)
}

// ChatModelFactoryFunc wraps a function as ChatModelFactory.
func ChatModelFactoryFunc(fn func(context.Context) (einomodel.ToolCallingChatModel, error)) ChatModelFactory {
	return chatModelFactoryFunc(fn)
}
