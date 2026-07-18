package embeddedmodels

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func proxyToRuntime(modelID, path string, body []byte) (statusCode int, respBody []byte, contentType string, err error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return 0, nil, "", fmt.Errorf("model 不能为空")
	}

	rt, ok := getRuntime(modelID)
	if !ok {
		return 0, nil, "", fmt.Errorf("模型 %s 未在运行", modelID)
	}

	rt.mu.Lock()
	running := rt.server != nil
	port := rt.port
	rt.mu.Unlock()

	if !running {
		return 0, nil, "", fmt.Errorf("模型 %s 未在运行", modelID)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/v1/%s", port, strings.TrimPrefix(path, "/"))
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Minute}
	res, err := client.Do(req)
	if err != nil {
		return 0, nil, "", fmt.Errorf("转发请求失败: %w", err)
	}
	defer res.Body.Close()

	respBody, err = io.ReadAll(res.Body)
	if err != nil {
		return 0, nil, "", fmt.Errorf("读取模型响应失败: %w", err)
	}
	contentType = res.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json; charset=utf-8"
	}
	return res.StatusCode, respBody, contentType, nil
}

// ProxyChatCompletion forwards a chat completion request to a running embedded model.
func ProxyChatCompletion(modelID string, body []byte) (statusCode int, respBody []byte, contentType string, err error) {
	return proxyToRuntime(modelID, "chat/completions", body)
}

// ProxyEmbeddings forwards an embeddings request to a running embedded model.
func ProxyEmbeddings(modelID string, body []byte) (statusCode int, respBody []byte, contentType string, err error) {
	return proxyToRuntime(modelID, "embeddings", body)
}
