// Package pluginsdk defines types for OpenClaw plugin extensions.
package pluginsdk

// ChannelConfigSchema describes a channel's config schema.
type ChannelConfigSchema struct {
	Schema  map[string]interface{}       `json:"schema"`
	UIHints map[string]ChannelConfigHint `json:"uiHints,omitempty"`
}

// ChannelConfigHint provides UI hints for a config field.
type ChannelConfigHint struct {
	Label       string `json:"label,omitempty"`
	Help        string `json:"help,omitempty"`
	Advanced    bool   `json:"advanced,omitempty"`
	Sensitive   bool   `json:"sensitive,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
}

// GatewayRequestHandler processes a Gateway WebSocket request.
type GatewayRequestHandler func(req GatewayRequest) error

// GatewayRequest represents a Gateway method request.
type GatewayRequest struct {
	ID     string
	Method string
	Params map[string]interface{}
	// Respond sends the response
	Respond func(ok bool, payload interface{}, err error)
}

// GatewayMethods returns additional Gateway methods a plugin provides.
type GatewayMethods []string
