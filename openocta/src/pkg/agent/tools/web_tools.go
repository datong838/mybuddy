package tools

import (
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/agent/tool"
	"github.com/openocta/openocta/pkg/config"
)

const (
	defaultWebUserAgent     = "Mozilla/5.0 (compatible; OpenOcta/1.0)"
	defaultSearchMaxResults = 5
	defaultFetchMaxChars    = 12000
	defaultHTTPTimeout      = 25 * time.Second
)

// WebToolNames lists built-in network tools.
var WebToolNames = []string{"web_search", "web_fetch"}

// IsWebToolName reports whether name is a built-in web tool.
func IsWebToolName(name string) bool {
	key := strings.ToLower(strings.TrimSpace(name))
	for _, n := range WebToolNames {
		if key == n {
			return true
		}
	}
	return false
}

// FilterOutWebTools removes web_search and web_fetch from a tool list.
func FilterOutWebTools(tools []tool.Tool) []tool.Tool {
	if len(tools) == 0 {
		return tools
	}
	out := make([]tool.Tool, 0, len(tools))
	for _, t := range tools {
		if t == nil || IsWebToolName(t.Name()) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// webToolsEnabled is a kill-switch while web_search / web_fetch are temporarily removed.
const webToolsEnabled = false

// WebToolsFromConfig returns enabled web_search and web_fetch tools.
func WebToolsFromConfig(cfg *config.OpenOctaConfig, projectRoot string) []tool.Tool {
	if !webToolsEnabled {
		return nil
	}
	webCfg := webToolsConfigFromOpenOcta(cfg)
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		root = "."
	}
	var out []tool.Tool
	if webCfg.isSearchEnabled() {
		out = append(out, &WebSearchTool{Config: webCfg})
	}
	if webCfg.isFetchEnabled() {
		out = append(out, &WebFetchTool{Config: webCfg, ProjectRoot: root})
	}
	return out
}

type webToolsConfig struct {
	SearchEnabled  *bool
	FetchEnabled   *bool
	SearchProvider string
	SearchAPIKey   string
	MaxResults     int
	FetchMaxChars  int
	Timeout        time.Duration
	UserAgent      string
}

func webToolsConfigFromOpenOcta(cfg *config.OpenOctaConfig) *webToolsConfig {
	out := &webToolsConfig{}
	if cfg == nil || cfg.Tools == nil || cfg.Tools.Web == nil {
		return out
	}
	web := cfg.Tools.Web
	if web.Search != nil {
		out.SearchEnabled = web.Search.Enabled
		if web.Search.Provider != nil {
			out.SearchProvider = strings.TrimSpace(*web.Search.Provider)
		}
		if web.Search.APIKey != nil {
			out.SearchAPIKey = strings.TrimSpace(*web.Search.APIKey)
		}
		if web.Search.MaxResults != nil && *web.Search.MaxResults > 0 {
			out.MaxResults = *web.Search.MaxResults
		}
		if web.Search.TimeoutSeconds != nil && *web.Search.TimeoutSeconds > 0 {
			out.Timeout = time.Duration(*web.Search.TimeoutSeconds) * time.Second
		}
	}
	if web.Fetch != nil {
		out.FetchEnabled = web.Fetch.Enabled
		if web.Fetch.MaxChars != nil && *web.Fetch.MaxChars > 0 {
			out.FetchMaxChars = *web.Fetch.MaxChars
		}
		if web.Fetch.TimeoutSeconds != nil && *web.Fetch.TimeoutSeconds > 0 {
			if out.Timeout == 0 {
				out.Timeout = time.Duration(*web.Fetch.TimeoutSeconds) * time.Second
			}
		}
		if web.Fetch.UserAgent != nil {
			out.UserAgent = strings.TrimSpace(*web.Fetch.UserAgent)
		}
	}
	return out
}

func (c *webToolsConfig) isSearchEnabled() bool {
	if c == nil || c.SearchEnabled == nil {
		return true
	}
	return *c.SearchEnabled
}

func (c *webToolsConfig) isFetchEnabled() bool {
	if c == nil || c.FetchEnabled == nil {
		return true
	}
	return *c.FetchEnabled
}

func (c *webToolsConfig) searchProvider() string {
	if c != nil {
		if p := strings.TrimSpace(c.SearchProvider); p != "" {
			return strings.ToLower(p)
		}
	}
	return "bing"
}

func (c *webToolsConfig) searchMaxResults() int {
	if c != nil && c.MaxResults > 0 {
		return c.MaxResults
	}
	return defaultSearchMaxResults
}

func (c *webToolsConfig) fetchMaxChars() int {
	if c != nil && c.FetchMaxChars > 0 {
		return c.FetchMaxChars
	}
	return defaultFetchMaxChars
}

func (c *webToolsConfig) httpTimeout() time.Duration {
	if c != nil && c.Timeout > 0 {
		return c.Timeout
	}
	return defaultHTTPTimeout
}

func (c *webToolsConfig) userAgent() string {
	if c != nil {
		if ua := strings.TrimSpace(c.UserAgent); ua != "" {
			return ua
		}
	}
	return defaultWebUserAgent
}

func (c *webToolsConfig) braveAPIKey() string {
	if c != nil {
		return strings.TrimSpace(c.SearchAPIKey)
	}
	return ""
}

func stringParam(params map[string]interface{}, key string) string {
	if params == nil {
		return ""
	}
	v, ok := params[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func toolErrorResult(msg string) *tool.ToolResult {
	return &tool.ToolResult{Success: false, Output: msg}
}
