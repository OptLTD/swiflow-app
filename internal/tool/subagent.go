package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/library/support"
)

// ChildRunner runs an isolated agent turn for subagent_spawn (implemented by agent.Runner).
type ChildRunner interface {
	RunChild(ctx context.Context, opts ChildRunOpts, onDelta func(string)) (ChildResult, error)
	LastAssistantContent(ctx context.Context, sessionID string) string
}

// ChildRunOpts configures a subagent run.
type ChildRunOpts struct {
	SessionID       string
	AgentKey        string
	UserMessage     string
	MaxRounds       int
	ParentSessionID string
	OnProgress      func(ToolProgress)
	MaxWallClock    time.Duration
	AllowTools      []string
}

// ChildMetrics summarizes a child run for the parent (kept small on purpose).
type ChildMetrics struct {
	Rounds    int `json:"rounds"`
	ToolCalls int `json:"tool_calls"`
	Failures  int `json:"failures"`
	WallMS    int64 `json:"wall_ms,omitempty"`
}

// ChildResult is the structured outcome of a delegated run.
type ChildResult struct {
	Status    string       `json:"status"` // done|budget|stall|error|blocked
	Summary   string       `json:"summary"`
	Artifacts []string     `json:"artifacts,omitempty"`
	Metrics   ChildMetrics `json:"metrics"`
	Err       string       `json:"error,omitempty"`
}

// SubagentBackend is implemented by agent.Runner for async subagent tools.
type SubagentBackend interface {
	SpawnSubagent(ctx context.Context, rc RunContext, goal, contextHint string, maxRounds int) (string, error)
	SubagentStatusJSON(ctx context.Context, parentSession, childSession string) (string, error)
	SubagentWaitJSON(ctx context.Context, parentSession, childSession string, timeoutSec int) (string, error)
}

type subagentTools struct {
	backend SubagentBackend
}

// RegisterSubagent registers subagent_spawn / subagent_status / subagent_wait.
func RegisterSubagent(r *Registry, backend SubagentBackend) {
	if backend == nil {
		return
	}
	t := &subagentTools{backend: backend}
	r.Register(&subagentSpawnTool{t: t})
	r.Register(&subagentStatusTool{t: t})
	r.Register(&subagentWaitTool{t: t})
}

type subagentSpawnTool struct{ t *subagentTools }

func (t *subagentSpawnTool) Name() string { return "subagent_spawn" }
func (t *subagentSpawnTool) Description() string {
	return "Start ONE isolated sub-agent for a BATCH of remaining work (async; returns immediately with child_session). " +
		"Put EVERY remaining @/ path and the output artifact (e.g. xlsx) inside goal. " +
		"The child picks tools itself. Use subagent_status to check progress; use subagent_wait only when this is the last running subagent."
}
func (t *subagentSpawnTool) Parameters() map[string]any {
	return spawnParameters()
}

func spawnParameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"goal": map[string]any{
				"type": "string",
				"description": "Full batch instructions: list every remaining @/ path inline, " +
					"what to extract, and where to write the deliverable (e.g. @/result.xlsx).",
			},
			"context":    map[string]any{"type": "string", "description": "Brief shared context; keep short"},
			"max_rounds": map[string]any{"type": "integer", "description": "Child round budget (default 10, max 16)"},
		},
		"required": []any{"goal"},
	}
}

func parseSpawnArgs(args map[string]any) (goal, contextHint string, maxRounds int, err error) {
	goal, _ = args["goal"].(string)
	if goal == "" {
		return "", "", 0, fmt.Errorf("goal required")
	}
	contextHint, _ = args["context"].(string)
	maxRounds = 10
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
	return goal, contextHint, maxRounds, nil
}

func (t *subagentSpawnTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	goal, contextHint, maxRounds, err := parseSpawnArgs(args)
	if err != nil {
		return "", err
	}
	rc, ok := RunContextFrom(ctx)
	if !ok {
		rc = RunContext{}
	}
	child, err := t.t.backend.SpawnSubagent(ctx, rc, goal, contextHint, maxRounds)
	if err != nil {
		return "", err
	}
	out, _ := json.Marshal(map[string]any{
		"child_session": child,
		"status":        "running",
		"goal":          goal,
	})
	return string(out), nil
}

type subagentStatusTool struct{ t *subagentTools }

func (t *subagentStatusTool) Name() string { return "subagent_status" }
func (t *subagentStatusTool) Description() string {
	return "Read progress of a subagent spawned by subagent_spawn (non-blocking). " +
		"Returns status, todos, last_action, metrics; includes summary/artifacts when terminal."
}
func (t *subagentStatusTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"child_session": map[string]any{"type": "string", "description": "Child session key from subagent_spawn"},
		},
		"required": []any{"child_session"},
	}
}

func (t *subagentStatusTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	child, _ := args["child_session"].(string)
	child = strings.TrimSpace(child)
	if child == "" {
		return "", fmt.Errorf("child_session required")
	}
	rc, _ := RunContextFrom(ctx)
	parent := rc.SessionID
	if parent == "" {
		parent = "unknown"
	}
	return t.t.backend.SubagentStatusJSON(ctx, parent, child)
}

type subagentWaitTool struct{ t *subagentTools }

func (t *subagentWaitTool) Name() string { return "subagent_wait" }
func (t *subagentWaitTool) Description() string {
	return "Block until a subagent finishes or timeout (default 900s). " +
		"Use ONLY when no more subagent_spawn calls are needed and this is the sole running subagent. " +
		"Otherwise use subagent_status."
}
func (t *subagentWaitTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"child_session":    map[string]any{"type": "string", "description": "Child session key from subagent_spawn"},
			"timeout_seconds": map[string]any{"type": "integer", "description": "Max wait (default 900, max 900)"},
		},
		"required": []any{"child_session"},
	}
}

func (t *subagentWaitTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	child, _ := args["child_session"].(string)
	child = strings.TrimSpace(child)
	if child == "" {
		return "", fmt.Errorf("child_session required")
	}
	timeoutSec := 900
	switch v := args["timeout_seconds"].(type) {
	case float64:
		timeoutSec = int(v)
	case int:
		timeoutSec = v
	}
	if timeoutSec <= 0 {
		timeoutSec = 900
	}
	if timeoutSec > 900 {
		timeoutSec = 900
	}
	rc, _ := RunContextFrom(ctx)
	parent := rc.SessionID
	if parent == "" {
		parent = "unknown"
	}
	return t.t.backend.SubagentWaitJSON(ctx, parent, child, timeoutSec)
}

// ParseChildSession extracts child_session from spawn result JSON (tests/UI).
func ParseChildSession(result string) string {
	var parsed map[string]any
	if json.Unmarshal([]byte(result), &parsed) != nil {
		return ""
	}
	cs, _ := parsed["child_session"].(string)
	return cs
}

// NewChildSessionKey builds a child session id for tests.
func NewChildSessionKey(parent string) string {
	id := support.NewID()
	if len(id) > 8 {
		id = id[:8]
	}
	return fmt.Sprintf("sub-%s-%s", parent, id)
}
