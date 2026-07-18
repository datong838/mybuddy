// Package model — Xunfei image API request/response types.
package model

// Message is a chat message for legacy Xunfei image completion.
type Message struct {
	Role          string
	Content       string
	ContentBlocks []ContentBlock
}

// Request is a completion request for Xunfei image API.
type Request struct {
	SessionID string
	Messages  []Message
}

// Response is a completion response from Xunfei image API.
type Response struct {
	Message    Message
	Usage      Usage
	StopReason string
}

// StreamResult is a streaming chunk from Xunfei image API.
type StreamResult struct {
	Delta    string
	Final    bool
	Response *Response
}

// StreamHandler receives streaming chunks.
type StreamHandler func(StreamResult) error
