package agent

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/OptLTD/swiflow/internal/llmclient"
	"github.com/OptLTD/swiflow/internal/store/testutil"
	"github.com/OptLTD/swiflow/internal/tool"
)

// flakyProvider returns scripted steps until call index reaches errAfter, then
// always returns err. Simulates a mid-run LLM stall/network failure.
type flakyProvider struct {
	mu       sync.Mutex
	steps    []*llmclient.ChatResponse
	errAfter int
	err      error
	i        int
}

func (p *flakyProvider) Name() string         { return "openai" }
func (p *flakyProvider) DefaultModel() string { return "mock" }
func (p *flakyProvider) Chat(ctx context.Context, req llmclient.ChatRequest) (*llmclient.ChatResponse, error) {
	return p.ChatStream(ctx, req, nil)
}
func (p *flakyProvider) ChatStream(ctx context.Context, req llmclient.ChatRequest, onChunk func(llmclient.StreamChunk)) (*llmclient.ChatResponse, error) {
	p.mu.Lock()
	i := p.i
	p.i++
	p.mu.Unlock()
	if i >= p.errAfter {
		return nil, p.err
	}
	resp := p.steps[i]
	if resp.Content != "" && onChunk != nil {
		onChunk(llmclient.StreamChunk{Content: resp.Content})
	}
	return resp, nil
}

// A sub-agent whose LLM stalls after doing tool work must still exit cleanly:
// no error returned, a done event emitted, and an honest wrap-up persisted.
func TestChildExitsCleanlyOnLLMError(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	reg := tool.NewRegistry()
	reg.Register(&echoTool{})

	prov := &flakyProvider{
		steps: []*llmclient.ChatResponse{
			{ToolCalls: []llmclient.ToolCall{{ID: "c1", Name: "echo", Arguments: map[string]any{}}}, FinishReason: "tool_calls"},
		},
		errAfter: 1,
		err:      errors.New("llm stream stalled: no data for 60s"),
	}
	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", prov)

	var sawDone bool
	var sawErr string
	err := runner.RunOpts(context.Background(), "child-llm-err", "default", "go", func(ev Event) {
		switch ev.Type {
		case "done":
			sawDone = true
		case "error":
			sawErr = ev.Error
		}
	}, RunOpts{MaxRounds: 5, DenyTools: map[string]bool{
		"subagent_spawn": true, "subagent_status": true, "subagent_wait": true, "clarify": true,
	}})
	if err != nil {
		t.Fatalf("want clean exit (nil err), got %v", err)
	}
	if !sawDone {
		t.Fatal("want done event on graceful LLM-error exit")
	}
	if sawErr != "" {
		t.Fatalf("did not expect error event, got %q", sawErr)
	}
	msgs, _ := st.ListMessages(context.Background(), "child-llm-err")
	var last string
	for _, m := range msgs {
		if m.Role == "assistant" && strings.TrimSpace(m.Content) != "" {
			last = m.Content
		}
	}
	if !strings.Contains(last, "未能完成") {
		t.Fatalf("want honest wrap-up summary, got %q", last)
	}
}

// Two tools dispatched in one turn run concurrently and are both awaited before
// the next LLM turn, which must observe the real results (never placeholders).
func TestBatchDrainsBeforeNextRound(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	reg := tool.NewRegistry()
	reg.Register(&sleepTool{name: "slow_a", d: 250 * time.Millisecond, body: "A-done"})
	reg.Register(&sleepTool{name: "slow_b", d: 250 * time.Millisecond, body: "B-done"})

	// Two attached files, both dispatched in round 0 (concurrent). The next real
	// LLM round (call index 1) must observe both real results.
	goal := "[UPLOAD FILES START]\n@/a.png\n@/b.png\n[UPLOAD FILES END]\nOCR both."
	inner := &scriptedProvider{steps: []*llmclient.ChatResponse{
		{ToolCalls: []llmclient.ToolCall{
			{ID: "ca", Name: "slow_a", Arguments: map[string]any{"path": "@/a.png"}},
			{ID: "cb", Name: "slow_b", Arguments: map[string]any{"path": "@/b.png"}},
		}, FinishReason: "tool_calls"},
		{Content: "done", FinishReason: "stop"},
	}}

	var mu sync.Mutex
	var round1Tool map[string]string
	prov := &captureProvider{inner: inner, onCall: func(i int, req llmclient.ChatRequest) {
		if i == 1 {
			mu.Lock()
			round1Tool = map[string]string{}
			for _, m := range req.Messages {
				if m.Role == "tool" {
					round1Tool[m.ToolCallID] = m.Content
				}
			}
			mu.Unlock()
		}
	}}

	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", prov)

	if err := runner.Run(context.Background(), "batch-drain", "default", goal, func(Event) {}); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if round1Tool["ca"] != "A-done" || round1Tool["cb"] != "B-done" {
		t.Fatalf("round-1 LLM should see drained real results, got %#v", round1Tool)
	}
}
