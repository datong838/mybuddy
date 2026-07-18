package handlers

import (
	"errors"
	"strings"

	"github.com/openocta/openocta/pkg/apikeys"
	"github.com/openocta/openocta/pkg/gateway/protocol"
)

func apiKeyService(ctx *Context) (*apikeys.Service, *protocol.ErrorShape) {
	if ctx == nil || ctx.ApiKeyService == nil {
		return nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "api key service not configured",
		}
	}
	return ctx.ApiKeyService, nil
}

func parseStringList(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func parseOptionalInt64(v interface{}) *int64 {
	switch n := v.(type) {
	case float64:
		i := int64(n)
		return &i
	case int64:
		return &n
	case int:
		i := int64(n)
		return &i
	default:
		return nil
	}
}

func parseCreateParams(params map[string]interface{}) (apikeys.CreateParams, error) {
	name, _ := params["name"].(string)
	bindingMode, _ := params["bindingMode"].(string)
	employeeID, _ := params["digitalEmployeeId"].(string)
	paths := parseStringList(params["allowedPaths"])
	if raw, ok := params["allowedPaths"].([]string); ok && len(raw) > 0 {
		paths = raw
	}
	return apikeys.CreateParams{
		Name:              name,
		AllowedPaths:      paths,
		BindingMode:       bindingMode,
		AllowedModels:     parseStringList(params["allowedModels"]),
		SkillKeys:         parseStringList(params["skillKeys"]),
		McpServers:        parseStringList(params["mcpServers"]),
		DigitalEmployeeID: employeeID,
		MonthlyTokenLimit: parseOptionalInt64(params["monthlyTokenLimit"]),
	}, nil
}

func parseUpdateParams(params map[string]interface{}) (string, apikeys.UpdateParams, error) {
	id, _ := params["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return "", apikeys.UpdateParams{}, errors.New("missing id")
	}
	patch := apikeys.UpdateParams{}
	if v, ok := params["name"].(string); ok {
		trimmed := strings.TrimSpace(v)
		patch.Name = &trimmed
	}
	if v, ok := params["bindingMode"].(string); ok {
		trimmed := strings.TrimSpace(v)
		patch.BindingMode = &trimmed
	}
	if v, ok := params["digitalEmployeeId"].(string); ok {
		trimmed := strings.TrimSpace(v)
		patch.DigitalEmployeeID = &trimmed
	}
	if v, ok := params["enabled"].(bool); ok {
		patch.Enabled = &v
	}
	if v, ok := params["monthlyTokenLimit"]; ok {
		patch.MonthlyTokenLimit = parseOptionalInt64(v)
	}
	if v, ok := params["allowedPaths"]; ok {
		list := parseStringList(v)
		patch.AllowedPaths = &list
	}
	if v, ok := params["allowedModels"]; ok {
		list := parseStringList(v)
		patch.AllowedModels = &list
	}
	if v, ok := params["skillKeys"]; ok {
		list := parseStringList(v)
		patch.SkillKeys = &list
	}
	if v, ok := params["mcpServers"]; ok {
		list := parseStringList(v)
		patch.McpServers = &list
	}
	return id, patch, nil
}

// ApiKeysListHandler handles "apiKeys.list".
func ApiKeysListHandler(opts HandlerOpts) error {
	svc, errShape := apiKeyService(opts.Context)
	if errShape != nil {
		opts.Respond(false, nil, errShape, nil)
		return nil
	}
	keys, err := svc.List()
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, map[string]interface{}{"keys": keys}, nil, nil)
	return nil
}

// ApiKeysCreateHandler handles "apiKeys.create".
func ApiKeysCreateHandler(opts HandlerOpts) error {
	svc, errShape := apiKeyService(opts.Context)
	if errShape != nil {
		opts.Respond(false, nil, errShape, nil)
		return nil
	}
	p, err := parseCreateParams(opts.Params)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: err.Error(),
		}, nil)
		return nil
	}
	result, err := svc.Create(p)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, result, nil, nil)
	return nil
}

// ApiKeysUpdateHandler handles "apiKeys.update".
func ApiKeysUpdateHandler(opts HandlerOpts) error {
	svc, errShape := apiKeyService(opts.Context)
	if errShape != nil {
		opts.Respond(false, nil, errShape, nil)
		return nil
	}
	id, patch, err := parseUpdateParams(opts.Params)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: err.Error(),
		}, nil)
		return nil
	}
	entry, err := svc.Update(id, patch)
	if err != nil {
		code := protocol.ErrCodeInternal
		if errors.Is(err, apikeys.ErrNotFound) {
			code = protocol.ErrCodeInvalidRequest
		}
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    code,
			Message: err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, entry, nil, nil)
	return nil
}

// ApiKeysRemoveHandler handles "apiKeys.remove".
func ApiKeysRemoveHandler(opts HandlerOpts) error {
	svc, errShape := apiKeyService(opts.Context)
	if errShape != nil {
		opts.Respond(false, nil, errShape, nil)
		return nil
	}
	id, _ := opts.Params["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "missing id",
		}, nil)
		return nil
	}
	if err := svc.Remove(id); err != nil {
		code := protocol.ErrCodeInternal
		if errors.Is(err, apikeys.ErrNotFound) {
			code = protocol.ErrCodeInvalidRequest
		}
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    code,
			Message: err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, map[string]interface{}{"ok": true, "removed": id}, nil, nil)
	return nil
}

// ApiKeysSecretHandler handles "apiKeys.secret".
func ApiKeysSecretHandler(opts HandlerOpts) error {
	svc, errShape := apiKeyService(opts.Context)
	if errShape != nil {
		opts.Respond(false, nil, errShape, nil)
		return nil
	}
	id, _ := opts.Params["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "missing id",
		}, nil)
		return nil
	}
	result, err := svc.GetSecret(id)
	if err != nil {
		code := protocol.ErrCodeInternal
		if errors.Is(err, apikeys.ErrNotFound) {
			code = protocol.ErrCodeInvalidRequest
		}
		if errors.Is(err, apikeys.ErrSecretUnavailable) {
			code = protocol.ErrCodeInvalidRequest
		}
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    code,
			Message: err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, result, nil, nil)
	return nil
}

// ApiKeysRegenerateHandler handles "apiKeys.regenerate".
func ApiKeysRegenerateHandler(opts HandlerOpts) error {
	svc, errShape := apiKeyService(opts.Context)
	if errShape != nil {
		opts.Respond(false, nil, errShape, nil)
		return nil
	}
	id, _ := opts.Params["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "missing id",
		}, nil)
		return nil
	}
	result, err := svc.RegenerateSecret(id)
	if err != nil {
		code := protocol.ErrCodeInternal
		if errors.Is(err, apikeys.ErrNotFound) {
			code = protocol.ErrCodeInvalidRequest
		}
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    code,
			Message: err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, result, nil, nil)
	return nil
}

// ApiKeysDefaultsHandler handles "apiKeys.defaults".
func ApiKeysDefaultsHandler(opts HandlerOpts) error {
	opts.Respond(true, map[string]interface{}{
		"allowedPaths": apikeys.DefaultAllowedPaths,
		"bindingModes": []string{apikeys.BindingModeResources, apikeys.BindingModeEmployee},
	}, nil, nil)
	return nil
}
