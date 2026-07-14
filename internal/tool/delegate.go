package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/OptLTD/swiflow/internal/util"
)

// ChildRunner runs an isolated agent turn for delegate_task (implemented by agent.Runner).
type ChildRunner interface {
	RunChild(ctx context.Context, opts ChildRunOpts, onDelta func(string)) error
}

// ChildRunOpts configures a subagent run.
type ChildRunOpts struct {
	SessionKey  string
	AgentKey    string
	UserMessage string
	MaxRounds   int
	// AllowTools: if non-nil, only these tool names are offered (delegate_task always denied).
	AllowTools []string
}

type delegateTaskTool struct {
	runner ChildRunner
}

// RegisterDelegate registers delegate_task (after Runner is constructed).
func RegisterDelegate(r *Registry, runner ChildRunner) {
	if runner == nil {
		return
	}
	r.Register(&delegateTaskTool{runner: runner})
}

func (t *delegateTaskTool) Name() string { return "delegate_task" }
func (t *delegateTaskTool) Description() string {
	return "Spawn an isolated sub-agent with its own session and round budget. Returns only the child's final summary."
}
func (t *delegateTaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"goal":       map[string]any{"type": "string", "description": "What the sub-agent should accomplish"},
			"context":    map[string]any{"type": "string", "description": "Brief context; keep short"},
			"max_rounds": map[string]any{"type": "integer", "description": "Child round budget (default 8, max 16)"},
			"tools": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional tool-name whitelist for the child (omit = all except delegate_task)",
			},
		},
		"required": []any{"goal"},
	}
}

func (t *delegateTaskTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.runner == nil {
		return "", fmt.Errorf("delegate_task unavailable")
	}
	goal, _ := args["goal"].(string)
	if goal == "" {
		return "", fmt.Errorf("goal required")
	}
	contextHint, _ := args["context"].(string)
	maxRounds := 8
	switch v := args["max_rounds"].(type) {
	case float64:
		maxRounds = int(v)
	case int:
		maxRounds = v
	}
	if maxRounds <= 0 {
		maxRounds = 8
	}
	if maxRounds > 16 {
		maxRounds = 16
	}

	var allowTools []string
	if raw, ok := args["tools"].([]any); ok && len(raw) > 0 {
		for _, it := range raw {
			if s, ok := it.(string); ok && s != "" && s != "delegate_task" {
				allowTools = append(allowTools, s)
			}
		}
	}

	rc, _ := RunContextFrom(ctx)
	parent := rc.SessionKey
	if parent == "" {
		parent = "unknown"
	}
	id := util.NewID()
	if len(id) > 8 {
		id = id[:8]
	}
	childKey := fmt.Sprintf("sub-%s-%s", parent, id)
	agentKey := rc.AgentKey
	if agentKey == "" {
		agentKey = "default"
	}

	userMsg := goal
	if contextHint != "" {
		userMsg = "Context:\n" + contextHint + "\n\nGoal:\n" + goal
	}

	var lastAssistant string
	err := t.runner.RunChild(ctx, ChildRunOpts{
		SessionKey:  childKey,
		AgentKey:    agentKey,
		UserMessage: userMsg,
		MaxRounds:   maxRounds,
		AllowTools:  allowTools,
	}, func(delta string) {
		lastAssistant += delta
	})
	if err != nil {
		return "", fmt.Errorf("sub-agent %s: %w", childKey, err)
	}
	summary := lastAssistant
	if summary == "" {
		summary = "(sub-agent finished with empty summary)"
	}
	out, _ := json.Marshal(map[string]any{
		"child_session": childKey,
		"summary":       summary,
	})
	return string(out), nil
}

// --- session todos ---

type todoItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

var (
	todoMu     sync.Mutex
	todoBySess = map[string][]todoItem{}
)

type todoWriteTool struct{}
type todoReadTool struct{}

// RegisterTodo registers todo_write / todo_read.
func RegisterTodo(r *Registry) {
	r.Register(&todoWriteTool{})
	r.Register(&todoReadTool{})
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
	sess := rc.SessionKey
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
	todoMu.Lock()
	todoBySess[sess] = items
	todoMu.Unlock()
	b, _ := json.Marshal(items)
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
	sess := rc.SessionKey
	if sess == "" {
		sess = "_"
	}
	todoMu.Lock()
	items := append([]todoItem(nil), todoBySess[sess]...)
	todoMu.Unlock()
	if items == nil {
		items = []todoItem{}
	}
	b, _ := json.Marshal(items)
	return string(b), nil
}
