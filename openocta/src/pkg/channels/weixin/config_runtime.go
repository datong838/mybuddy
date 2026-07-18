package weixin

import (
	"fmt"
	"strings"

	"github.com/openocta/openocta/pkg/channels"
)

// NewRuntimeFromConfig 基于 channels.weixin 创建个人微信（iLink 长轮询）运行时。
//
// 配置示例：
//
//	{
//	  "enabled": true,
//	  "allowedIds": ["ilink-user-id-1"],
//	  "credentials": {
//	    "botToken": "扫码登录后的 Bot Token",
//	    "botId": "iLink Bot ID",
//	    "baseUrl": "可选，默认 https://ilinkai.weixin.qq.com",
//	    "botType": "可选，默认 3",
//	    "routeTag": "可选 SKRouteTag",
//	    "getUpdatesBuf": "可选，同步游标（与 openilink Monitor 一致）"
//	  }
//	}
func NewRuntimeFromConfig(raw map[string]interface{}, sink channels.InboundSink) (channels.RuntimeChannel, error) {
	if raw == nil {
		return nil, fmt.Errorf("weixin: channels.weixin not configured")
	}

	enabled := extractBool(raw, "enabled", false)

	creds, ok := raw["credentials"].(map[string]interface{})
	if !ok || creds == nil {
		if !enabled {
			baseCfg := channels.BaseRuntimeConfig{
				Enabled:    false,
				AccountID:  "default",
				Name:       "Weixin",
				AllowedIDs: extractStringSlice(raw, "allowedIds"),
			}
			return NewRuntime("", "", "", "", "", "", "", baseCfg, sink), nil
		}
		return nil, fmt.Errorf("weixin: credentials not found in config")
	}

	botToken, _ := creds["botToken"].(string)
	botID, _ := creds["botId"].(string)
	baseURL, _ := creds["baseUrl"].(string)
	if alt, ok := creds["baseURL"].(string); ok && strings.TrimSpace(alt) != "" {
		baseURL = alt
	}
	botType, _ := creds["botType"].(string)
	routeTag, _ := creds["routeTag"].(string)
	syncBuf, _ := creds["getUpdatesBuf"].(string)
	if alt, ok := creds["syncBuf"].(string); ok && strings.TrimSpace(alt) != "" {
		syncBuf = alt
	}
	userID, _ := creds["userId"].(string)
	if alt, ok := creds["ilinkUserId"].(string); ok && strings.TrimSpace(alt) != "" {
		userID = alt
	}

	botToken = strings.TrimSpace(botToken)
	botID = strings.TrimSpace(botID)

	if enabled && (botToken == "" || botID == "") {
		return nil, fmt.Errorf("weixin: botToken and botId are required when enabled")
	}

	baseCfg := channels.BaseRuntimeConfig{
		Enabled:    enabled,
		AccountID:  "default",
		Name:       "Weixin",
		AllowedIDs: extractStringSlice(raw, "allowedIds"),
	}

	return NewRuntime(botToken, botID, baseURL, botType, routeTag, syncBuf, strings.TrimSpace(userID), baseCfg, sink), nil
}

func extractBool(m map[string]interface{}, key string, def bool) bool {
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

func extractStringSlice(m map[string]interface{}, key string) []string {
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
