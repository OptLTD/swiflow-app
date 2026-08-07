package llmclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestBuildResponsesInput(t *testing.T) {
	instr, input := buildResponsesInput([]Message{
		{Role: "system", Content: "be brief"},
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "calling", ToolCalls: []ToolCall{
			{ID: "call_1", Name: "skill_use", Arguments: map[string]any{"slug": "demo"}},
		}},
		{Role: "tool", ToolCallID: "call_1", Content: "ok"},
	})
	if instr != "be brief" {
		t.Fatalf("instructions=%q", instr)
	}
	if len(input) != 4 {
		t.Fatalf("input len=%d %#v", len(input), input)
	}
	if input[2]["type"] != "function_call" || input[2]["call_id"] != "call_1" {
		t.Fatalf("function_call item=%#v", input[2])
	}
	if input[3]["type"] != "function_call_output" {
		t.Fatalf("output item=%#v", input[3])
	}
}

func TestParseResponsesNonStream(t *testing.T) {
	body := `{
	  "status": "completed",
	  "output": [
	    {"type":"reasoning","content":[{"type":"reasoning_text","text":"think"}]},
	    {"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]},
	    {"type":"function_call","call_id":"call_1","name":"skill_use","arguments":"{\"slug\":\"demo\"}"}
	  ]
	}`
	resp, err := parseResponsesNonStream(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hello" || resp.Thinking != "think" {
		t.Fatalf("content=%q thinking=%q", resp.Content, resp.Thinking)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "skill_use" || resp.FinishReason != "tool_calls" {
		t.Fatalf("got %+v reason=%s", resp.ToolCalls, resp.FinishReason)
	}
}

func TestParseResponsesStream(t *testing.T) {
	stream := strings.Join([]string{
		`event: response.reasoning_text.delta`,
		`data: {"type":"response.reasoning_text.delta","delta":"why"}`,
		``,
		`event: response.output_text.delta`,
		`data: {"type":"response.output_text.delta","delta":"hi"}`,
		``,
		`event: response.output_item.added`,
		`data: {"type":"response.output_item.added","output_index":1,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"skill_use","arguments":""}}`,
		``,
		`event: response.function_call_arguments.delta`,
		`data: {"type":"response.function_call_arguments.delta","output_index":1,"item_id":"fc_1","delta":"{\"slug\":"}`,
		``,
		`event: response.function_call_arguments.delta`,
		`data: {"type":"response.function_call_arguments.delta","output_index":1,"item_id":"fc_1","delta":"\"demo\"}"}`,
		``,
		`event: response.completed`,
		`data: {"type":"response.completed","response":{"status":"completed","output":[]}}`,
		``,
	}, "\n")

	var contents, thinkings []string
	resp, err := parseResponsesStream(strings.NewReader(stream), func(c StreamChunk) {
		if c.Content != "" {
			contents = append(contents, c.Content)
		}
		if c.Thinking != "" {
			thinkings = append(thinkings, c.Thinking)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hi" || strings.Join(contents, "") != "hi" {
		t.Fatalf("content=%q chunks=%v", resp.Content, contents)
	}
	if resp.Thinking != "why" || strings.Join(thinkings, "") != "why" {
		t.Fatalf("thinking=%q chunks=%v", resp.Thinking, thinkings)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "skill_use" {
		t.Fatalf("tools=%+v", resp.ToolCalls)
	}
	if resp.ToolCalls[0].Arguments["slug"] != "demo" {
		t.Fatalf("args=%v", resp.ToolCalls[0].Arguments)
	}
	if resp.FinishReason != "tool_calls" {
		t.Fatalf("finish=%s", resp.FinishReason)
	}
}

func TestResponsesProviderChat(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`))
	}))
	defer srv.Close()

	p := NewResponsesProvider("openai", srv.URL+"/v1", "key", "m")
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "m",
		Messages: []Message{{Role: "system", Content: "sys"}, {Role: "user", Content: "hi"}},
		Tools:    []ToolDef{{Name: "skill_use", Description: "d", Parameters: map[string]any{"type": "object"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok" {
		t.Fatalf("content=%q", resp.Content)
	}
	if gotPath != "/v1/responses" {
		t.Fatalf("path=%s", gotPath)
	}
	if gotBody["instructions"] != "sys" {
		t.Fatalf("body=%v", gotBody)
	}
	tools, _ := gotBody["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tools=%v", gotBody["tools"])
	}
	tool0, _ := tools[0].(map[string]any)
	if tool0["type"] != "function" || tool0["name"] != "skill_use" {
		t.Fatalf("tool0=%v", tool0)
	}
	if _, ok := tool0["function"]; ok {
		t.Fatal("Responses tools must be flat, not nested under function")
	}
}

func TestIsResponsesUnsupported(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{&APIError{StatusCode: 404, Body: "nope"}, true},
		{&APIError{StatusCode: 405, Body: ""}, true},
		{&APIError{StatusCode: 400, Body: `{"error":"model not supported for responses"}`}, true},
		{&APIError{StatusCode: 400, Body: `{"error":"only supports deepseek-v4-flash"}`}, true},
		{&APIError{StatusCode: 400, Body: `{"error":"invalid json schema"}`}, false},
		{&APIError{StatusCode: 401, Body: "unauthorized"}, false},
		{&APIError{StatusCode: 429, Body: "rate"}, false},
		{errorString("network boom"), false},
	}
	for _, c := range cases {
		if got := isResponsesUnsupported(c.err); got != c.want {
			t.Errorf("isResponsesUnsupported(%v)=%v want %v", c.err, got, c.want)
		}
	}
}

func TestAdaptiveFallback404ThenChat(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch {
		case strings.HasSuffix(r.URL.Path, "/responses"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		case strings.HasSuffix(r.URL.Path, "/chat/completions"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"via-chat"},"finish_reason":"stop"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := NewAdaptiveProvider("p", srv.URL+"/v1", "key", "m")
	resp, err := p.Chat(context.Background(), ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "via-chat" {
		t.Fatalf("content=%q", resp.Content)
	}
	if len(paths) < 2 || !strings.HasSuffix(paths[0], "/responses") || !strings.HasSuffix(paths[1], "/chat/completions") {
		t.Fatalf("paths=%v", paths)
	}

	// Second call should skip Responses (cached chat mode).
	paths = nil
	resp, err = p.Chat(context.Background(), ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "again"}}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "via-chat" {
		t.Fatalf("content=%q", resp.Content)
	}
	if len(paths) != 1 || !strings.HasSuffix(paths[0], "/chat/completions") {
		t.Fatalf("cached paths=%v", paths)
	}
}

func TestAdaptiveModelUnsupportedFallsBackPerModel(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch {
		case strings.HasSuffix(r.URL.Path, "/responses"):
			b, _ := io.ReadAll(r.Body)
			var body struct {
				Model string `json:"model"`
			}
			_ = json.Unmarshal(b, &body)
			if body.Model == "pro" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"model not supported"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"via-responses"}]}]}`))
		case strings.HasSuffix(r.URL.Path, "/chat/completions"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"via-chat"},"finish_reason":"stop"}]}`))
		}
	}))
	defer srv.Close()

	p := NewAdaptiveProvider("p", srv.URL+"/v1", "key", "flash")
	resp, err := p.Chat(context.Background(), ChatRequest{Model: "pro", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "via-chat" {
		t.Fatalf("pro content=%q", resp.Content)
	}

	resp, err = p.Chat(context.Background(), ChatRequest{Model: "flash", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "via-responses" {
		t.Fatalf("flash should still use responses, got %q", resp.Content)
	}
}

func TestAdaptiveNoFallbackOn401(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad key"}`))
	}))
	defer srv.Close()

	p := NewAdaptiveProvider("p", srv.URL+"/v1", "key", "m")
	_, err := p.Chat(context.Background(), ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err == nil {
		t.Fatal("want error")
	}
	if len(paths) != 1 || !strings.HasSuffix(paths[0], "/responses") {
		t.Fatalf("401 must not fall back to chat, paths=%v", paths)
	}
}
