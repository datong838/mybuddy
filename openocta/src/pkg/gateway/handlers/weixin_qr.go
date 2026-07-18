package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/channels/weixin"
	"github.com/openocta/openocta/pkg/gateway/protocol"
)

// WeixinQRStartHandler handles "channels.weixin.qr.start" — fetch iLink QR (qrcode + image content).
func WeixinQRStartHandler(opts HandlerOpts) error {
	if opts.Params == nil {
		opts.Params = map[string]interface{}{}
	}
	baseURL, _ := opts.Params["baseUrl"].(string)
	if alt, ok := opts.Params["baseURL"].(string); ok && strings.TrimSpace(alt) != "" {
		baseURL = alt
	}
	botType, _ := opts.Params["botType"].(string)
	routeTag, _ := opts.Params["routeTag"].(string)

	reqCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	qrcode, img, err := weixin.QRStartSession(reqCtx, baseURL, botType, routeTag)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeServiceUnavailable,
			Message: err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, map[string]interface{}{
		"qrcode":         qrcode,
		"qrImageContent": img,
		"baseUrl":        strings.TrimSpace(baseURL),
		"botType":        strings.TrimSpace(botType),
	}, nil, nil)
	return nil
}

// WeixinQRPollHandler handles "channels.weixin.qr.poll" — single PollQRStatus for UI.
func WeixinQRPollHandler(opts HandlerOpts) error {
	params := opts.Params
	if params == nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "missing params",
		}, nil)
		return nil
	}
	qrcode, _ := params["qrcode"].(string)
	baseURL, _ := params["baseUrl"].(string)
	if alt, ok := params["baseURL"].(string); ok && strings.TrimSpace(alt) != "" {
		baseURL = alt
	}
	botType, _ := params["botType"].(string)
	routeTag, _ := params["routeTag"].(string)

	reqCtx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	status, botToken, botID, baseOut, userID, err := weixin.QRPollStatus(reqCtx, baseURL, botType, routeTag, qrcode)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeServiceUnavailable,
			Message: err.Error(),
		}, nil)
		return nil
	}
	out := map[string]interface{}{
		"status": strings.TrimSpace(status),
	}
	if status == "confirmed" {
		out["botToken"] = botToken
		out["botId"] = botID
		if baseOut != "" {
			out["baseUrl"] = baseOut
		}
		out["userId"] = userID
	}
	opts.Respond(true, out, nil, nil)
	return nil
}
