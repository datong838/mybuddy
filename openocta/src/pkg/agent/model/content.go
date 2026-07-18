// Package model defines shared model types for agent execution.
package model

import "strings"

type ContentBlockType string

const (
	ContentBlockText     ContentBlockType = "text"
	ContentBlockImage    ContentBlockType = "image"
	ContentBlockVideo    ContentBlockType = "video"
	ContentBlockDocument ContentBlockType = "document"
)

type ContentBlock struct {
	Type      ContentBlockType `json:"type"`
	Text      string           `json:"text,omitempty"`
	MediaType string           `json:"media_type,omitempty"`
	Data      string           `json:"data,omitempty"`
	URL       string           `json:"url,omitempty"`
}

type Usage struct {
	InputTokens         int
	OutputTokens        int
	TotalTokens         int
	CacheReadTokens     int
	CacheCreationTokens int
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]interface{}
}

const PlaceholderAssistantContent = "."

func IsPlaceholderAssistantContent(content string) bool {
	return strings.TrimSpace(content) == PlaceholderAssistantContent
}

func NormalizeAssistantContent(content string, hasToolCalls bool) string {
	if hasToolCalls && IsPlaceholderAssistantContent(content) {
		return ""
	}
	return content
}
