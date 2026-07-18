package wework

import (
	"fmt"
	"strings"

	"github.com/openocta/openocta/pkg/channels"
)

// NewRuntimeFromConfig 基于 channels.wework 创建智能机器人（WebSocket）运行时。
//
// 配置示例：
//
//	{
//	  "enabled": true,
//	  "allowedIds": ["userid-1"],
//	  "credentials": {
//	    "botId": "从管理后台或扫码创建获得",
//	    "botSecret": "机器人 Secret",
//	    "wsUrl": "可选，默认 wss://openws.work.weixin.qq.com"
//	  }
//	}
func NewRuntimeFromConfig(raw map[string]interface{}, sink channels.InboundSink) (channels.RuntimeChannel, error) {
	if raw == nil {
		return nil, fmt.Errorf("wework: channels.wework not configured")
	}

	enabled := extractBoolWeWork(raw, "enabled", false)

	creds, ok := raw["credentials"].(map[string]interface{})
	if !ok || creds == nil {
		if !enabled {
			baseCfg := channels.BaseRuntimeConfig{
				Enabled:    false,
				AccountID:  "default",
				Name:       "WeWork",
				AllowedIDs: extractStringSliceWeWork(raw, "allowedIds"),
			}
			return NewRuntime("", "", "", baseCfg, sink), nil
		}
		return nil, fmt.Errorf("wework: credentials not found in config")
	}

	botID, _ := creds["botId"].(string)
	botSecret, _ := creds["botSecret"].(string)
	wsURL, _ := creds["wsUrl"].(string)
	if alt, ok := creds["wsURL"].(string); ok && strings.TrimSpace(alt) != "" {
		wsURL = alt
	}

	botID = strings.TrimSpace(botID)
	botSecret = strings.TrimSpace(botSecret)

	if enabled && (botID == "" || botSecret == "") {
		return nil, fmt.Errorf("wework: botId and botSecret are required when enabled")
	}

	baseCfg := channels.BaseRuntimeConfig{
		Enabled:    enabled,
		AccountID:  "default",
		Name:       "WeWork",
		AllowedIDs: extractStringSliceWeWork(raw, "allowedIds"),
	}

	return NewRuntime(botID, botSecret, wsURL, baseCfg, sink), nil
}

func extractBoolWeWork(m map[string]interface{}, key string, def bool) bool {
	if m == nil {
		return def
	}
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func extractStringSliceWeWork(m map[string]interface{}, key string) []string {
	if m == nil {
		return nil
	}
	raw, ok := m[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	default:
		return nil
	}
}
