package eino

import (
	"context"
	"encoding/json"
	"fmt"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"

	octool "github.com/openocta/openocta/pkg/agent/tool"
)

type wrappedTool struct {
	inner octool.Tool
}

func WrapTool(t octool.Tool) einotool.InvokableTool {
	if t == nil {
		return nil
	}
	return &wrappedTool{inner: t}
}

func WrapTools(tools []octool.Tool) []einotool.BaseTool {
	if len(tools) == 0 {
		return nil
	}
	out := make([]einotool.BaseTool, 0, len(tools))
	for _, t := range tools {
		if t == nil {
			continue
		}
		out = append(out, WrapTool(t))
	}
	return out
}

func (w *wrappedTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	paramsOneOf, err := toolSchemaToParamsOneOf(w.inner.Schema())
	if err != nil {
		return nil, err
	}
	return &schema.ToolInfo{
		Name:        w.inner.Name(),
		Desc:        w.inner.Description(),
		ParamsOneOf: paramsOneOf,
	}, nil
}

func toolSchemaToParamsOneOf(s *octool.JSONSchema) (*schema.ParamsOneOf, error) {
	if s == nil {
		return schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}), nil
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal tool schema: %w", err)
	}
	var js jsonschema.Schema
	if err := json.Unmarshal(raw, &js); err != nil {
		return nil, fmt.Errorf("decode tool schema: %w", err)
	}
	return schema.NewParamsOneOfByJSONSchema(&js), nil
}

func (w *wrappedTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	var params map[string]interface{}
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			return "", fmt.Errorf("decode tool args: %w", err)
		}
	}
	if params == nil {
		params = map[string]interface{}{}
	}
	res, err := w.inner.Execute(ctx, params)
	if err != nil {
		return "", err
	}
	if res == nil {
		return "", nil
	}
	if res.Error != nil {
		return res.Output, res.Error
	}
	return res.Output, nil
}
