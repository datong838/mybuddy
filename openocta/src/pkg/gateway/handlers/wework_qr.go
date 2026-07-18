package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/go-sphere/wecom-aibot-go-sdk/aibot"
	"github.com/openocta/openocta/pkg/channels/wework"
	"github.com/openocta/openocta/pkg/gateway/protocol"
)

// WeWorkQRStartHandler handles "channels.wework.qr.start" — fetch scan session (auth_url / scode).
func WeWorkQRStartHandler(opts HandlerOpts) error {
	if opts.Context == nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "context not configured",
		}, nil)
		return nil
	}
	reqCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	sess, err := wework.QRStartSession(reqCtx)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeServiceUnavailable,
			Message: err.Error(),
		}, nil)
		return nil
	}
	genPage := aibot.QRGenPageURLForSession(aibot.QRScanHTTPConfig{}, sess.Scode)
	opts.Respond(true, map[string]interface{}{
		"scode":      sess.Scode,
		"authUrl":    sess.AuthURL,
		"genPageUrl": genPage,
	}, nil, nil)
	return nil
}

// WeWorkQRPollHandler handles "channels.wework.qr.poll" — single query_result poll for UI.
func WeWorkQRPollHandler(opts HandlerOpts) error {
	params := opts.Params
	if params == nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "missing params",
		}, nil)
		return nil
	}
	scode, _ := params["scode"].(string)
	scode = strings.TrimSpace(scode)
	if scode == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "scode required",
		}, nil)
		return nil
	}
	reqCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status, botID, secret, err := wework.QRPollStatus(reqCtx, scode)
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
	if status == "success" {
		out["botId"] = botID
		out["botSecret"] = secret
	}
	opts.Respond(true, out, nil, nil)
	return nil
}
