package llmclient

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
)

// wireMode is the in-process cached Responses vs Chat preference.
type wireMode int

const (
	wireUnknown wireMode = iota
	wireResponses
	wireChat
)

// AdaptiveProvider defaults to the Responses API and falls back to Chat
// Completions when the endpoint or model does not support Responses.
type AdaptiveProvider struct {
	name      string
	apiBase   string
	responses *ResponsesProvider
	chat      *OpenAIProvider

	mu         sync.Mutex
	mode       wireMode
	chatModels map[string]bool // models forced onto Chat (Responses model unsupported)
}

// NewAdaptiveProvider constructs the production agent LLM client: Responses
// first, Chat as compatibility fallback.
func NewAdaptiveProvider(name, apiBase, apiKey, defaultModel string) *AdaptiveProvider {
	return &AdaptiveProvider{
		name:       name,
		apiBase:    strings.TrimRight(apiBase, "/"),
		responses:  NewResponsesProvider(name, apiBase, apiKey, defaultModel),
		chat:       NewOpenAIProvider(name, apiBase, apiKey, defaultModel),
		chatModels: map[string]bool{},
	}
}

// SetDisableThinking forwards to the Chat fallback only (GLM extension).
func (p *AdaptiveProvider) SetDisableThinking(v bool) {
	p.chat.SetDisableThinking(v)
}

func (p *AdaptiveProvider) Name() string         { return p.name }
func (p *AdaptiveProvider) DefaultModel() string { return p.responses.DefaultModel() }

func (p *AdaptiveProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return p.ChatStream(ctx, req, nil)
}

func (p *AdaptiveProvider) ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.DefaultModel()
	}
	if p.preferChat(model) {
		return p.chat.ChatStream(ctx, req, onChunk)
	}

	emitted := false
	hook := onChunk
	if onChunk != nil {
		hook = func(c StreamChunk) {
			if c.Content != "" || c.Thinking != "" {
				emitted = true
			}
			onChunk(c)
		}
	}

	resp, err := p.responses.ChatStream(ctx, req, hook)
	if err == nil {
		p.markResponsesOK()
		return resp, nil
	}
	if emitted || !isResponsesUnsupported(err) {
		return resp, err
	}

	p.markWireFallback(model, err)
	slog.Info("llm.wire_fallback",
		"provider", p.name,
		"api_base", p.apiBase,
		"model", model,
		"error", err.Error(),
	)
	return p.chat.ChatStream(ctx, req, onChunk)
}

func (p *AdaptiveProvider) preferChat(model string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.mode == wireChat {
		return true
	}
	return p.chatModels[model]
}

func (p *AdaptiveProvider) markResponsesOK() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.mode == wireUnknown {
		p.mode = wireResponses
	}
}

func (p *AdaptiveProvider) markWireFallback(model string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var ae *APIError
	if errors.As(err, &ae) && ae.StatusCode == http.StatusBadRequest {
		// Model-level: keep trying Responses for other models on this base.
		p.chatModels[model] = true
		return
	}
	p.mode = wireChat
}

// isResponsesUnsupported reports whether err means this endpoint/model cannot
// speak Responses and Chat should be used instead.
func isResponsesUnsupported(err error) bool {
	if err == nil {
		return false
	}
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	if ae.StatusCode == http.StatusNotFound || ae.StatusCode == http.StatusMethodNotAllowed {
		return true
	}
	if ae.StatusCode != http.StatusBadRequest {
		return false
	}
	msg := strings.ToLower(ae.Body + " " + ae.Error())
	for _, s := range []string{
		"unknown endpoint",
		"unknown url",
		"invalid endpoint",
		"does not support",
		"not supported for",
		"not supported by",
		"unsupported model",
		"model is not supported",
		"model not supported",
		"only supports",
		"not available for responses",
		"responses api is not",
		"invalid model",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
