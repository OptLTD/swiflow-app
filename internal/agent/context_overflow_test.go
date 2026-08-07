package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/OptLTD/swiflow/internal/llmclient"
	"github.com/OptLTD/swiflow/internal/store/testutil"
	"github.com/OptLTD/swiflow/internal/tool"
)

// overflowThenOKProvider fails once with context overflow, then succeeds.
type overflowThenOKProvider struct {
	mu    sync.Mutex
	calls int
}

func (p *overflowThenOKProvider) Name() string         { return "openai" }
func (p *overflowThenOKProvider) DefaultModel() string { return "mock" }
func (p *overflowThenOKProvider) Chat(ctx context.Context, req llmclient.ChatRequest) (*llmclient.ChatResponse, error) {
	return p.ChatStream(ctx, req, nil)
}
func (p *overflowThenOKProvider) ChatStream(ctx context.Context, req llmclient.ChatRequest, onChunk func(llmclient.StreamChunk)) (*llmclient.ChatResponse, error) {
	p.mu.Lock()
	p.calls++
	n := p.calls
	p.mu.Unlock()
	if n == 1 {
		return nil, &llmclient.APIError{StatusCode: 400, Body: `{"error":{"code":"context_length_exceeded","message":"maximum context length exceeded"}}`}
	}
	resp := &llmclient.ChatResponse{Content: "recovered after compact", FinishReason: "stop"}
	if onChunk != nil {
		onChunk(llmclient.StreamChunk{Content: resp.Content})
	}
	return resp, nil
}

func TestContextOverflowCompactsAndRetries(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	reg := tool.NewRegistry()

	prov := &overflowThenOKProvider{}
	runner := NewRunner(RunnerDeps{
		Store:           st,
		Tools:           reg,
		MaxContextChars: 50_000,
	})
	runner.SetProvider("openai", prov)

	var sawDone bool
	err := runner.RunOpts(context.Background(), "ctx-overflow", "default", "hello", func(ev Event) {
		if ev.Type == "done" {
			sawDone = true
		}
		if ev.Type == "error" {
			t.Fatalf("unexpected error event: %s", ev.Error)
		}
	}, RunOpts{MaxRounds: 4, DenyTools: map[string]bool{
		"subagent_spawn": true, "subagent_status": true, "subagent_wait": true, "clarify": true,
	}})
	if err != nil {
		t.Fatalf("want success after compact retry, got %v", err)
	}
	if !sawDone {
		t.Fatal("want done")
	}
	prov.mu.Lock()
	calls := prov.calls
	prov.mu.Unlock()
	if calls < 2 {
		t.Fatalf("want at least 2 LLM calls (overflow + retry), got %d", calls)
	}
}
