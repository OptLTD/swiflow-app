package agent

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/OptLTD/swiflow/internal/llmclient"
	"github.com/OptLTD/swiflow/internal/store/testutil"
	"github.com/OptLTD/swiflow/internal/tool"
)

func TestSubagentSpawnReturnsImmediately(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)

	reg := tool.NewRegistry()
	prov := &scriptedProvider{
		steps: []*llmclient.ChatResponse{
			{
				ToolCalls: []llmclient.ToolCall{{
					ID:   "call1",
					Name: "subagent_spawn",
					Arguments: map[string]any{
						"goal":       "summarize Paris",
						"max_rounds": float64(2),
					},
				}},
				FinishReason: "tool_calls",
			},
			{Content: "Subagent started.", FinishReason: "stop"},
		},
	}

	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", prov)
	tool.RegisterSubagent(reg, runner)

	var spawnResult string
	var spawnDur time.Duration
	var tSpawn time.Time
	err := runner.Run(context.Background(), "parent-1", "default", "spawn please", func(ev Event) {
		if ev.Type == "tool_call" && ev.Name == "subagent_spawn" {
			tSpawn = time.Now()
		}
		if ev.Type == "tool_result" && ev.Name == "subagent_spawn" {
			spawnDur = time.Since(tSpawn)
			spawnResult = ev.Result
		}
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if spawnResult == "" {
		t.Fatal("missing spawn tool result")
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(spawnResult), &parsed); err != nil {
		t.Fatalf("spawn json: %v", err)
	}
	if parsed["status"] != "running" {
		t.Fatalf("spawn status=%v, want running", parsed["status"])
	}
	cs, _ := parsed["child_session"].(string)
	if !strings.HasPrefix(cs, "sub-parent-1-") {
		t.Fatalf("child_session=%q", cs)
	}
	if spawnDur > 2*time.Second {
		t.Fatalf("spawn took too long (%v); should return immediately", spawnDur)
	}
}

func TestSubagentWaitRejectsMultipleRunning(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)

	runner := NewRunner(RunnerDeps{Store: st, Tools: tool.NewRegistry()})

	runner.subagents.mu.Lock()
	runner.subagents.jobs["sub-a"] = &subagentJob{
		parentSession: "parent-m",
		childSession:  "sub-a",
		status:        subagentStatusRunning,
		done:          make(chan struct{}),
	}
	runner.subagents.jobs["sub-b"] = &subagentJob{
		parentSession: "parent-m",
		childSession:  "sub-b",
		status:        subagentStatusRunning,
		done:          make(chan struct{}),
	}
	runner.subagents.mu.Unlock()

	_, err := runner.SubagentWaitJSON(context.Background(), "parent-m", "sub-a", 1)
	if err == nil || !strings.Contains(err.Error(), "multiple subagents still running") {
		t.Fatalf("wait err=%v, want multiple running rejection", err)
	}
}

func TestSubagentSpawnEmitsProgress(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	reg := tool.NewRegistry()
	reg.Register(&echoTool{})

	prov := &scriptedProvider{
		steps: []*llmclient.ChatResponse{
			{ToolCalls: []llmclient.ToolCall{{ID: "e1", Name: "echo"}}, FinishReason: "tool_calls"},
			{Content: "all done", FinishReason: "stop"},
		},
	}
	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", prov)

	var sawProgress bool
	_, err := runner.subagents.Spawn(SpawnOpts{
		ParentSession:   "parent-9",
		SpawnToolCallID: "call1",
		AgentKey:        "default",
		UserMessage:     "do work",
		Goal:            "do work",
		MaxRounds:       3,
		OnProgress: func(p tool.ToolProgress) {
			if p.Content == "echo" {
				sawProgress = true
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for !sawProgress && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !sawProgress {
		t.Fatal("expected progress from child run")
	}
}

func TestSubagentChildFullToolkit(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	reg := tool.NewRegistry()
	reg.Register(&echoTool{})

	var childToolNames []string
	prov := &scriptedProvider{
		steps: []*llmclient.ChatResponse{
			{ToolCalls: []llmclient.ToolCall{{ID: "e1", Name: "echo"}}, FinishReason: "tool_calls"},
			{Content: "child done", FinishReason: "stop"},
		},
	}
	capturing := &captureToolsProvider{inner: prov, onCall: func(_ int, tools []llmclient.ToolDef) {
		for _, d := range tools {
			childToolNames = append(childToolNames, d.Name)
		}
	}}

	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", capturing)

	_, err := runner.RunChild(context.Background(), tool.ChildRunOpts{
		SessionID:       "sub-direct",
		AgentKey:        "default",
		UserMessage:     "echo",
		MaxRounds:       3,
		ParentSessionID: "parent-x",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range childToolNames {
		if strings.HasPrefix(n, "subagent_") {
			t.Fatalf("child must not get subagent tools, got %s", n)
		}
	}
	foundEcho := false
	for _, n := range childToolNames {
		if n == "echo" {
			foundEcho = true
		}
	}
	if !foundEcho {
		t.Fatalf("child tools=%v", childToolNames)
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
