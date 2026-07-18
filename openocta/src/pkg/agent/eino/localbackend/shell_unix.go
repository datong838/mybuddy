//go:build !windows

package localbackend

import (
	"context"

	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/schema"
)

func (b *Backend) Execute(ctx context.Context, input *filesystem.ExecuteRequest) (*filesystem.ExecuteResponse, error) {
	normalized, err := normalizeExecuteRequest(input)
	if err != nil {
		return nil, err
	}
	return b.Local.Execute(ctx, normalized)
}

func (b *Backend) ExecuteStreaming(ctx context.Context, input *filesystem.ExecuteRequest) (*schema.StreamReader[*filesystem.ExecuteResponse], error) {
	normalized, err := normalizeExecuteRequest(input)
	if err != nil {
		return nil, err
	}
	return b.Local.ExecuteStreaming(ctx, normalized)
}
