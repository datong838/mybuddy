package weixin

import "github.com/openocta/openocta/pkg/channels"

const channelID = "weixin"

type weixinGatewayPlugin struct {
	*channels.BasePlugin
}

func (p *weixinGatewayPlugin) LogoutAccount(_ *channels.LogoutContext) (*channels.LogoutResult, error) {
	return &channels.LogoutResult{Cleared: true, LoggedOut: true}, nil
}

// Plugin is the personal WeChat (个人微信 iLink) channel metadata.
var Plugin = &weixinGatewayPlugin{
	BasePlugin: &channels.BasePlugin{
		Id: channelID,
		MetaData: channels.ChannelMeta{
			ID:             channelID,
			Label:          "微信（个人）",
			SelectionLabel: "微信（个人 iLink 扫码）",
			DocsPath:       "/channels/weixin",
			DocsLabel:      "weixin",
			Blurb:          "个人微信 iLink Bot：HTTPS 长轮询收发消息，扫码登录获取 botToken / botId。",
			SystemImage:    "message-square",
			Order:          55,
		},
	},
}
