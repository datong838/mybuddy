package wework

import "github.com/openocta/openocta/pkg/channels"

const channelID = "wework"

// Plugin is the WeWork (企业微信) channel plugin metadata.
var Plugin = &channels.BasePlugin{
	Id: channelID,
	MetaData: channels.ChannelMeta{
		ID:             channelID,
		Label:          "微信",
		SelectionLabel: "微信（企业智能机器人）",
		DocsPath:       "/channels/wework",
		DocsLabel:      "wework",
		Blurb:          "企业微信智能机器人：WebSocket 长连接（aibot），支持扫码创建与凭据配置。",
		SystemImage:    "message-square",
		Order:          50,
	},
}
