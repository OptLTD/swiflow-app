package agent

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/OptLTD/swiflow/internal/llmclient"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/store/testutil"
	"github.com/OptLTD/swiflow/internal/tool"
)

// reflectScriptProvider records requests so tests can assert reflect nudges.
type reflectScriptProvider struct {
	mu    sync.Mutex
	steps []*llmclient.ChatResponse
	i     int
	reqs  []llmclient.ChatRequest
}

func (p *reflectScriptProvider) Name() string         { return "openai" }
func (p *reflectScriptProvider) DefaultModel() string { return "mock" }
func (p *reflectScriptProvider) Chat(ctx context.Context, req llmclient.ChatRequest) (*llmclient.ChatResponse, error) {
	return p.ChatStream(ctx, req, nil)
}
func (p *reflectScriptProvider) ChatStream(ctx context.Context, req llmclient.ChatRequest, onChunk func(llmclient.StreamChunk)) (*llmclient.ChatResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reqs = append(p.reqs, req)
	if p.i >= len(p.steps) {
		return &llmclient.ChatResponse{Content: "done", FinishReason: "stop"}, nil
	}
	resp := p.steps[p.i]
	p.i++
	if resp.Content != "" && onChunk != nil {
		onChunk(llmclient.StreamChunk{Content: resp.Content})
	}
	return resp, nil
}

func TestReflectGateBlocksPrematureDone(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	reg := tool.NewRegistry()
	reg.Register(&echoTool{})

	prov := &reflectScriptProvider{
		steps: []*llmclient.ChatResponse{
			{ToolCalls: []llmclient.ToolCall{{ID: "c1", Name: "echo", Arguments: map[string]any{}}}, FinishReason: "tool_calls"},
			{Content: "premature done", FinishReason: "stop"},
			{Content: "real done after reflect", FinishReason: "stop"},
		},
	}
	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", prov)

	err := runner.RunOpts(context.Background(), "reflect-1", "default", "do work", func(ev Event) {}, RunOpts{MaxRounds: 8})
	if err != nil {
		t.Fatal(err)
	}
	if prov.i < 3 {
		t.Fatalf("expected at least 3 LLM calls (tool, premature, ship), got %d", prov.i)
	}
	foundReflect := false
	for _, req := range prov.reqs {
		for _, m := range req.Messages {
			if m.Role == "user" && strings.Contains(m.Content, "self-check against the user's goal") {
				foundReflect = true
			}
		}
	}
	if !foundReflect {
		t.Fatal("expected reflect nudge in LLM messages")
	}
}

func TestShortAnswerSkipsReflect(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	reg := tool.NewRegistry()

	prov := &reflectScriptProvider{
		steps: []*llmclient.ChatResponse{
			{Content: "hello!", FinishReason: "stop"},
		},
	}
	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", prov)

	err := runner.RunOpts(context.Background(), "reflect-short", "default", "hi", func(ev Event) {}, RunOpts{MaxRounds: 4})
	if err != nil {
		t.Fatal(err)
	}
	if prov.i != 1 {
		t.Fatalf("short answer should be one LLM call, got %d", prov.i)
	}
	for _, req := range prov.reqs {
		for _, m := range req.Messages {
			if m.Role == "user" && strings.Contains(m.Content, "self-check against the user's goal") {
				t.Fatal("short answer must not enter reflect")
			}
		}
	}
}

func TestBuildSystemInjectsDefaultCharter(t *testing.T) {
	r := NewRunner(RunnerDeps{})
	sys := r.buildSystem(context.Background(), "s1", &store.Agent{Key: "default"}, "", "")
	if !strings.Contains(sys, "## Ways of working") {
		t.Fatal("missing Ways of working section")
	}
	if !strings.Contains(sys, "Prefer delivering the user's stated goal") {
		t.Fatal("expected default charter seed")
	}
}

func TestBuildSystemUsesAgentCharter(t *testing.T) {
	r := NewRunner(RunnerDeps{})
	sys := r.buildSystem(context.Background(), "s1", &store.Agent{
		Key: "default", Charter: "- Always prefer gbk for csv",
	}, "", "")
	if !strings.Contains(sys, "Always prefer gbk for csv") {
		t.Fatal("expected custom charter")
	}
	if strings.Contains(sys, "Prefer delivering the user's stated goal") {
		t.Fatal("custom charter should replace default seed")
	}
}

func TestParseOpenTodos(t *testing.T) {
	if parseOpenTodos(`[{"id":"1","text":"a","done":true}]`) {
		t.Fatal("all done")
	}
	if !parseOpenTodos(`[{"id":"1","text":"a","done":false}]`) {
		t.Fatal("want open")
	}
}

func TestLooksLikeCorrection(t *testing.T) {
	if !looksLikeCorrection("以后都用 gbk") {
		t.Fatal("expected correction")
	}
	if looksLikeCorrection("请分析这份表") {
		t.Fatal("normal request")
	}
}
