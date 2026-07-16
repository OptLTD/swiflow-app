package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/OptLTD/swiflow/library/window"
)

type clarifyTool struct {
	bridge *window.Bridge
}

// RegisterClarify registers the clarify (ask-user) tool.
func RegisterClarify(r *Registry, bridge *window.Bridge) {
	if bridge == nil {
		bridge = window.NewBridge()
	}
	r.Register(&clarifyTool{bridge: bridge})
}

func (t *clarifyTool) Name() string { return "clarify" }
func (t *clarifyTool) Description() string {
	return "Ask the user a clarifying question and wait for their answer before continuing. Use when you need a choice, confirmation, or missing info."
}
func (t *clarifyTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "Question shown to the user",
			},
			"options": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional short choices; user may still type free text unless allow_free_text is false",
			},
			"allow_free_text": map[string]any{
				"type":        "boolean",
				"description": "If false, only options may be chosen (default true)",
			},
		},
		"required": []any{"question"},
	}
}

func (t *clarifyTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	question, _ := args["question"].(string)
	if question == "" {
		return "", fmt.Errorf("question required")
	}
	allowFree := true
	if v, ok := args["allow_free_text"].(bool); ok {
		allowFree = v
	}
	var options []string
	if raw, ok := args["options"].([]any); ok {
		for _, it := range raw {
			if s, ok := it.(string); ok && s != "" {
				options = append(options, s)
			}
		}
	}
	if !allowFree && len(options) == 0 {
		return "", fmt.Errorf("options required when allow_free_text is false")
	}

	rc, ok := RunContextFrom(ctx)
	if !ok || rc.SessionID == "" {
		return "", fmt.Errorf("ui client unavailable")
	}

	payload := map[string]any{
		"question":        question,
		"options":         options,
		"allow_free_text": allowFree,
	}
	result, err := t.bridge.RequestTimeout(ctx, rc.SessionID, "clarify", payload, window.ClarifyTimeout)
	if err != nil {
		return "", err
	}
	// Normalize answer JSON for the model.
	var parsed map[string]any
	if json.Unmarshal([]byte(result), &parsed) == nil {
		if ans, _ := parsed["answer"].(string); ans != "" {
			out, _ := json.Marshal(map[string]any{"answer": ans})
			return string(out), nil
		}
	}
	if result == "" {
		return "", fmt.Errorf("empty clarify answer")
	}
	out, _ := json.Marshal(map[string]any{"answer": result})
	return string(out), nil
}
