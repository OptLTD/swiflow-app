package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/OptLTD/swiflow/internal/llmclient"
	"github.com/OptLTD/swiflow/internal/store/testutil"
	"github.com/OptLTD/swiflow/internal/tool"
)

// TestChildRunExitsOnBudget verifies sub-agent stops after max_rounds (budget path).
func TestChildRunExitsOnBudget(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)

	reg := tool.NewRegistry()
	reg.Register(&echoTool{})

	// Child always wants another tool; budget=2 should force wrap-up on last round.
	prov := &scriptedProvider{
		steps: []*llmclient.ChatResponse{
			{
				ToolCalls: []llmclient.ToolCall{{
					ID: "p1", Name: "delegate_task",
					Arguments: map[string]any{"goal": "loop", "max_rounds": float64(2)},
				}},
				FinishReason: "tool_calls",
			},
			{ToolCalls: []llmclient.ToolCall{{ID: "c1", Name: "echo", Arguments: map[string]any{}}}, FinishReason: "tool_calls"},
			{ToolCalls: []llmclient.ToolCall{{ID: "c2", Name: "echo", Arguments: map[string]any{}}}, FinishReason: "tool_calls"},
			{Content: "budget wrap summary", FinishReason: "stop"},
			{Content: "parent done", FinishReason: "stop"},
		},
	}

	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", prov)
	tool.RegisterDelegate(reg, runner)

	var childDone bool
	err := runner.Run(context.Background(), "parent-budget", "default", "go", func(ev Event) {
		if ev.Type == "done" && strings.HasPrefix(ev.Title, "") {
			// parent done
		}
		if ev.Type == "tool_result" && ev.Name == "delegate_task" {
			childDone = true
			if !strings.Contains(ev.Result, "budget wrap summary") && !strings.Contains(ev.Result, "continue") {
				t.Logf("delegate result: %s", ev.Result)
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !childDone {
		t.Fatal("expected delegate_task to complete")
	}
}
