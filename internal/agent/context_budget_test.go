package agent

import (
	"strings"
	"testing"

	"github.com/OptLTD/swiflow/internal/llmclient"
)

func longContent(n int) string {
	return strings.Repeat("x", n)
}

func TestFitMessagesToBudgetCompactsOldTools(t *testing.T) {
	msgs := []llmclient.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "start"},
		{Role: "assistant", ToolCalls: []llmclient.ToolCall{{ID: "c1", Name: "fs_read"}}},
		{Role: "tool", ToolCallID: "c1", ToolName: "fs_read", Content: longContent(5000)},
		{Role: "user", Content: "continue"},
		{Role: "assistant", Content: "ok"},
	}
	// Soft target = 85% of budget. KeepTail=2 protects last two messages.
	out := fitMessagesToBudget(msgs, 2500, contextFitOpts{KeepTail: 2})
	if estimateChars(out) > 2500 {
		t.Fatalf("still over budget: %d", estimateChars(out))
	}
	var toolMsg string
	for _, m := range out {
		if m.Role == "tool" && m.ToolCallID == "c1" {
			toolMsg = m.Content
		}
	}
	if toolMsg == "" {
		t.Fatal("expected tool message retained")
	}
	if !strings.Contains(toolMsg, "[compacted]") {
		t.Fatalf("expected compacted tool result, got len=%d", len(toolMsg))
	}
	if out[0].Role != "system" {
		t.Fatalf("system must be kept, got %q", out[0].Role)
	}
	if out[len(out)-1].Content != "ok" {
		t.Fatalf("last message should be preserved, got %+v", out[len(out)-1])
	}
}

func TestFitMessagesToBudgetKeepsToolPairs(t *testing.T) {
	msgs := []llmclient.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "u1"},
		{Role: "assistant", ToolCalls: []llmclient.ToolCall{{ID: "a", Name: "t"}}},
		{Role: "tool", ToolCallID: "a", Content: longContent(3000)},
		{Role: "user", Content: "u2"},
		{Role: "assistant", ToolCalls: []llmclient.ToolCall{{ID: "b", Name: "t"}}},
		{Role: "tool", ToolCallID: "b", Content: longContent(3000)},
		{Role: "user", Content: "latest"},
	}
	out := fitMessagesToBudget(msgs, 1200, contextFitOpts{Aggressive: true, KeepTail: 4})
	out = sanitizeLLMMessages(out)
	for i, m := range out {
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			id := m.ToolCalls[0].ID
			if i+1 >= len(out) || out[i+1].Role != "tool" || out[i+1].ToolCallID != id {
				t.Fatalf("broken tool pair at %d: %+v", i, out)
			}
		}
	}
}

func TestFitMessagesToBudgetNoopWhenUnderBudget(t *testing.T) {
	msgs := []llmclient.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hi"},
	}
	out := fitMessagesToBudget(msgs, 10_000, contextFitOpts{})
	if len(out) != 2 || out[1].Content != "hi" {
		t.Fatalf("unexpected %+v", out)
	}
}

func TestFitMessagesToBudgetDisabled(t *testing.T) {
	msgs := []llmclient.Message{
		{Role: "tool", Content: longContent(9000), ToolCallID: "x"},
	}
	out := fitMessagesToBudget(msgs, 0, contextFitOpts{})
	if len(out[0].Content) != 9000 {
		t.Fatal("budget 0 should skip fitting")
	}
}

func TestEstimateCharsIncludesToolCalls(t *testing.T) {
	msgs := []llmclient.Message{
		{Role: "assistant", ToolCalls: []llmclient.ToolCall{
			{ID: "c1", Name: "fs_read", Arguments: map[string]any{"path": "@/a"}},
		}},
	}
	if estimateChars(msgs) < 20 {
		t.Fatalf("expected tool_calls to count, got %d", estimateChars(msgs))
	}
}

func TestBuildLLMErrorSummaryOverflow(t *testing.T) {
	err := &llmclient.APIError{StatusCode: 400, Body: "context_length_exceeded"}
	got := buildLLMErrorSummary(err)
	if !strings.Contains(got, "上下文过长") {
		t.Fatalf("want overflow wording, got %q", got)
	}
}
