// OpenAI-compatible Responses API provider (/v1/responses) with SSE streaming
// and function-call accumulation. Agent default wire; Chat is Adaptive fallback.
package llmclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/library/httputil"
)

// ResponsesProvider talks to an OpenAI-compatible /responses endpoint.
type ResponsesProvider struct {
	name         string
	apiBase      string
	apiKey       string
	defaultModel string
	client       *http.Client
}

// NewResponsesProvider constructs a Responses-API provider. apiBase should not
// have a trailing slash (it is trimmed).
func NewResponsesProvider(name, apiBase, apiKey, defaultModel string) *ResponsesProvider {
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}
	apiBase = strings.TrimRight(apiBase, "/")
	if defaultModel == "" {
		defaultModel = "gpt-4o-mini"
	}
	return &ResponsesProvider{
		name:         name,
		apiBase:      apiBase,
		apiKey:       apiKey,
		defaultModel: defaultModel,
		client:       httputil.Client(180 * time.Second),
	}
}

func (p *ResponsesProvider) Name() string         { return p.name }
func (p *ResponsesProvider) DefaultModel() string { return p.defaultModel }

// Chat is a non-streaming completion (onChunk == nil).
func (p *ResponsesProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return p.ChatStream(ctx, req, nil)
}

// ChatStream calls /responses, streaming chunks to onChunk when non-nil.
// Retry policy matches OpenAIProvider (transient only, no double-emit).
func (p *ResponsesProvider) ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error) {
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
		resp, err := p.responsesOnce(ctx, req, hook)
		if err == nil {
			return resp, nil
		}
		lastResp, lastErr = resp, err
		if attempt >= llmMaxRetries || ctx.Err() != nil || emitted || !retryableLLMError(err) {
			return lastResp, lastErr
		}
		slog.Warn("llm.retry", "provider", p.name, "wire", "responses", "attempt", attempt+1, "delay", delay.String(), "error", err.Error())
		select {
		case <-ctx.Done():
			return lastResp, lastErr
		case <-time.After(delay):
		}
		delay *= 2
	}
}

func (p *ResponsesProvider) responsesOnce(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}
	instructions, input := buildResponsesInput(req.Messages)
	body := map[string]any{
		"model":  model,
		"input":  input,
		"stream": onChunk != nil,
	}
	if instructions != "" {
		body["instructions"] = instructions
	}
	if tools := buildResponsesTools(req.Tools); len(tools) > 0 {
		body["tools"] = tools
		body["tool_choice"] = "auto"
	}
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiBase+"/responses", bytes.NewReader(raw))
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
		return parseResponsesNonStream(resp.Body)
	}
	return streamWithIdleGuardParse(resp.Body, onChunk, streamIdleTimeout, parseResponsesStream)
}

// buildResponsesInput maps chat-style messages into Responses instructions + input items.
func buildResponsesInput(msgs []Message) (instructions string, input []map[string]any) {
	var sysParts []string
	input = make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "system":
			if m.Content != "" {
				sysParts = append(sysParts, m.Content)
			}
		case "user":
			input = append(input, map[string]any{
				"type":    "message",
				"role":    "user",
				"content": m.Content,
			})
		case "assistant":
			if m.Thinking != "" {
				input = append(input, map[string]any{
					"type": "reasoning",
					"content": []map[string]any{
						{"type": "reasoning_text", "text": m.Thinking},
					},
				})
			}
			if m.Content != "" {
				input = append(input, map[string]any{
					"type":    "message",
					"role":    "assistant",
					"content": m.Content,
				})
			}
			for _, tc := range m.ToolCalls {
				args, _ := json.Marshal(tc.Arguments)
				item := map[string]any{
					"type":      "function_call",
					"call_id":   tc.ID,
					"name":      tc.Name,
					"arguments": string(args),
				}
				input = append(input, item)
			}
		case "tool":
			input = append(input, map[string]any{
				"type":    "function_call_output",
				"call_id": m.ToolCallID,
				"output":  m.Content,
			})
		default:
			input = append(input, map[string]any{
				"type":    "message",
				"role":    m.Role,
				"content": m.Content,
			})
		}
	}
	if len(sysParts) > 0 {
		instructions = strings.Join(sysParts, "\n\n")
	}
	return instructions, input
}

func buildResponsesTools(tools []ToolDef) []map[string]any {
	if len(tools) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		params := t.Parameters
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		out = append(out, map[string]any{
			"type":        "function",
			"name":        t.Name,
			"description": t.Description,
			"parameters":  params,
		})
	}
	return out
}

func parseResponsesNonStream(r io.Reader) (*ChatResponse, error) {
	var out struct {
		Status string `json:"status"`
		Output []struct {
			Type      string `json:"type"`
			Role      string `json:"role"`
			CallID    string `json:"call_id"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
			Content   []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		IncompleteDetails *struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
	}
	if err := json.NewDecoder(r).Decode(&out); err != nil {
		return nil, err
	}
	r2 := &ChatResponse{FinishReason: "stop"}
	for _, item := range out.Output {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				if part.Type == "output_text" || part.Type == "" {
					r2.Content += part.Text
				}
			}
		case "reasoning":
			for _, part := range item.Content {
				if part.Type == "reasoning_text" || part.Type == "summary_text" || part.Type == "" {
					r2.Thinking += part.Text
				}
			}
		case "function_call":
			r2.ToolCalls = append(r2.ToolCalls, ToolCall{
				ID:        item.CallID,
				Name:      item.Name,
				Arguments: parseArgs(item.Arguments),
			})
		}
	}
	if len(r2.ToolCalls) > 0 {
		r2.FinishReason = "tool_calls"
	} else if out.Status == "incomplete" || (out.IncompleteDetails != nil && out.IncompleteDetails.Reason == "max_output_tokens") {
		r2.FinishReason = "length"
	}
	return r2, nil
}

func parseResponsesStream(r io.Reader, onChunk func(StreamChunk)) (*ChatResponse, error) {
	result := &ChatResponse{FinishReason: "stop"}
	accs := map[int]*toolCallAcc{}
	byItemID := map[string]int{}
	nextIdx := 0

	ensureAcc := func(outputIndex int, itemID string) *toolCallAcc {
		if itemID != "" {
			if i, ok := byItemID[itemID]; ok {
				return accs[i]
			}
		}
		if outputIndex < 0 {
			outputIndex = nextIdx
		}
		a, ok := accs[outputIndex]
		if !ok {
			a = &toolCallAcc{}
			accs[outputIndex] = a
			if outputIndex >= nextIdx {
				nextIdx = outputIndex + 1
			}
		}
		if itemID != "" {
			byItemID[itemID] = outputIndex
		}
		return a
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	eventType := ""
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			eventType = ""
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue
		}
		typ, _ := ev["type"].(string)
		if typ == "" {
			typ = eventType
		}
		switch typ {
		case "response.output_text.delta":
			if d := asString(ev["delta"]); d != "" {
				result.Content += d
				onChunk(StreamChunk{Content: d})
			}
		case "response.reasoning_text.delta", "response.reasoning_summary_text.delta":
			if d := asString(ev["delta"]); d != "" {
				result.Thinking += d
				onChunk(StreamChunk{Thinking: d})
			}
		case "response.output_item.added":
			item, _ := ev["item"].(map[string]any)
			if item == nil {
				continue
			}
			if asString(item["type"]) != "function_call" {
				continue
			}
			idx := asInt(ev["output_index"])
			itemID := asString(item["id"])
			a := ensureAcc(idx, itemID)
			if id := asString(item["call_id"]); id != "" {
				a.ID = id
			}
			if name := strings.TrimSpace(asString(item["name"])); name != "" {
				a.Name = name
			}
			if args := asString(item["arguments"]); args != "" {
				a.rawArgs = args
			}
		case "response.function_call_arguments.delta":
			idx := asInt(ev["output_index"])
			itemID := asString(ev["item_id"])
			a := ensureAcc(idx, itemID)
			a.rawArgs += asString(ev["delta"])
		case "response.function_call_arguments.done":
			idx := asInt(ev["output_index"])
			itemID := asString(ev["item_id"])
			a := ensureAcc(idx, itemID)
			if args := asString(ev["arguments"]); args != "" {
				a.rawArgs = args
			}
			if name := strings.TrimSpace(asString(ev["name"])); name != "" {
				a.Name = name
			}
		case "response.incomplete":
			result.FinishReason = "length"
			if resp, ok := ev["response"].(map[string]any); ok {
				mergeResponsesOutput(result, resp)
			}
		case "response.failed":
			result.FinishReason = "error"
			if errObj, ok := ev["response"].(map[string]any); ok {
				if e, ok := errObj["error"].(map[string]any); ok {
					return result, fmt.Errorf("responses failed: %s", asString(e["message"]))
				}
			}
			return result, fmt.Errorf("responses failed")
		case "response.completed":
			if resp, ok := ev["response"].(map[string]any); ok {
				mergeResponsesOutput(result, resp)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return result, err
	}

	indices := make([]int, 0, len(accs))
	for i := range accs {
		indices = append(indices, i)
	}
	sort.Ints(indices)
	seen := map[string]bool{}
	for _, i := range indices {
		a := accs[i]
		if a.Name == "" {
			continue
		}
		key := a.ID + "\x00" + a.Name + "\x00" + a.rawArgs
		if seen[key] {
			continue
		}
		seen[key] = true
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:        a.ID,
			Name:      a.Name,
			Arguments: parseArgs(a.rawArgs),
		})
	}
	if len(result.ToolCalls) > 0 && result.FinishReason != "length" && result.FinishReason != "error" {
		result.FinishReason = "tool_calls"
	}
	onChunk(StreamChunk{Done: true})
	return result, nil
}

// mergeResponsesOutput fills gaps from a final response object (completed/incomplete).
func mergeResponsesOutput(dst *ChatResponse, resp map[string]any) {
	raw, err := json.Marshal(resp)
	if err != nil {
		return
	}
	parsed, err := parseResponsesNonStream(bytes.NewReader(raw))
	if err != nil || parsed == nil {
		return
	}
	if dst.Content == "" {
		dst.Content = parsed.Content
	}
	if dst.Thinking == "" {
		dst.Thinking = parsed.Thinking
	}
	if len(dst.ToolCalls) == 0 && len(parsed.ToolCalls) > 0 {
		dst.ToolCalls = parsed.ToolCalls
	}
	if parsed.FinishReason == "length" {
		dst.FinishReason = "length"
	} else if len(dst.ToolCalls) > 0 || len(parsed.ToolCalls) > 0 {
		dst.FinishReason = "tool_calls"
	}
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	default:
		return ""
	}
}

func asInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(t)
		return n
	default:
		return -1
	}
}
