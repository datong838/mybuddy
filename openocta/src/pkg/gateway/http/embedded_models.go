package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/openocta/openocta/pkg/embeddedmodels"
)

func (s *Server) handleEmbeddedModelsOptions(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization, X-Gateway-Token")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleEmbeddedModelsModelInfo(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 GET"})
		return
	}
	repo := strings.TrimSpace(r.URL.Query().Get("repo"))
	if repo == "" {
		hfURL := strings.TrimSpace(r.URL.Query().Get("hfUrl"))
		repo = embeddedmodels.RepoFromHFURL(hfURL)
	}
	if repo == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "repo 或 hfUrl 不能为空"})
		return
	}
	detail := embeddedmodels.FetchPlazaModelDetail(os.Getenv, repo)
	if detail.Error != "" && detail.Readme == "" && detail.Description == "" {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      false,
			"message": detail.Error,
			"repo":    detail.Repo,
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":          true,
		"repo":        detail.Repo,
		"description": detail.Description,
		"readme":      detail.Readme,
		"tags":        detail.Tags,
		"downloads":   detail.Downloads,
		"likes":       detail.Likes,
		"pipelineTag": detail.PipelineTag,
		"license":     detail.License,
		"error":       detail.Error,
	})
}

func (s *Server) handleEmbeddedModelsRecommendations(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 GET 或 POST"})
		return
	}

	var override *embeddedmodels.HardwareProfile
	if r.Method == http.MethodPost {
		var body embeddedmodels.HardwareProfile
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			override = &body
		}
	} else {
		q := r.URL.Query()
		if hasAnyHardwareQuery(q) {
			override = &embeddedmodels.HardwareProfile{}
			if v := strings.TrimSpace(q.Get("gpuName")); v != "" {
				override.GPUName = v
			}
			if v, ok := parseOptionalFloat(q.Get("vramGb")); ok {
				override.VramGb = v
			}
			if v, ok := parseOptionalFloat(q.Get("ramGb")); ok {
				override.RamGb = v
			}
			if v, ok := parseOptionalInt(q.Get("cpuCores")); ok {
				override.CPUCores = v
			}
			if v, ok := parseOptionalFloat(q.Get("bandwidthGbs")); ok {
				override.BandwidthGbs = v
			}
			if strings.EqualFold(strings.TrimSpace(q.Get("isAppleSilicon")), "true") ||
				q.Get("isAppleSilicon") == "1" {
				override.IsAppleSilicon = true
			}
		}
	}

	result := embeddedmodels.BuildRecommendations(os.Getenv, override)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":              true,
		"hardware":        result.Hardware,
		"recommendations": result.Recommendations,
	})
}

func hasAnyHardwareQuery(q url.Values) bool {
	for _, key := range []string{"gpuName", "vramGb", "ramGb", "cpuCores", "bandwidthGbs", "isAppleSilicon"} {
		if strings.TrimSpace(q.Get(key)) != "" {
			return true
		}
	}
	return false
}

func parseOptionalFloat(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

func parseOptionalInt(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

func (s *Server) handleEmbeddedModelsCatalog(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 GET"})
		return
	}
	env := os.Getenv
	runtime := embeddedmodels.RuntimeStatus(env)
	if running, _ := runtime["running"].(bool); running {
		_ = embeddedmodels.PersistMergedProviderConfig(env, 0)
		runtime = embeddedmodels.RuntimeStatus(env)
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":       true,
		"models":   embeddedmodels.ListCatalog(env),
		"runtime":  runtime,
		"download": embeddedmodels.DownloadStatus(),
		"provider": embeddedmodels.MergedProviderConfig(env, 0),
	})
}

func (s *Server) handleEmbeddedModelsDownloadStart(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 POST"})
		return
	}
	var body struct {
		ModelID string `json:"modelId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	modelID := strings.TrimSpace(body.ModelID)
	if modelID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "modelId 不能为空"})
		return
	}
	started, err := embeddedmodels.StartDownloadAsync(os.Getenv, modelID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":       true,
		"started":  started,
		"download": embeddedmodels.DownloadStatus(),
	})
}

func (s *Server) handleEmbeddedModelsDownloadStatus(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 GET"})
		return
	}
	_ = json.NewEncoder(w).Encode(embeddedmodels.DownloadStatus())
}

func (s *Server) handleEmbeddedModelsDownloadCancel(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 POST"})
		return
	}
	cancelled := embeddedmodels.CancelDownload()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":        true,
		"cancelled": cancelled,
		"download":  embeddedmodels.DownloadStatus(),
	})
}

func (s *Server) handleEmbeddedModelsStart(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 POST"})
		return
	}
	var body struct {
		ModelID string `json:"modelId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	modelID := strings.TrimSpace(body.ModelID)
	if modelID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "modelId 不能为空"})
		return
	}
	port, err := embeddedmodels.StartModel(os.Getenv, modelID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": err.Error()})
		return
	}
	_ = embeddedmodels.PersistMergedProviderConfig(os.Getenv, 0)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":       true,
		"modelId":  modelID,
		"port":     port,
		"endpoint": fmt.Sprintf("http://127.0.0.1:%d/v1", port),
		"provider": embeddedmodels.MergedProviderConfig(os.Getenv, 0),
		"runtime":  embeddedmodels.RuntimeStatus(os.Getenv),
	})
}

func (s *Server) handleEmbeddedModelsStop(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 POST"})
		return
	}
	var body struct {
		ModelID string `json:"modelId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	modelID := strings.TrimSpace(body.ModelID)
	if err := embeddedmodels.StopModel(os.Getenv, modelID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": err.Error()})
		return
	}
	_ = embeddedmodels.PersistMergedProviderConfig(os.Getenv, 0)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":       true,
		"provider": embeddedmodels.MergedProviderConfig(os.Getenv, 0),
		"runtime":  embeddedmodels.RuntimeStatus(os.Getenv),
	})
}

func (s *Server) handleEmbeddedModelsDelete(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 POST"})
		return
	}
	var body struct {
		ModelID string `json:"modelId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	modelID := strings.TrimSpace(body.ModelID)
	if modelID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "modelId 不能为空"})
		return
	}
	if err := embeddedmodels.DeleteModel(os.Getenv, modelID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":     true,
		"models": embeddedmodels.ListCatalog(os.Getenv),
	})
}

func (s *Server) handleEmbeddedModelsChatCompletions(w http.ResponseWriter, r *http.Request) {
	s.handleEmbeddedModelsProxy(w, r, embeddedmodels.ProxyChatCompletion)
}

func (s *Server) handleEmbeddedModelsEmbeddings(w http.ResponseWriter, r *http.Request) {
	s.handleEmbeddedModelsProxy(w, r, embeddedmodels.ProxyEmbeddings)
}

func (s *Server) handleEmbeddedModelsProxy(
	w http.ResponseWriter,
	r *http.Request,
	proxy func(modelID string, body []byte) (statusCode int, respBody []byte, contentType string, err error),
) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization, X-Gateway-Token")
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "仅支持 POST"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "读取请求体失败"})
		return
	}

	var payload struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal(body, &payload)
	modelID := strings.TrimSpace(payload.Model)
	if modelID == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": "model 不能为空"})
		return
	}

	statusCode, respBody, contentType, err := proxy(modelID, body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": err.Error()})
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)
	_, _ = w.Write(respBody)
}
