package embeddedmodels

import (
	"net/url"
	"strings"
)

const (
	defaultHFMirror    = "https://hf-mirror.com"
	defaultGitHubProxy = "https://gh-proxy.com"
)

// GitHubDownloadURLs returns candidate download URLs for a GitHub release asset.
// By default uses gh-proxy.com only (no direct GitHub fallback).
// Set OPENOCTA_GITHUB_PROXY=off to use only the original URL; a custom value overrides the default.
func GitHubDownloadURLs(raw string, env func(string) string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	proxies := resolveGitHubProxies(env)
	if len(proxies) == 0 {
		return []string{raw}
	}
	out := make([]string, 0, len(proxies))
	seen := make(map[string]struct{}, len(proxies))
	for _, proxy := range proxies {
		u := rewriteGitHubProxy(raw, proxy)
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}

// ApplyGitHubProxy rewrites github.com release/download URLs through the first configured proxy.
func ApplyGitHubProxy(raw string, env func(string) string) string {
	urls := GitHubDownloadURLs(raw, env)
	if len(urls) == 0 {
		return raw
	}
	return urls[0]
}

func resolveGitHubProxies(env func(string) string) []string {
	if env != nil {
		if v := strings.TrimSpace(env("OPENOCTA_GITHUB_PROXY")); v != "" {
			if strings.EqualFold(v, "off") || strings.EqualFold(v, "false") || v == "0" {
				return nil
			}
			part := strings.TrimRight(strings.TrimSpace(v), "/")
			if part != "" {
				return []string{part}
			}
			return nil
		}
	}
	return []string{defaultGitHubProxy}
}

func rewriteGitHubProxy(raw, proxy string) string {
	raw = strings.TrimSpace(raw)
	proxy = strings.TrimRight(strings.TrimSpace(proxy), "/")
	if raw == "" || proxy == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	host := strings.ToLower(u.Host)
	if host != "github.com" && !strings.HasSuffix(host, ".github.com") {
		return raw
	}
	return proxy + "/" + raw
}

// DownloadURLs returns the download URL for a catalog file.
// By default rewrites huggingface.co links to hf-mirror.com (faster in China).
// Set OPENOCTA_HF_MIRROR=off to use the catalog URL as-is; HF_ENDPOINT overrides the mirror host.
func DownloadURLs(catalogURL string, env func(string) string) []string {
	if catalogURL == "" {
		return nil
	}
	mirror := resolveHFMirror(env)
	if mirror != "" {
		if mirrored := rewriteHFMirror(catalogURL, mirror); mirrored != "" {
			return []string{mirrored}
		}
	}
	return []string{catalogURL}
}

func resolveHFMirror(env func(string) string) string {
	if env != nil {
		if v := strings.TrimSpace(env("OPENOCTA_HF_MIRROR")); v != "" {
			if strings.EqualFold(v, "off") || strings.EqualFold(v, "false") || v == "0" {
				return ""
			}
			return strings.TrimRight(v, "/")
		}
		if v := strings.TrimSpace(env("HF_ENDPOINT")); v != "" {
			return strings.TrimRight(v, "/")
		}
	}
	return defaultHFMirror
}

func rewriteHFMirror(raw, mirror string) string {
	raw = strings.TrimSpace(raw)
	mirror = strings.TrimRight(strings.TrimSpace(mirror), "/")
	if raw == "" || mirror == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	host := strings.ToLower(u.Host)
	if host != "huggingface.co" && host != "www.huggingface.co" && !strings.HasSuffix(host, ".huggingface.co") {
		return raw
	}
	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 5 || parts[2] != "resolve" {
		return mirror + "/" + path
	}
	org, repo, rev := parts[0], parts[1], parts[3]
	filePath := strings.Join(parts[4:], "/")
	if filePath == "" {
		return mirror + "/" + path
	}
	return mirror + "/" + org + "/" + repo + "/resolve/" + rev + "/" + filePath
}
