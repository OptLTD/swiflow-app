// Package llmclient defines the provider abstraction and an OpenAI-compatible client.
// Spec §6.4, §7.
package llmclient

import (
	"context"
)

// ToolCall is a model-requested tool invocation.
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// Message is one chat-completions message.
type Message struct {
	Role       string     `json:"role"` // system|user|assistant|tool
	Content    string     `json:"content"`
	Thinking   string     `json:"thinking,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolName   string     `json:"tool_name,omitempty"`
}

// ToolDef advertises a tool to the model.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ChatRequest is a chat-completions request.
type ChatRequest struct {
	Model    string
	Messages []Message
	Tools    []ToolDef
}

// StreamChunk is one piece of a streamed response.
type StreamChunk struct {
	Content  string
	Thinking string
	Done     bool
}

// ChatResponse is the aggregated result of a (possibly streamed) completion.
type ChatResponse struct {
	Content      string
	Thinking     string
	ToolCalls    []ToolCall
	FinishReason string // stop|tool_calls|length|error
}

// Provider is the LLM provider interface.
type Provider interface {
	Name() string
	DefaultModel() string
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error)
}
