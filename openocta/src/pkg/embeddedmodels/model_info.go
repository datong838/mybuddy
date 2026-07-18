package embeddedmodels

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const hfModelInfoTimeout = 15 * time.Second

// RepoFromHFURL extracts org/repo from a HuggingFace page or resolve URL.
func RepoFromHFURL(raw string) string {
	repo, err := ExtractHFRepo(raw)
	if err != nil {
		return ""
	}
	return repo
}

type HFModelInfo struct {
	ID           string   `json:"id"`
	ModelID      string   `json:"modelId,omitempty"`
	Author       string   `json:"author,omitempty"`
	Sha          string   `json:"sha,omitempty"`
	LastModified string   `json:"lastModified,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	PipelineTag  string   `json:"pipeline_tag,omitempty"`
	Downloads    int      `json:"downloads,omitempty"`
	Likes        int      `json:"likes,omitempty"`
	LibraryName  string   `json:"library_name,omitempty"`
	CardData     *struct {
		Language        []string `json:"language,omitempty"`
		License         string   `json:"license,omitempty"`
		LicenseName     string   `json:"license_name,omitempty"`
		LicenseLink     string   `json:"license_link,omitempty"`
		BaseModel       string   `json:"base_model,omitempty"`
		Datasets        []string `json:"datasets,omitempty"`
		LanguageDetails string   `json:"language_creators,omitempty"`
	} `json:"cardData,omitempty"`
}

// PlazaModelDetail aggregates remote metadata for the model plaza detail view.
type PlazaModelDetail struct {
	Repo        string   `json:"repo"`
	Description string   `json:"description,omitempty"`
	Readme      string   `json:"readme,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Downloads   int      `json:"downloads,omitempty"`
	Likes       int      `json:"likes,omitempty"`
	PipelineTag string   `json:"pipelineTag,omitempty"`
	License     string   `json:"license,omitempty"`
	Error       string   `json:"error,omitempty"`
}

func fetchHFJSON(env func(string) string, path string, dest any) error {
	base := strings.TrimRight(hfAPIBase(env), "/")
	url := base + path
	client := &http.Client{Timeout: hfModelInfoTimeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "OpenOcta/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HF API %s: HTTP %d %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func fetchHFText(env func(string) string, url string) (string, error) {
	client := &http.Client{Timeout: hfModelInfoTimeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "OpenOcta/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func tryFetchReadme(env func(string) string, repo string) string {
	base := strings.TrimRight(hfAPIBase(env), "/")
	candidates := []string{
		base + "/" + repo + "/raw/main/README.md",
		base + "/" + repo + "/resolve/main/README.md",
	}
	for _, u := range candidates {
		text, err := fetchHFText(env, u)
		if err == nil && strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
}

// FetchPlazaModelDetail loads HuggingFace card + README for a repo id (org/name).
func FetchPlazaModelDetail(env func(string) string, repo string) PlazaModelDetail {
	repo = strings.Trim(strings.TrimSpace(repo), "/")
	out := PlazaModelDetail{Repo: repo}
	if repo == "" || !strings.Contains(repo, "/") {
		out.Error = "无效的 HuggingFace 仓库"
		return out
	}

	var info HFModelInfo
	if err := fetchHFJSON(env, "/api/models/"+repo, &info); err != nil {
		out.Error = err.Error()
		return out
	}

	out.Tags = info.Tags
	out.Downloads = info.Downloads
	out.Likes = info.Likes
	out.PipelineTag = info.PipelineTag
	if info.CardData != nil {
		if info.CardData.LicenseName != "" {
			out.License = info.CardData.LicenseName
		} else if info.CardData.License != "" {
			out.License = info.CardData.License
		}
	}

	readme := tryFetchReadme(env, repo)
	if readme != "" {
		out.Readme = readme
		// Use first paragraph of README as description fallback
		if desc := extractReadmeSummary(readme); desc != "" {
			out.Description = desc
		}
	}

	return out
}

func extractReadmeSummary(readme string) string {
	lines := strings.Split(readme, "\n")
	var parts []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "---") {
			if len(parts) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(trimmed, "<") || strings.HasPrefix(trimmed, "!") || strings.HasPrefix(trimmed, "[") {
			continue
		}
		parts = append(parts, trimmed)
		if len(strings.Join(parts, " ")) > 120 {
			break
		}
	}
	summary := strings.Join(parts, " ")
	if len(summary) > 280 {
		return summary[:277] + "..."
	}
	return summary
}
