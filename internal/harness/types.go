// Package harness is a phase-1 runtime observer: track agent/subagent runs,
// detect goal drift with deterministic rules, and surface harness_warn to the UI.
// It does not inject into the LLM loop (phase 2).
package harness

import "time"

// TodoItem mirrors the agent checklist item.
type TodoItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

// RunMetrics is a compact counters snapshot.
type RunMetrics struct {
	Rounds    int   `json:"rounds"`
	ToolCalls int   `json:"tool_calls"`
	Failures  int   `json:"failures"`
	WallMS    int64 `json:"wall_ms"`
}

// DriftSignal is one detected deviation from expected progress.
type DriftSignal struct {
	Code     string    `json:"code"`
	Severity string    `json:"severity"` // info|warn|error
	Message  string    `json:"message"`
	At       time.Time `json:"at"`
}

// RunSnapshot is the live (or recently finished) view of one session run.
type RunSnapshot struct {
	SessionID   string        `json:"session_id"`
	Tid         string        `json:"tid,omitempty"`
	ParentID    string        `json:"parent_id,omitempty"`
	Agent       string        `json:"agent,omitempty"`
	Status      string        `json:"status"` // idle|running|queued|done|error|budget|stall
	Goal        string        `json:"goal,omitempty"`
	Round       int           `json:"round"`
	MaxRounds   int           `json:"max_rounds,omitempty"`
	Model       string        `json:"model,omitempty"`
	StartedAt   time.Time     `json:"started_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	CurrentTool string        `json:"current_tool,omitempty"`
	LastAction  string        `json:"last_action,omitempty"`
	Todos       []TodoItem    `json:"todos,omitempty"`
	Children    []string      `json:"children,omitempty"`
	Metrics     RunMetrics    `json:"metrics"`
	Drift       []DriftSignal `json:"drift,omitempty"`
}

// Status constants.
const (
	StatusIdle    = "idle"
	StatusRunning = "running"
	StatusQueued  = "queued"
	StatusDone    = "done"
	StatusError   = "error"
	StatusBudget  = "budget"
	StatusStall   = "stall"
)

// Drift codes (phase 1).
const (
	DriftStallRepeatTools = "stall_repeat_tools"
	DriftStallToolErrors  = "stall_tool_errors"
	DriftBudgetPressure   = "budget_pressure"
	DriftTodoStale        = "todo_stale"
	DriftDoneOpenTodos    = "done_with_open_todos"
	DriftNoProgress       = "no_progress"
	DriftGoalToolMismatch = "goal_tool_mismatch"
)
