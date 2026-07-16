package llmclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseNonStreamToolCalls(t *testing.T) {
	body := `{
	  "choices": [{
	    "message": {
	      "content": "ok",
	      "tool_calls": [{
	        "id": "call_1",
	        "function": {"name": "skill_use", "arguments": "{\"slug\":\"demo\"}"}
	      }]
	    },
	    "finish_reason": "tool_calls"
	  }]
	}`
	resp, err := parseNonStream(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "skill_use" {
		t.Fatalf("got %+v", resp.ToolCalls)
	}
}

func TestParseStreamAccumulates(t *testing.T) {
	stream := "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n" +
		"data: [DONE]\n\n"
	var chunks []string
	resp, err := parseStream(strings.NewReader(stream), func(c StreamChunk) {
		chunks = append(chunks, c.Content)
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hi" || strings.Join(chunks, "") != "hi" {
		t.Fatalf("content=%q chunks=%v", resp.Content, chunks)
	}
}

// hangReader blocks in Read until Close is called; mimics a stalled SSE body.
type hangReader struct {
	closed chan struct{}
	once   sync.Once
}

func (h *hangReader) Read(p []byte) (int, error) {
	<-h.closed
	return 0, io.EOF
}
func (h *hangReader) Close() error {
	h.once.Do(func() { close(h.closed) })
	return nil
}

func TestStreamIdleGuardAbortsStall(t *testing.T) {
	r := &hangReader{closed: make(chan struct{})}
	t0 := time.Now()
	_, err := streamWithIdleGuard(r, func(StreamChunk) {}, 80*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "stalled") {
		t.Fatalf("want stall error, got %v", err)
	}
	if d := time.Since(t0); d > time.Second {
		t.Fatalf("idle guard fired too slowly: %v", d)
	}
}

func TestStreamIdleGuardPassesThroughData(t *testing.T) {
	stream := "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n" +
		"data: [DONE]\n\n"
	resp, err := streamWithIdleGuard(io.NopCloser(strings.NewReader(stream)), func(StreamChunk) {}, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hi" {
		t.Fatalf("content=%q", resp.Content)
	}
}

func TestRetryableLLMError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{&APIError{StatusCode: 429}, true},
		{&APIError{StatusCode: 503}, true},
		{&APIError{StatusCode: 400}, false},
		{&APIError{StatusCode: 401}, false},
		{context.Canceled, false},
		{context.DeadlineExceeded, false},
		{errorString("llm stream stalled: no data for 60s"), true},
		{errorString("dial tcp: connection refused"), true},
		{errorString("totally fatal parse issue"), false},
	}
	for _, c := range cases {
		if got := retryableLLMError(c.err); got != c.want {
			t.Errorf("retryableLLMError(%v)=%v want %v", c.err, got, c.want)
		}
	}
}

type errorString string

func (e errorString) Error() string { return string(e) }

// ChatStream retries transient 5xx with backoff, then succeeds.
func TestChatStreamRetriesTransient(t *testing.T) {
	prevMax, prevDelay := llmMaxRetries, llmRetryBaseDelay
	llmMaxRetries, llmRetryBaseDelay = 3, 1*time.Millisecond
	t.Cleanup(func() { llmMaxRetries, llmRetryBaseDelay = prevMax, prevDelay })

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("busy"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider("openai", srv.URL, "key", "m")
	resp, err := p.Chat(context.Background(), ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("want success after retries, got %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("content=%q", resp.Content)
	}
	if calls.Load() != 3 {
		t.Fatalf("want 3 attempts (2 x 503 + 1 ok), got %d", calls.Load())
	}
}

// A fatal 400 is not retried.
func TestChatStreamNoRetryOnFatal(t *testing.T) {
	prevMax, prevDelay := llmMaxRetries, llmRetryBaseDelay
	llmMaxRetries, llmRetryBaseDelay = 3, 1*time.Millisecond
	t.Cleanup(func() { llmMaxRetries, llmRetryBaseDelay = prevMax, prevDelay })

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad"))
	}))
	defer srv.Close()

	p := NewOpenAIProvider("openai", srv.URL, "key", "m")
	_, err := p.Chat(context.Background(), ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err == nil {
		t.Fatal("want error for 400")
	}
	if calls.Load() != 1 {
		t.Fatalf("400 must not be retried, attempts=%d", calls.Load())
	}
}
