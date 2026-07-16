package agent

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/OptLTD/swiflow/internal/llmclient"
	"github.com/OptLTD/swiflow/internal/store/testutil"
	"github.com/OptLTD/swiflow/internal/tool"
)

// scriptedProvider returns canned ChatResponses in order.
type scriptedProvider struct {
	mu    sync.Mutex
	steps []*llmclient.ChatResponse
	i     int
}

func (p *scriptedProvider) Name() string         { return "openai" }
func (p *scriptedProvider) DefaultModel() string { return "mock" }

func (p *scriptedProvider) Chat(ctx context.Context, req llmclient.ChatRequest) (*llmclient.ChatResponse, error) {
	return p.next()
}

func (p *scriptedProvider) ChatStream(ctx context.Context, req llmclient.ChatRequest, onChunk func(llmclient.StreamChunk)) (*llmclient.ChatResponse, error) {
	resp, err := p.next()
	if err != nil {
		return nil, err
	}
	if resp.Content != "" && onChunk != nil {
		onChunk(llmclient.StreamChunk{Content: resp.Content})
	}
	return resp, nil
}

func (p *scriptedProvider) next() (*llmclient.ChatResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.i >= len(p.steps) {
		return &llmclient.ChatResponse{Content: "unexpected extra call", FinishReason: "stop"}, nil
	}
	resp := p.steps[p.i]
	p.i++
	return resp, nil
}

func TestDelegateTaskSummary(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)

	reg := tool.NewRegistry()
	// Register a noop so whitelist filtering can be exercised separately;
	// this test uses child with no tools (one-shot text reply).
	prov := &scriptedProvider{
		steps: []*llmclient.ChatResponse{
			// Parent: request delegate_task
			{
				ToolCalls: []llmclient.ToolCall{{
					ID:   "call1",
					Name: "delegate_task",
					Arguments: map[string]any{
						"goal":       "summarize Paris",
						"max_rounds": float64(2),
					},
				}},
				FinishReason: "tool_calls",
			},
			// Child: plain summary, no tools
			{Content: "Paris is the capital of France.", FinishReason: "stop"},
			// Parent: after tool result
			{Content: "Subagent said: Paris is the capital of France.", FinishReason: "stop"},
		},
	}

	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", prov)
	tool.RegisterDelegate(reg, runner)

	var events []Event
	err := runner.Run(context.Background(), "parent-1", "default", "delegate please", func(ev Event) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var sawTool, sawSummary bool
	var final string
	for _, ev := range events {
		if ev.Type == "tool_call" && ev.Name == "delegate_task" {
			sawTool = true
		}
		if ev.Type == "tool_result" && ev.Name == "delegate_task" {
			sawSummary = true
			if !strings.Contains(ev.Result, "Paris is the capital of France") {
				t.Fatalf("tool result missing child summary: %s", ev.Result)
			}
			var parsed map[string]any
			if err := json.Unmarshal([]byte(ev.Result), &parsed); err != nil {
				t.Fatalf("result json: %v", err)
			}
			cs, _ := parsed["child_session"].(string)
			if !strings.HasPrefix(cs, "sub-parent-1-") {
				t.Fatalf("child_session=%q", cs)
			}
		}
		if ev.Type == "delta" {
			final += ev.Content
		}
	}
	if !sawTool || !sawSummary {
		t.Fatalf("expected delegate tool call+result, events=%+v", events)
	}
	if !strings.Contains(final, "Subagent said") {
		t.Fatalf("parent final=%q", final)
	}
	if prov.i != 3 {
		t.Fatalf("expected 3 LLM calls (parent, child, parent), got %d", prov.i)
	}
}

func TestDelegateToolsWhitelist(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)

	reg := tool.NewRegistry()
	reg.Register(&echoTool{})

	var childToolNames []string
	prov := &scriptedProvider{
		steps: []*llmclient.ChatResponse{
			{
				ToolCalls: []llmclient.ToolCall{{
					ID:   "c1",
					Name: "delegate_task",
					Arguments: map[string]any{
						"goal":  "echo",
						"tools": []any{"echo"},
					},
				}},
				FinishReason: "tool_calls",
			},
			// Child sees only echo (+ no delegate). Capture via side channel in first child call —
			// we inspect during ChatStream by wrapping... easier: child replies with text.
			{Content: "child done", FinishReason: "stop"},
			{Content: "parent done", FinishReason: "stop"},
		},
	}

	// Wrap provider to capture tools offered on 2nd call (child).
	capturing := &captureToolsProvider{inner: prov, onCall: func(i int, tools []llmclient.ToolDef) {
		if i == 1 {
			for _, d := range tools {
				childToolNames = append(childToolNames, d.Name)
			}
		}
	}}

	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", capturing)
	tool.RegisterDelegate(reg, runner)

	if err := runner.Run(context.Background(), "p2", "default", "go", nil); err != nil {
		t.Fatal(err)
	}
	for _, n := range childToolNames {
		if n == "delegate_task" {
			t.Fatal("child must not get delegate_task")
		}
	}
	foundEcho := false
	for _, n := range childToolNames {
		if n == "echo" {
			foundEcho = true
		}
	}
	if !foundEcho {
		t.Fatalf("child tools=%v, want echo", childToolNames)
	}
}

type echoTool struct{}

func (t *echoTool) Name() string        { return "echo" }
func (t *echoTool) Description() string { return "echo" }
func (t *echoTool) Parameters() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func (t *echoTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	return "ok", nil
}

type captureToolsProvider struct {
	inner  *scriptedProvider
	onCall func(i int, tools []llmclient.ToolDef)
	n      int
}

func (p *captureToolsProvider) Name() string         { return p.inner.Name() }
func (p *captureToolsProvider) DefaultModel() string { return p.inner.DefaultModel() }
func (p *captureToolsProvider) Chat(ctx context.Context, req llmclient.ChatRequest) (*llmclient.ChatResponse, error) {
	i := p.n
	p.n++
	if p.onCall != nil {
		p.onCall(i, req.Tools)
	}
	return p.inner.Chat(ctx, req)
}
func (p *captureToolsProvider) ChatStream(ctx context.Context, req llmclient.ChatRequest, onChunk func(llmclient.StreamChunk)) (*llmclient.ChatResponse, error) {
	i := p.n
	p.n++
	if p.onCall != nil {
		p.onCall(i, req.Tools)
	}
	return p.inner.ChatStream(ctx, req, onChunk)
}
