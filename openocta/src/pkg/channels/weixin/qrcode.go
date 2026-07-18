package weixin

import (
	"context"
	"fmt"
	"strings"

	ilink "github.com/openilink/openilink-sdk-go"
)

// qrClient builds an iLink client for pre-login QR APIs (no token).
func qrClient(baseURL, botType, routeTag string) *ilink.Client {
	baseURL = strings.TrimSpace(baseURL)
	botType = strings.TrimSpace(botType)
	routeTag = strings.TrimSpace(routeTag)
	opts := []ilink.Option{}
	if baseURL != "" {
		opts = append(opts, ilink.WithBaseURL(baseURL))
	}
	if botType != "" {
		opts = append(opts, ilink.WithBotType(botType))
	}
	if routeTag != "" {
		opts = append(opts, ilink.WithRouteTag(routeTag))
	}
	return ilink.NewClient("", opts...)
}

// QRStartSession requests a new personal WeChat (iLink) login QR session.
func QRStartSession(ctx context.Context, baseURL, botType, routeTag string) (qrcode, qrImageContent string, err error) {
	c := qrClient(baseURL, botType, routeTag)
	resp, err := c.FetchQRCode(ctx)
	if err != nil {
		return "", "", err
	}
	return strings.TrimSpace(resp.QRCode), strings.TrimSpace(resp.QRCodeImgContent), nil
}

// QRPollStatus polls scan status once (long-poll friendly). On confirmed, returns token and bot identity fields.
func QRPollStatus(ctx context.Context, baseURL, botType, routeTag, qrcode string) (
	status string,
	botToken, botID, baseURLOut, userID string,
	err error,
) {
	qrcode = strings.TrimSpace(qrcode)
	if qrcode == "" {
		return "", "", "", "", "", fmt.Errorf("weixin qr: empty qrcode")
	}
	c := qrClient(baseURL, botType, routeTag)
	st, err := c.PollQRStatus(ctx, qrcode)
	if err != nil {
		return "", "", "", "", "", err
	}
	status = strings.TrimSpace(st.Status)
	switch status {
	case "confirmed":
		return status, strings.TrimSpace(st.BotToken), strings.TrimSpace(st.ILinkBotID),
			strings.TrimSpace(st.BaseURL), strings.TrimSpace(st.ILinkUserID), nil
	default:
		return status, "", "", "", "", nil
	}
}
