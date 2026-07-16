package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/OptLTD/swiflow/internal/llmclient"
	"github.com/OptLTD/swiflow/internal/store"
)

func TestToolCallKeyOrderIndependent(t *testing.T) {
	a := []llmclient.ToolCall{{ID: "b", Name: "x"}, {ID: "a", Name: "y"}}
	b := []llmclient.ToolCall{{ID: "a", Name: "y"}, {ID: "b", Name: "x"}}
	if toolCallKey(a) != toolCallKey(b) {
		t.Fatal("expected same key for permuted tool calls")
	}
}

func TestToolCallKeyDiffers(t *testing.T) {
	a := []llmclient.ToolCall{{ID: "a", Name: "x"}}
	b := []llmclient.ToolCall{{ID: "b", Name: "x"}}
	if toolCallKey(a) == toolCallKey(b) {
		t.Fatal("expected different keys")
	}
}

func TestEventJSON(t *testing.T) {
	ev := Event{Type: "error", Error: "fail"}
	b, err := json.Marshal(ev)
	if err != nil || !json.Valid(b) {
		t.Fatal("invalid event json")
	}
}

func TestSanitizeToolHistoryFillsMissing(t *testing.T) {
	in := []store.Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "", ToolCalls: []store.ToolCall{
			{ID: "c1", Name: "bash"},
			{ID: "c2", Name: "read"},
		}},
		{Role: "tool", Content: "ok", ToolCallId: "c1", ToolName: "bash"},
		{Role: "user", Content: "again"},
	}
	out := sanitizeToolHistory(in)
	if len(out) != 5 {
		t.Fatalf("want 5 msgs, got %d: %+v", len(out), out)
	}
	if out[3].Role != "tool" || out[3].ToolCallId != "c2" {
		t.Fatalf("expected synthetic tool for c2, got %+v", out[3])
	}
	if out[4].Role != "user" {
		t.Fatalf("expected trailing user, got %+v", out[4])
	}
}

func TestSanitizeToolHistoryDropsOrphanTools(t *testing.T) {
	in := []store.Message{
		{Role: "tool", Content: "orphan", ToolCallId: "x"},
		{Role: "user", Content: "hi"},
	}
	out := sanitizeToolHistory(in)
	if len(out) != 1 || out[0].Role != "user" {
		t.Fatalf("want only user, got %+v", out)
	}
}

func TestFormatToolErrorKeepsOutput(t *testing.T) {
	got := formatToolError("Traceback...\nTypeError: bad", fmt.Errorf("failed: exit status 1"))
	if !strings.Contains(got, "Traceback") {
		t.Fatalf("expected traceback in result, got %q", got)
	}
	if !strings.Contains(got, "error: failed: exit status 1") {
		t.Fatalf("expected error suffix, got %q", got)
	}
}

func TestFormatToolErrorEmptyOutput(t *testing.T) {
	got := formatToolError("", fmt.Errorf("tool disabled: exec"))
	if got != "error: tool disabled: exec" {
		t.Fatalf("got %q", got)
	}
}

func TestTruncateHistoryKeepsToolPairs(t *testing.T) {
	in := []store.Message{
		{Role: "user", Content: "1"},
		{Role: "assistant", ToolCalls: []store.ToolCall{{ID: "c1", Name: "bash"}}},
		{Role: "tool", Content: "ok", ToolCallId: "c1"},
		{Role: "user", Content: "2"},
		{Role: "assistant", Content: "done"},
	}
	// Cut such that the window would start mid-pair without repair.
	out := truncateHistory(in, 3)
	for i, m := range out {
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			if i+1 >= len(out) || out[i+1].Role != "tool" || out[i+1].ToolCallId != m.ToolCalls[0].ID {
				t.Fatalf("broken tool pair at %d: %+v", i, out)
			}
		}
	}
	if out[0].Role == "tool" {
		t.Fatal("leading orphan tool after truncate")
	}
}
