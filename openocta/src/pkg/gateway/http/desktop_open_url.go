package http

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/pkg/browser"
)

type desktopOpenURLRequest struct {
	URL string `json:"url"`
}

type desktopOpenURLResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

// handleDesktopOpenURLOptions handles CORS preflight for POST /api/desktop/open-url.
func (s *Server) handleDesktopOpenURLOptions(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization, X-Gateway-Token")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// handleDesktopOpenURL opens the given http(s) URL in the system default browser.
// Requires gateway token. Only when OPENOCTA_RUN_MODE=desktop.
func (s *Server) handleDesktopOpenURL(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization, X-Gateway-Token")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(desktopOpenURLResponse{OK: false, Message: "仅支持 POST"})
		return
	}

	if strings.TrimSpace(os.Getenv("OPENOCTA_RUN_MODE")) != "desktop" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(desktopOpenURLResponse{
			OK:      false,
			Message: "仅桌面应用可使用此接口",
			Detail:  "在浏览器中请使用普通链接打开",
		})
		return
	}

	var body desktopOpenURLRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(desktopOpenURLResponse{OK: false, Message: "请求体无效", Detail: err.Error()})
		return
	}
	raw := strings.TrimSpace(body.URL)
	if raw == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(desktopOpenURLResponse{OK: false, Message: "缺少 url"})
		return
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(desktopOpenURLResponse{OK: false, Message: "无效的 URL"})
		return
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(desktopOpenURLResponse{OK: false, Message: "仅允许 http(s) URL"})
		return
	}

	if err := browser.OpenURL(u.String()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(desktopOpenURLResponse{
			OK:      false,
			Message: "无法在系统浏览器中打开链接",
			Detail:  err.Error(),
		})
		return
	}
	_ = json.NewEncoder(w).Encode(desktopOpenURLResponse{OK: true, Message: "ok"})
}
