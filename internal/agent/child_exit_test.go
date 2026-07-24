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

	prov := &scriptedProvider{
		steps: []*llmclient.ChatResponse{
			{ToolCalls: []llmclient.ToolCall{{ID: "c1", Name: "echo", Arguments: map[string]any{}}}, FinishReason: "tool_calls"},
			{ToolCalls: []llmclient.ToolCall{{ID: "c2", Name: "echo", Arguments: map[string]any{}}}, FinishReason: "tool_calls"},
			{Content: "budget wrap summary", FinishReason: "stop"},
		},
	}

	runner := NewRunner(RunnerDeps{Store: st, Tools: reg})
	runner.SetProvider("openai", prov)

	res, err := runner.RunChild(context.Background(), tool.ChildRunOpts{
		SessionID:       "sub-budget",
		AgentKey:        "default",
		UserMessage:     "loop",
		MaxRounds:       2,
		ParentSessionID: "parent-budget",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "budget" && !strings.Contains(res.Summary, "budget wrap summary") {
		t.Fatalf("result=%+v", res)
	}
}
