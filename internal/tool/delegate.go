package tool

import (
	"context"
	"encoding/json"
	"fmt"
)

// --- session todos ---

type todoItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type todoStore interface {
	SaveTodos(ctx context.Context, sessionID string, itemsJSON string) error
	LoadTodos(ctx context.Context, sessionID string) (string, error)
}

type todoWriteTool struct{ st todoStore }
type todoReadTool struct{ st todoStore }

// RegisterTodo registers todo_write / todo_read with persistent storage.
func RegisterTodo(r *Registry, st todoStore) {
	r.Register(&todoWriteTool{st: st})
	r.Register(&todoReadTool{st: st})
}

// RegisterDelegate is deprecated; use RegisterSubagent.
func RegisterDelegate(r *Registry, runner ChildRunner) {
	_ = runner
}

func (t *todoWriteTool) Name() string { return "todo_write" }
func (t *todoWriteTool) Description() string {
	return "Replace the session task checklist. Pass items [{id?, text, done?}]."
}
func (t *todoWriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":   map[string]any{"type": "string"},
						"text": map[string]any{"type": "string"},
						"done": map[string]any{"type": "boolean"},
					},
					"required": []any{"text"},
				},
			},
		},
		"required": []any{"items"},
	}
}

func (t *todoWriteTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	raw, _ := args["items"].([]any)
	rc, _ := RunContextFrom(ctx)
	sess := rc.SessionID
	if sess == "" {
		sess = "_"
	}
	items := make([]todoItem, 0, len(raw))
	for i, it := range raw {
		m, _ := it.(map[string]any)
		if m == nil {
			continue
		}
		text, _ := m["text"].(string)
		if text == "" {
			continue
		}
		id, _ := m["id"].(string)
		if id == "" {
			id = fmt.Sprintf("%d", i+1)
		}
		done, _ := m["done"].(bool)
		items = append(items, todoItem{ID: id, Text: text, Done: done})
	}
	b, _ := json.Marshal(items)
	if err := t.st.SaveTodos(ctx, sess, string(b)); err != nil {
		return "", err
	}
	return string(b), nil
}

func (t *todoReadTool) Name() string { return "todo_read" }
func (t *todoReadTool) Description() string {
	return "Read the current session task checklist."
}
func (t *todoReadTool) Parameters() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func (t *todoReadTool) Execute(ctx context.Context, _ map[string]any) (string, error) {
	rc, _ := RunContextFrom(ctx)
	sess := rc.SessionID
	if sess == "" {
		sess = "_"
	}
	return t.st.LoadTodos(ctx, sess)
}
