package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/appupdate"
)

type updateSkipRequest struct {
	Version string `json:"version"`
}

type updateInstallRequest struct {
	Version string `json:"version"`
}

func (s *Server) handleDesktopUpdateOptions(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization, X-Gateway-Token")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleDesktopUpdateCheck(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 GET"})
		return
	}

	force := queryTruthy(r, "force")
	record := queryTruthy(r, "record") || !force
	dailyOnly := queryTruthy(r, "dailyOnly")

	result := appupdate.Check(r.Context(), appupdate.CheckOptions{
		Force:     force,
		Record:    record,
		DailyOnly: dailyOnly,
		Now:       time.Now().UTC(),
	})

	status := appupdate.InstallStatus()
	out := map[string]interface{}{
		"ok":                   result.CheckError == "",
		"currentVersion":       result.CurrentVersion,
		"latestVersion":        result.LatestVersion,
		"hasUpdate":            result.HasUpdate,
		"skipped":              result.Skipped,
		"downloadSupported":    result.DownloadSupported,
		"downloadUrl":          result.DownloadURL,
		"autoInstallSupported": result.AutoInstallSupported,
		"installAllowed":       result.InstallAllowed,
		"packageFormat":        result.PackageFormat,
		"downloadUrls":         result.DownloadURLs,
		"manualInstallHint":    result.ManualInstallHint,
		"lastCheckAt":          result.LastCheckAt,
		"skippedDaily":         result.SkippedDaily,
		"desktopMode":          appupdate.DesktopMode(),
	}
	if result.CheckError != "" {
		out["message"] = result.CheckError
	}
	for k, v := range status {
		out[k] = v
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleDesktopUpdateSkip(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 POST"})
		return
	}

	var body updateSkipRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "请求体无效"})
		return
	}
	version := strings.TrimSpace(body.Version)
	if version == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "version 不能为空"})
		return
	}
	if err := appupdate.SkipVersion(version); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "version": version})
}

func (s *Server) handleDesktopUpdateInstall(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 POST"})
		return
	}
	if !appupdate.InstallAllowed() {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "当前运行模式不支持自动安装"})
		return
	}

	var body updateInstallRequest
	_ = json.NewDecoder(r.Body).Decode(&body)
	version := strings.TrimSpace(body.Version)
	if version == "" {
		check := appupdate.Check(context.Background(), appupdate.CheckOptions{Force: true, Record: false})
		version = strings.TrimSpace(check.LatestVersion)
	}
	if version == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "无法确定要安装的版本"})
		return
	}

	started, err := appupdate.StartInstallAsync(version)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"started": started,
		"status":  appupdate.InstallStatus(),
	})
}

func (s *Server) handleDesktopUpdateStatus(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 GET"})
		return
	}
	_ = json.NewEncoder(w).Encode(appupdate.InstallStatus())
}

func queryTruthy(r *http.Request, key string) bool {
	v := strings.TrimSpace(strings.ToLower(r.URL.Query().Get(key)))
	return v == "1" || v == "true" || v == "yes"
}
