// OpenAI-compatible chat-completions provider with SSE streaming and tool-call
// accumulation. Spec §6.4, §7.
package llmclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/OptLTD/swiflow/library/httputil"
)

// streamIdleTimeout aborts a streaming response whose body stalls (no SSE data)
// for this long. Guards against providers that return 200 then hang mid-stream,
// which otherwise blocks the whole agent run until the outer ctx deadline.
// Package var so tests can shorten it.
var streamIdleTimeout = 60 * time.Second

// llmMaxRetries / llmRetryBaseDelay control exponential backoff for transient
// failures (429, 5xx, stalls, network). Package vars so tests can tune them.
var (
	llmMaxRetries     = 2
	llmRetryBaseDelay = 500 * time.Millisecond
)

// APIError carries the HTTP status of a non-2xx provider response so callers can
// classify it (429/5xx transient vs 400/401/403 fatal). Modeled loosely on
// go-openai's status-based error handling — without adopting the library.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("provider http %d: %s", e.StatusCode, e.Body)
}

// IsContextOverflow reports whether err is a context-window / prompt-too-long
// failure. Callers should compact messages and retry rather than treating this
// as a generic fatal LLM error. 400-class overflows are NOT retryableLLMError.
func IsContextOverflow(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	var ae *APIError
	if errors.As(err, &ae) {
		msg = strings.ToLower(ae.Body + " " + msg)
	}
	for _, s := range []string{
		"context_length_exceeded",
		"maximum context length",
		"max context length",
		"context window",
		"prompt is too long",
		"prompt too long",
		"request too large",
		"too many tokens",
		"token limit",
		"tokens exceed",
		"exceeds the context",
		"exceed context",
		"input is too long",
		"message length too long",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// retryableLLMError reports whether err is worth retrying with backoff.
func retryableLLMError(err error) bool {
	if err == nil {
		return false
	}
	if IsContextOverflow(err) {
		return false
	}
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.StatusCode == 429 || ae.StatusCode >= 500
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, s := range []string{
		"stalled", "timeout", "reset by peer", "connection refused",
		"i/o timeout", "eof", "network is unreachable", "client.timeout",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// OpenAIProvider talks to any OpenAI-compatible /chat/completions endpoint.
type OpenAIProvider struct {
	name         string
	apiBase      string
	apiKey       string
	defaultModel string
	client       *http.Client
	// disableThinking sends GLM's `thinking:{type:disabled}` to turn off model
	// reasoning. Only emitted when true so non-GLM providers are unaffected.
	disableThinking bool
}

// SetDisableThinking toggles sending the GLM thinking-disabled parameter.
func (p *OpenAIProvider) SetDisableThinking(v bool) { p.disableThinking = v }

// NewOpenAIProvider constructs a provider. apiBase should not have a trailing
// slash (it is trimmed).
func NewOpenAIProvider(name, apiBase, apiKey, defaultModel string) *OpenAIProvider {
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}
	apiBase = strings.TrimRight(apiBase, "/")
	if defaultModel == "" {
		defaultModel = "gpt-4o-mini"
	}
	return &OpenAIProvider{
		name:         name,
		apiBase:      apiBase,
		apiKey:       apiKey,
		defaultModel: defaultModel,
		client:       httputil.Client(180 * time.Second),
	}
}

func (p *OpenAIProvider) Name() string         { return p.name }
func (p *OpenAIProvider) DefaultModel() string { return p.defaultModel }

// Chat is a non-streaming completion (onChunk == nil).
func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return p.ChatStream(ctx, req, nil)
}

// ChatStream calls the completions endpoint, streaming chunks to onChunk when
// non-nil, and returns the aggregated response. Transient failures (429, 5xx,
// stalls, network) are retried with exponential backoff, but only while nothing
// has been streamed yet (so a retry never double-emits partial output).
func (p *OpenAIProvider) ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error) {
	delay := llmRetryBaseDelay
	var lastResp *ChatResponse
	var lastErr error
	for attempt := 0; ; attempt++ {
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
		resp, err := p.chatOnce(ctx, req, hook)
		if err == nil {
			return resp, nil
		}
		lastResp, lastErr = resp, err
		if attempt >= llmMaxRetries || ctx.Err() != nil || emitted || !retryableLLMError(err) {
			return lastResp, lastErr
		}
		slog.Warn("llm.retry", "provider", p.name, "attempt", attempt+1, "delay", delay.String(), "error", err.Error())
		select {
		case <-ctx.Done():
			return lastResp, lastErr
		case <-time.After(delay):
		}
		delay *= 2
	}
}

// chatOnce performs a single request/parse without retry.
func (p *OpenAIProvider) chatOnce(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}
	body := map[string]any{
		"model":    model,
		"messages": buildMessages(req.Messages),
		"stream":   onChunk != nil,
	}
	if tools := buildTools(req.Tools); len(tools) > 0 {
		body["tools"] = tools
		body["tool_choice"] = "auto"
	}
	if p.disableThinking {
		// GLM (Zhipu) chat-completions accepts a thinking switch; disabling it
		// removes the long reasoning phase that dominates TTFT.
		body["thinking"] = map[string]any{"type": "disabled"}
	}
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiBase+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(b)}
	}

	if onChunk == nil {
		return parseNonStream(resp.Body)
	}
	return streamWithIdleGuard(resp.Body, onChunk, streamIdleTimeout)
}

// idleResetReader signals activity every time the underlying stream yields bytes,
// so a watchdog can distinguish a slow-but-live stream from a fully stalled one.
type idleResetReader struct {
	r     io.Reader
	reset func()
}

func (ir *idleResetReader) Read(p []byte) (int, error) {
	n, err := ir.r.Read(p)
	if n > 0 && ir.reset != nil {
		ir.reset()
	}
	return n, err
}

type streamParser func(r io.Reader, onChunk func(StreamChunk)) (*ChatResponse, error)

// streamWithIdleGuard parses an SSE stream but aborts if no data arrives for
// idle. On idle expiry it closes the body (unblocking the scanner) and returns a
// clear error so the caller can wrap up instead of hanging until ctx deadline.
func streamWithIdleGuard(body io.ReadCloser, onChunk func(StreamChunk), idle time.Duration) (*ChatResponse, error) {
	return streamWithIdleGuardParse(body, onChunk, idle, parseStream)
}

func streamWithIdleGuardParse(body io.ReadCloser, onChunk func(StreamChunk), idle time.Duration, parse streamParser) (*ChatResponse, error) {
	if parse == nil {
		parse = parseStream
	}
	if idle <= 0 {
		return parse(body, onChunk)
	}
	resetCh := make(chan struct{}, 1)
	reset := func() {
		select {
		case resetCh <- struct{}{}:
		default:
		}
	}
	watchDone := make(chan struct{})
	var stalled atomic.Bool
	go func() {
		t := time.NewTimer(idle)
		defer t.Stop()
		for {
			select {
			case <-watchDone:
				return
			case <-resetCh:
				if !t.Stop() {
					select {
					case <-t.C:
					default:
					}
				}
				t.Reset(idle)
			case <-t.C:
				stalled.Store(true)
				_ = body.Close()
				return
			}
		}
	}()

	out, err := parse(&idleResetReader{r: body, reset: reset}, onChunk)
	close(watchDone)
	if stalled.Load() {
		return out, fmt.Errorf("llm stream stalled: no data for %s", idle)
	}
	return out, err
}

func buildMessages(msgs []Message) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		// content is always present (some strict providers reject a missing
		// field even for assistant tool-call messages / empty tool results).
		msg := map[string]any{"role": m.Role, "content": m.Content}
		if m.Thinking != "" {
			msg["reasoning_content"] = m.Thinking
		}
		if len(m.ToolCalls) > 0 {
			tcs := make([]map[string]any, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				args, _ := json.Marshal(tc.Arguments)
				tcs = append(tcs, map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Name,
						"arguments": string(args),
					},
				})
			}
			msg["tool_calls"] = tcs
		}
		if m.ToolCallID != "" {
			msg["tool_call_id"] = m.ToolCallID
		}
		if m.ToolName != "" {
			msg["name"] = m.ToolName
		}
		out = append(out, msg)
	}
	return out
}

func buildTools(tools []ToolDef) []map[string]any {
	if len(tools) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		})
	}
	return out
}

func parseNonStream(r io.Reader) (*ChatResponse, error) {
	var out struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
				Reasoning        string `json:"reasoning"`
				ToolCalls        []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(r).Decode(&out); err != nil {
		return nil, err
	}
	r2 := &ChatResponse{FinishReason: "stop"}
	if len(out.Choices) > 0 {
		c := out.Choices[0]
		r2.Content = c.Message.Content
		r2.Thinking = firstNonEmpty(c.Message.ReasoningContent, c.Message.Reasoning)
		r2.FinishReason = c.FinishReason
		for _, tc := range c.Message.ToolCalls {
			r2.ToolCalls = append(r2.ToolCalls, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: parseArgs(tc.Function.Arguments),
			})
		}
		if len(r2.ToolCalls) > 0 {
			r2.FinishReason = "tool_calls"
		}
	}
	return r2, nil
}

type toolCallAcc struct {
	ID      string
	Name    string
	rawArgs string
}

func parseStream(r io.Reader, onChunk func(StreamChunk)) (*ChatResponse, error) {
	result := &ChatResponse{FinishReason: "stop"}
	accs := map[int]*toolCallAcc{}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					Reasoning        string `json:"reasoning"`
					ToolCalls        []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		d := chunk.Choices[0].Delta
		if t := firstNonEmpty(d.ReasoningContent, d.Reasoning); t != "" {
			result.Thinking += t
			onChunk(StreamChunk{Thinking: t})
		}
		if d.Content != "" {
			result.Content += d.Content
			onChunk(StreamChunk{Content: d.Content})
		}
		for _, tc := range d.ToolCalls {
			a, ok := accs[tc.Index]
			if !ok {
				a = &toolCallAcc{}
				accs[tc.Index] = a
			}
			if tc.ID != "" {
				a.ID = tc.ID
			}
			if tc.Function.Name != "" {
				a.Name = strings.TrimSpace(tc.Function.Name)
			}
			a.rawArgs += tc.Function.Arguments
		}
		if fr := chunk.Choices[0].FinishReason; fr != "" {
			result.FinishReason = fr
		}
	}
	if err := sc.Err(); err != nil {
		return result, err
	}

	// Collect+sort indices so non-contiguous tool-call indices are not dropped.
	indices := make([]int, 0, len(accs))
	for i := range accs {
		indices = append(indices, i)
	}
	sort.Ints(indices)
	for _, i := range indices {
		a := accs[i]
		if a.Name == "" {
			continue
		}
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:        a.ID,
			Name:      a.Name,
			Arguments: parseArgs(a.rawArgs),
		})
	}
	if len(result.ToolCalls) > 0 && result.FinishReason != "length" {
		result.FinishReason = "tool_calls"
	}
	onChunk(StreamChunk{Done: true})
	return result, nil
}

func parseArgs(s string) map[string]any {
	args := map[string]any{}
	if s == "" {
		return args
	}
	_ = json.Unmarshal([]byte(s), &args)
	return args
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
