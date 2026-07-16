package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/internal/observe"
	"github.com/OptLTD/swiflow/library/support"
)

// ChildRunner runs an isolated agent turn for delegate_task (implemented by agent.Runner).
type ChildRunner interface {
	RunChild(ctx context.Context, opts ChildRunOpts, onDelta func(string)) (ChildResult, error)
	LastAssistantContent(ctx context.Context, sessionID string) string
}

// ChildRunOpts configures a subagent run.
type ChildRunOpts struct {
	SessionID   string
	AgentKey    string
	UserMessage string
	MaxRounds   int
	// ParentSessionID marks the child's session as owned by this parent so it
	// stays out of the top-level session list.
	ParentSessionID string
	// OnProgress, when non-nil, receives the child's latest action (tool name or
	// streamed text) so the parent UI can show live progress on the delegate block.
	OnProgress func(ToolProgress)
	// MaxWallClock caps the child's wall-clock independently of the parent's tool
	// timeout; 0 = inherit parent context deadline only.
	MaxWallClock time.Duration
	// AllowTools: optional; if non-nil, only these tool names are offered.
	// delegate_task does not expose this — children get the full toolkit (minus DenyTools).
	AllowTools []string
}

// ChildMetrics summarizes a child run for the parent (kept small on purpose).
type ChildMetrics struct {
	Rounds    int `json:"rounds"`
	ToolCalls int `json:"tool_calls"`
	Failures  int `json:"failures"`
}

// ChildResult is the structured outcome of a delegated run. The parent only ever
// sees this (plus artifacts left in the workspace) — keeping its context clean.
type ChildResult struct {
	Status    string       `json:"status"` // done|budget|stall|error|blocked
	Summary   string       `json:"summary"`
	Artifacts []string     `json:"artifacts,omitempty"`
	Metrics   ChildMetrics `json:"metrics"`
	Err       string       `json:"error,omitempty"`
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
	return "Spawn ONE isolated sub-agent for a BATCH of remaining work (own session + round budget); " +
		"returns only its final summary. Put EVERY remaining @/ path and the output artifact " +
		"(e.g. xlsx under workspace) inside goal — never one file per delegate_task, never invent a path/tools parameter. " +
		"The child picks tools itself (document_extract, fs_*, etc.). Required when many attachments or table/Excel batch work remains."
}
func (t *delegateTaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"goal": map[string]any{
				"type": "string",
				"description": "Full batch instructions for the child: list every remaining @/ path inline, " +
					"what to extract, and where to write the deliverable (e.g. @/result.xlsx). One call covers all remaining files.",
			},
			"context":    map[string]any{"type": "string", "description": "Brief shared context; keep short (schema hints, column names)"},
			"max_rounds": map[string]any{"type": "integer", "description": "Child round budget (default 10, max 16)"},
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
	maxRounds := 10
	switch v := args["max_rounds"].(type) {
	case float64:
		maxRounds = int(v)
	case int:
		maxRounds = v
	}
	if maxRounds <= 0 {
		maxRounds = 10
	}
	if maxRounds > 16 {
		maxRounds = 16
	}

	rc, _ := RunContextFrom(ctx)
	parent := rc.SessionID
	if parent == "" {
		parent = "unknown"
	}
	id := support.NewID()
	if len(id) > 8 {
		id = id[:8]
	}
	childKey := fmt.Sprintf("sub-%s-%s", parent, id)
	agentKey := rc.Agent
	if agentKey == "" {
		agentKey = "default"
	}

	userMsg := goal
	if contextHint != "" {
		userMsg = "Context:\n" + contextHint + "\n\nGoal:\n" + goal
	}

	var lastAssistant string
	observe.DelegateStart(parent, childKey, maxRounds)
	t0 := time.Now()
	result, err := t.runner.RunChild(ctx, ChildRunOpts{
		SessionID:       childKey,
		AgentKey:        agentKey,
		UserMessage:     userMsg,
		MaxRounds:       maxRounds,
		ParentSessionID: parent,
		OnProgress:      rc.Emit,
		// Child gets the full toolkit (minus deny list); it chooses tools from the goal.
	}, func(delta string) {
		lastAssistant += delta
	})
	observe.DelegateEnd(parent, childKey, time.Since(t0).Milliseconds(), err)

	summary := strings.TrimSpace(result.Summary)
	if summary == "" {
		summary = strings.TrimSpace(lastAssistant)
	}
	if summary == "" {
		summary = strings.TrimSpace(t.runner.LastAssistantContent(ctx, childKey))
	}
	// Hard failure with no usable work: surface the error to the parent.
	if err != nil && summary == "" {
		return "", fmt.Errorf("sub-agent %s: %w", childKey, err)
	}
	if summary == "" {
		summary = "(sub-agent finished with empty summary)"
	}
	status := result.Status
	if status == "" {
		status = "done"
	}
	out, _ := json.Marshal(map[string]any{
		"child_session": childKey,
		"status":        status,
		"summary":       summary,
		"artifacts":     result.Artifacts,
		"metrics":       result.Metrics,
	})
	return string(out), nil
}

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
