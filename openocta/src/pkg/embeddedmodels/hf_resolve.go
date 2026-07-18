package embeddedmodels

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var hfRepoRE = regexp.MustCompile(`(?i)huggingface\.co/([^/]+/[^/?#]+)`)

var ggufBlockTokens = []string{
	"mmproj", "clip", "projector", "vision", "image",
	"encoder", "embed", "embedding", "rerank", "tei", "vocoder",
}

type hfTreeEntry struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	LFS  *struct {
		Size int64 `json:"size"`
	} `json:"lfs"`
}

type ggufCandidate struct {
	name string
	size int64
}

// ExtractHFRepo returns "org/repo" from a HuggingFace model or file URL.
func ExtractHFRepo(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty HuggingFace URL")
	}
	m := hfRepoRE.FindStringSubmatch(raw)
	if len(m) < 2 {
		return "", fmt.Errorf("无法解析 HuggingFace 仓库地址: %s", raw)
	}
	return strings.TrimSuffix(m[1], "/"), nil
}

func isMainModelGGUF(filename string) bool {
	lower := strings.ToLower(filename)
	if !strings.HasSuffix(lower, ".gguf") {
		return false
	}
	for _, token := range ggufBlockTokens {
		if strings.Contains(lower, token) {
			return false
		}
	}
	return true
}

func stripNonAlnum(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func matchQuantFile(files []ggufCandidate, quant string) (ggufCandidate, bool) {
	quantLower := strings.ToLower(strings.TrimSpace(quant))
	if quantLower == "" {
		quantLower = "q4_k_m"
	}
	quantStripped := stripNonAlnum(quant)

	type scored struct {
		candidate ggufCandidate
		score     int
	}
	var best *scored

	for _, file := range files {
		lower := strings.ToLower(file.name)
		stripped := stripNonAlnum(file.name)
		score := 0
		switch {
		case strings.Contains(lower, quantLower):
			score = 100
		case strings.Contains(stripped, quantStripped):
			score = 80
		default:
			continue
		}
		if best == nil || score > best.score || (score == best.score && len(file.name) < len(best.candidate.name)) {
			best = &scored{candidate: file, score: score}
		}
	}
	if best == nil {
		return ggufCandidate{}, false
	}
	return best.candidate, true
}

func hfAPIBase(env func(string) string) string {
	mirror := resolveHFMirror(env)
	if mirror != "" {
		return mirror
	}
	return "https://huggingface.co"
}

func listRepoGGUFFiles(repo string, env func(string) string) ([]ggufCandidate, error) {
	apiURL := fmt.Sprintf("%s/api/models/%s/tree/main", strings.TrimRight(hfAPIBase(env), "/"), repo)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "openocta-embedded-models/1.0")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("查询 HuggingFace 文件列表失败: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HuggingFace API HTTP %d", res.StatusCode)
	}

	var entries []hfTreeEntry
	if err := json.NewDecoder(res.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("解析 HuggingFace 文件列表失败: %w", err)
	}

	out := make([]ggufCandidate, 0, len(entries))
	for _, entry := range entries {
		if entry.Type != "file" || !isMainModelGGUF(entry.Path) {
			continue
		}
		name := entry.Path
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		size := entry.Size
		if entry.LFS != nil && entry.LFS.Size > 0 {
			size = entry.LFS.Size
		}
		if name == "" {
			continue
		}
		out = append(out, ggufCandidate{name: name, size: size})
	}
	return out, nil
}

// ResolveGGUFFile picks a GGUF artifact from a HuggingFace repo page URL.
func ResolveGGUFFile(hfURL, quantization string, env func(string) string) (ModelFile, error) {
	hfURL = strings.TrimSpace(hfURL)
	if strings.Contains(hfURL, "/resolve/") {
		u, err := url.Parse(hfURL)
		if err != nil {
			return ModelFile{}, err
		}
		name := u.Path
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		return ModelFile{Name: name, URL: hfURL}, nil
	}

	repo, err := ExtractHFRepo(hfURL)
	if err != nil {
		return ModelFile{}, err
	}
	files, err := listRepoGGUFFiles(repo, env)
	if err != nil {
		return ModelFile{}, err
	}
	if len(files) == 0 {
		return ModelFile{}, fmt.Errorf("HuggingFace 仓库 %s 中未找到 GGUF 权重，请打开模型详情中的 GGUF 链接或稍后重试", repo)
	}

	chosen, ok := matchQuantFile(files, quantization)
	if !ok {
		// Fallback to the smallest main GGUF when quant name doesn't match.
		chosen = files[0]
		for _, f := range files[1:] {
			if f.size > 0 && (chosen.size == 0 || f.size < chosen.size) {
				chosen = f
			}
		}
	}

	downloadURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", repo, chosen.name)
	return ModelFile{
		Name: chosen.name,
		URL:  downloadURL,
		Size: chosen.size,
	}, nil
}

func ensureDownloadFiles(entry CatalogEntry, env func(string) string) (CatalogEntry, error) {
	if len(entry.Files) > 0 {
		return entry, nil
	}
	if !entry.Downloadable || strings.TrimSpace(entry.HfURL) == "" {
		return entry, fmt.Errorf("模型 %s 暂未配置内嵌下载", entry.ID)
	}
	file, err := ResolveGGUFFile(entry.HfURL, entry.Quantization, env)
	if err != nil {
		return entry, err
	}
	entry.Files = []ModelFile{file}
	return entry, nil
}
