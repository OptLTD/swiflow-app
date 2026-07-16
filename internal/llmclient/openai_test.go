package llmclient

import (
	"strings"
	"testing"
)

func TestParseNonStreamToolCalls(t *testing.T) {
	body := `{
	  "choices": [{
	    "message": {
	      "content": "ok",
	      "tool_calls": [{
	        "id": "call_1",
	        "function": {"name": "skill_use", "arguments": "{\"slug\":\"demo\"}"}
	      }]
	    },
	    "finish_reason": "tool_calls"
	  }]
	}`
	resp, err := parseNonStream(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "skill_use" {
		t.Fatalf("got %+v", resp.ToolCalls)
	}
}

func TestParseStreamAccumulates(t *testing.T) {
	stream := "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n" +
		"data: [DONE]\n\n"
	var chunks []string
	resp, err := parseStream(strings.NewReader(stream), func(c StreamChunk) {
		chunks = append(chunks, c.Content)
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hi" || strings.Join(chunks, "") != "hi" {
		t.Fatalf("content=%q chunks=%v", resp.Content, chunks)
	}
}
