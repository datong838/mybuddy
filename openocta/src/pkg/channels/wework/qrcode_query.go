package wework

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-sphere/wecom-aibot-go-sdk/aibot"
)

// QRStartSession requests a new enterprise WeCom scan session (generate).
func QRStartSession(ctx context.Context) (*aibot.QRCodeSession, error) {
	return aibot.FetchQRCodeSession(ctx, aibot.QRScanHTTPConfig{})
}

type qrQueryResponse struct {
	Data *struct {
		Status  string `json:"status"`
		BotInfo *struct {
			Botid  string `json:"botid"`
			Secret string `json:"secret"`
		} `json:"bot_info"`
	} `json:"data"`
}

// QRPollStatus performs a single query_result request (for UI polling).
func QRPollStatus(ctx context.Context, scode string) (status string, botID, secret string, err error) {
	if strings.TrimSpace(scode) == "" {
		return "", "", "", fmt.Errorf("wework qr: empty scode")
	}
	u, err := url.Parse("https://work.weixin.qq.com/ai/qc/query_result")
	if err != nil {
		return "", "", "", err
	}
	q := u.Query()
	q.Set("scode", scode)
	u.RawQuery = q.Encode()

	client := &http.Client{Timeout: 25 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", "", "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("wework qr: http %s: %s", resp.Status, truncateErrBody(body))
	}
	var parsed qrQueryResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", "", fmt.Errorf("wework qr: decode query_result: %w", err)
	}
	if parsed.Data == nil {
		return "", "", "", fmt.Errorf("wework qr: empty data in query_result")
	}
	st := strings.TrimSpace(parsed.Data.Status)
	if st == "success" && parsed.Data.BotInfo != nil {
		return st, strings.TrimSpace(parsed.Data.BotInfo.Botid), strings.TrimSpace(parsed.Data.BotInfo.Secret), nil
	}
	return st, "", "", nil
}

func truncateErrBody(b []byte) string {
	const max = 200
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}
