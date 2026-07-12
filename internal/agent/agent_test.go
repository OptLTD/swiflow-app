package agent

import (
	"encoding/json"
	"testing"

	"mira/internal/llm"
)

func TestToolCallKeyOrderIndependent(t *testing.T) {
	a := []llm.ToolCall{{ID: "b", Name: "x"}, {ID: "a", Name: "y"}}
	b := []llm.ToolCall{{ID: "a", Name: "y"}, {ID: "b", Name: "x"}}
	if toolCallKey(a) != toolCallKey(b) {
		t.Fatal("expected same key for permuted tool calls")
	}
}

func TestToolCallKeyDiffers(t *testing.T) {
	a := []llm.ToolCall{{ID: "a", Name: "x"}}
	b := []llm.ToolCall{{ID: "b", Name: "x"}}
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
