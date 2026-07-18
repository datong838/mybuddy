package channels

// BasePlugin is a minimal ChannelPlugin implementation for built-in channels.
type BasePlugin struct {
	Id                 ChannelId
	MetaData           ChannelMeta
	GatewayMethodsList []string
}

// ID returns the channel ID.
func (p *BasePlugin) ID() ChannelId {
	return p.Id
}

// Meta returns channel metadata.
func (p *BasePlugin) Meta() ChannelMeta {
	return p.MetaData
}

// GatewayMethods returns additional gateway methods for this channel.
func (p *BasePlugin) GatewayMethods() []string {
	if p.GatewayMethodsList == nil {
		return []string{}
	}
	return p.GatewayMethodsList
}
