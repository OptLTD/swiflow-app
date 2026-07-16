package agent

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OptLTD/swiflow/internal/llmclient"
	"github.com/OptLTD/swiflow/internal/store/testutil"
	"github.com/OptLTD/swiflow/internal/tool"
)

// sleepTool blocks for d (or until ctx is cancelled) then returns body.
type sleepTool struct {
	name string
	d    time.Duration
	body string
	n    atomic.Int32
}

func (t *sleepTool) Name() string        { return t.name }
func (t *sleepTool) Description() string { return "sleep" }
func (t *sleepTool) Parameters() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func (t *sleepTool) Execute(ctx context.Context, _ map[string]any) (string, error) {
	t.n.Add(1)
	select {
	case <-time.After(t.d):
		return t.body, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// captureProvider wraps a scriptedProvider and reports each request.
type captureProvider struct {
	inner  *scriptedProvider
	onCall func(i int, req llmclient.ChatRequest)
	i      int
	mu     sync.Mutex
}

func (p *captureProvider) Name() string         { return p.inner.Name() }
func (p *captureProvider) DefaultModel() string { return p.inner.DefaultModel() }
func (p *captureProvider) Chat(ctx context.Context, req llmclient.ChatRequest) (*llmclient.ChatResponse, error) {
	p.mu.Lock()
	i := p.i
	p.i++
	p.mu.Unlock()
	if p.onCall != nil {
		p.onCall(i, req)
	}
	return p.inner.Chat(ctx, req)
}
func (p *captureProvider) ChatStream(ctx context.Context, req llmclient.ChatRequest, onChunk func(llmclient.StreamChunk)) (*llmclient.ChatResponse, error) {
	p.mu.Lock()
	i := p.i
	p.i++
	p.mu.Unlock()
	if p.onCall != nil {
		p.onCall(i, req)
	}
	return p.inner.ChatStream(ctx, req, onChunk)
}

// Independent tools in one turn run concurrently (bounded pool), not serially.
func TestExecutorRunsToolsConcurrently(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	reg := tool.NewRegistry()
	a := &sleepTool{name: "slow_a", d: 200 * time.Millisecond, body: "A"}
	b := &sleepTool{name: "slow_b", d: 200 * time.Millisecond, body: "B"}
	reg.Register(a)
	reg.Register(b)

	prov := &scriptedProvider{steps: []*llmclient.ChatResponse{
		{ToolCalls: []llmclient.ToolCall{
			{ID: "ca", Name: "slow_a", Arguments: map[string]any{}},
			{ID: "cb", Name: "slow_b", Arguments: map[string]any{}},
		}, FinishReason: "tool_calls"},
		{Content: "done", FinishReason: "stop"},
	}}
	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", prov)

	results := map[string]string{}
	var mu sync.Mutex
	t0 := time.Now()
	err := runner.Run(context.Background(), "exec-parallel", "default", "go", func(ev Event) {
		if ev.Type == "tool_result" {
			mu.Lock()
			results[ev.ID] = ev.Result
			mu.Unlock()
		}
	})
	elapsed := time.Since(t0)
	if err != nil {
		t.Fatal(err)
	}
	if results["ca"] != "A" || results["cb"] != "B" {
		t.Fatalf("want both real results, got %#v", results)
	}
	if a.n.Load() != 1 || b.n.Load() != 1 {
		t.Fatalf("exec counts a=%d b=%d", a.n.Load(), b.n.Load())
	}
	// Concurrent: ~200ms, well under the 400ms serial sum.
	if elapsed > 350*time.Millisecond {
		t.Fatalf("expected concurrency; elapsed=%v", elapsed)
	}
}

// The next LLM turn always sees the real (awaited) tool results — no placeholders.
func TestExecutorAwaitsAllBeforeNextTurn(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	reg := tool.NewRegistry()
	reg.Register(&sleepTool{name: "slow_a", d: 150 * time.Millisecond, body: "A-done"})
	reg.Register(&sleepTool{name: "slow_b", d: 150 * time.Millisecond, body: "B-done"})

	inner := &scriptedProvider{steps: []*llmclient.ChatResponse{
		{ToolCalls: []llmclient.ToolCall{
			{ID: "ca", Name: "slow_a", Arguments: map[string]any{}},
			{ID: "cb", Name: "slow_b", Arguments: map[string]any{}},
		}, FinishReason: "tool_calls"},
		{Content: "done", FinishReason: "stop"},
	}}
	var mu sync.Mutex
	var round1 map[string]string
	prov := &captureProvider{inner: inner, onCall: func(i int, req llmclient.ChatRequest) {
		if i == 1 {
			mu.Lock()
			round1 = map[string]string{}
			for _, m := range req.Messages {
				if m.Role == "tool" {
					round1[m.ToolCallID] = m.Content
				}
			}
			mu.Unlock()
		}
	}}
	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", prov)

	if err := runner.Run(context.Background(), "exec-await", "default", "go", func(Event) {}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if round1["ca"] != "A-done" || round1["cb"] != "B-done" {
		t.Fatalf("next turn should see real results, got %#v", round1)
	}
}

// A per-call timeout turns a hung tool into an error outcome; the run continues.
func TestExecutorPerCallTimeout(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	reg := tool.NewRegistry()
	reg.Register(&sleepTool{name: "hang", d: 5 * time.Second, body: "nope"})

	prov := &scriptedProvider{steps: []*llmclient.ChatResponse{
		{ToolCalls: []llmclient.ToolCall{{ID: "c1", Name: "hang", Arguments: map[string]any{}}}, FinishReason: "tool_calls"},
		{Content: "wrapped up", FinishReason: "stop"},
	}}
	runner := NewRunner(RunnerDeps{
		Store: st, Tools: reg,
		ToolTimeouts: map[string]time.Duration{"hang": 60 * time.Millisecond},
	})
	runner.SetProvider("openai", prov)

	var toolErr bool
	t0 := time.Now()
	err := runner.Run(context.Background(), "exec-timeout", "default", "go", func(ev Event) {
		if ev.Type == "tool_result" && ev.ID == "c1" && ev.IsError {
			toolErr = true
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !toolErr {
		t.Fatal("want tool_result with IsError after per-call timeout")
	}
	if time.Since(t0) > 2*time.Second {
		t.Fatalf("per-call timeout did not bound the hung tool: %v", time.Since(t0))
	}
}

// Cancelling the run context unblocks an in-flight tool and finishes the run.
func TestExecutorCancel(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	reg := tool.NewRegistry()
	reg.Register(&sleepTool{name: "hang", d: 5 * time.Second, body: "nope"})

	prov := &scriptedProvider{steps: []*llmclient.ChatResponse{
		{ToolCalls: []llmclient.ToolCall{{ID: "c1", Name: "hang", Arguments: map[string]any{}}}, FinishReason: "tool_calls"},
		{Content: "unreached", FinishReason: "stop"},
	}}
	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", prov)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx, "exec-cancel", "default", "go", func(Event) {})
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("run did not finish after cancel")
	}
}
