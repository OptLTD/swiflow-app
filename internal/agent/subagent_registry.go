package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/OptLTD/swiflow/internal/observe"
	"github.com/OptLTD/swiflow/internal/tenant"
	"github.com/OptLTD/swiflow/internal/tool"
	"github.com/OptLTD/swiflow/library/support"
)

const subagentStatusRunning = "running"

// SubagentStatusResponse is the JSON shape returned by subagent_status / subagent_wait.
type SubagentStatusResponse struct {
	ChildSession string            `json:"child_session"`
	Status       string            `json:"status"`
	Goal         string            `json:"goal,omitempty"`
	Todos        []subagentTodo    `json:"todos,omitempty"`
	LastAction   string            `json:"last_action,omitempty"`
	Metrics      tool.ChildMetrics `json:"metrics"`
	Summary      string            `json:"summary,omitempty"`
	Artifacts    []string          `json:"artifacts,omitempty"`
	Error        string            `json:"error,omitempty"`
}

type subagentTodo struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type subagentJob struct {
	parentSession   string
	childSession    string
	spawnToolCallID string
	goal            string
	maxRounds       int
	startedAt       time.Time
	lastAction      string
	onProgress      func(tool.ToolProgress)

	mu     sync.Mutex
	status string
	result tool.ChildResult
	done   chan struct{}

	cancel context.CancelFunc
}

// SubagentRegistry tracks async subagent runs for a parent session.
type SubagentRegistry struct {
	runner  *Runner
	publish EventPublisher
	todos   subagentTodoStore

	mu   sync.Mutex
	jobs map[string]*subagentJob
}

type subagentTodoStore interface {
	LoadTodos(ctx context.Context, sessionID string) (string, error)
}

// NewSubagentRegistry constructs a registry bound to runner and optional todo storage.
func NewSubagentRegistry(runner *Runner, publish EventPublisher, todos subagentTodoStore) *SubagentRegistry {
	return &SubagentRegistry{
		runner:  runner,
		publish: publish,
		todos:   todos,
		jobs:    map[string]*subagentJob{},
	}
}

// SpawnOpts configures an async child run.
type SpawnOpts struct {
	ParentSession   string
	SpawnToolCallID string
	AgentKey        string
	UserMessage     string
	Goal            string
	MaxRounds       int
	Tid             string
	OnProgress      func(tool.ToolProgress)
}

// Spawn starts a child run in the background and returns immediately.
func (reg *SubagentRegistry) Spawn(opts SpawnOpts) (string, error) {
	if reg == nil || reg.runner == nil {
		return "", fmt.Errorf("subagent_spawn unavailable")
	}
	parent := opts.ParentSession
	if parent == "" {
		parent = "unknown"
	}
	id := support.NewID()
	id = strings.ReplaceAll(id, "-", "")
	if len(id) > 12 {
		id = id[len(id)-12:]
	}
	childKey := fmt.Sprintf("sub-%s-%s", parent, id)
	agentKey := opts.AgentKey
	if agentKey == "" {
		agentKey = "default"
	}

	childCtx, cancel := context.WithCancel(context.Background())
	if opts.Tid != "" {
		childCtx = tenant.WithID(childCtx, opts.Tid)
	}
	job := &subagentJob{
		parentSession:   parent,
		childSession:    childKey,
		spawnToolCallID: opts.SpawnToolCallID,
		goal:            opts.Goal,
		maxRounds:       opts.MaxRounds,
		startedAt:       time.Now(),
		status:          subagentStatusRunning,
		done:            make(chan struct{}),
		cancel:          cancel,
		onProgress:      opts.OnProgress,
	}

	reg.mu.Lock()
	reg.jobs[childKey] = job
	reg.mu.Unlock()

	observe.DelegateStart(parent, childKey, opts.MaxRounds)
	go reg.runChild(childCtx, job, agentKey, opts.UserMessage)
	return childKey, nil
}

func (reg *SubagentRegistry) runChild(ctx context.Context, job *subagentJob, agentKey, userMessage string) {
	t0 := time.Now()
	var lastAssistant string
	emitProgress := func(p tool.ToolProgress) {
		content := p.Content
		job.mu.Lock()
		if content != "" {
			job.lastAction = content
		}
		job.mu.Unlock()
		if job.onProgress != nil {
			job.onProgress(p)
		}
		reg.publishParent(job.parentSession, Event{
			Type:    "subagent_progress",
			ID:      job.spawnToolCallID,
			Child:   job.childSession,
			Content: content,
			Name:    "subagent_spawn",
		})
		// Also emit tool_progress for spawn block backward compat in UI.
		reg.publishParent(job.parentSession, Event{
			Type:    "tool_progress",
			ID:      job.spawnToolCallID,
			Child:   job.childSession,
			Content: content,
			Name:    "subagent_spawn",
		})
	}

	result, err := reg.runner.RunChild(ctx, tool.ChildRunOpts{
		SessionID:       job.childSession,
		AgentKey:        agentKey,
		UserMessage:     userMessage,
		MaxRounds:       job.maxRounds,
		ParentSessionID: job.parentSession,
		OnProgress:      emitProgress,
	}, func(delta string) {
		lastAssistant += delta
	})
	observe.DelegateEnd(job.parentSession, job.childSession, time.Since(t0).Milliseconds(), err)

	summary := strings.TrimSpace(result.Summary)
	if summary == "" {
		summary = strings.TrimSpace(lastAssistant)
	}
	if summary == "" {
		summary = strings.TrimSpace(reg.runner.LastAssistantContent(context.Background(), job.childSession))
	}
	if summary == "" {
		summary = "(sub-agent finished with empty summary)"
	}
	status := result.Status
	if status == "" {
		status = "done"
	}
	if err != nil && result.Err == "" {
		result.Err = err.Error()
		if status == "" || status == "done" {
			status = "error"
		}
	}
	result.Status = status
	result.Summary = summary

	job.mu.Lock()
	job.status = status
	job.result = result
	close(job.done)
	job.mu.Unlock()

	reg.publishParent(job.parentSession, Event{
		Type:    "subagent_done",
		ID:      job.spawnToolCallID,
		Child:   job.childSession,
		Content: summary,
		Name:    "subagent_spawn",
		Result:  mustJSON(map[string]any{
			"child_session": job.childSession,
			"status":        status,
			"summary":       summary,
			"artifacts":     result.Artifacts,
			"metrics":       result.Metrics,
		}),
	})
}

func (reg *SubagentRegistry) publishParent(parent string, ev Event) {
	if reg.publish == nil || parent == "" {
		return
	}
	reg.publish.Publish(parent, ev)
}

// Status returns a lightweight snapshot for subagent_status.
func (reg *SubagentRegistry) Status(ctx context.Context, parentSession, childSession string) (SubagentStatusResponse, error) {
	job, err := reg.jobForParent(parentSession, childSession)
	if err != nil {
		return SubagentStatusResponse{}, err
	}
	return reg.buildResponse(ctx, job), nil
}

// Wait blocks until the child is terminal or timeout elapses.
func (reg *SubagentRegistry) Wait(ctx context.Context, parentSession, childSession string, timeout time.Duration) (SubagentStatusResponse, error) {
	if n := reg.CountRunning(parentSession); n > 1 {
		return SubagentStatusResponse{}, fmt.Errorf("multiple subagents still running; spawn or finish others first, use subagent_status meanwhile")
	}
	job, err := reg.jobForParent(parentSession, childSession)
	if err != nil {
		return SubagentStatusResponse{}, err
	}
	if reg.CountRunning(parentSession) == 1 && job.status != subagentStatusRunning {
		// terminal already
		return reg.buildResponse(ctx, job), nil
	}
	if job.status != subagentStatusRunning {
		return reg.buildResponse(ctx, job), nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return SubagentStatusResponse{}, ctx.Err()
	case <-job.done:
		return reg.buildResponse(ctx, job), nil
	case <-timer.C:
		resp := reg.buildResponse(ctx, job)
		resp.Status = "timeout"
		return resp, nil
	}
}

// CountRunning returns how many subagents are still running for a parent.
func (reg *SubagentRegistry) CountRunning(parentSession string) int {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	n := 0
	for _, job := range reg.jobs {
		if job.parentSession != parentSession {
			continue
		}
		job.mu.Lock()
		st := job.status
		job.mu.Unlock()
		if st == subagentStatusRunning {
			n++
		}
	}
	return n
}

// CancelParent cancels all running subagents owned by parentSession.
func (reg *SubagentRegistry) CancelParent(parentSession string) {
	reg.mu.Lock()
	jobs := make([]*subagentJob, 0)
	for _, job := range reg.jobs {
		if job.parentSession == parentSession {
			jobs = append(jobs, job)
		}
	}
	reg.mu.Unlock()
	for _, job := range jobs {
		if job.cancel != nil {
			job.cancel()
		}
	}
}

func (reg *SubagentRegistry) jobForParent(parentSession, childSession string) (*subagentJob, error) {
	reg.mu.Lock()
	job := reg.jobs[childSession]
	reg.mu.Unlock()
	if job == nil {
		return nil, fmt.Errorf("unknown child_session %q", childSession)
	}
	if job.parentSession != parentSession {
		return nil, fmt.Errorf("child_session %q does not belong to this session", childSession)
	}
	return job, nil
}

func (reg *SubagentRegistry) buildResponse(ctx context.Context, job *subagentJob) SubagentStatusResponse {
	job.mu.Lock()
	st := job.status
	res := job.result
	last := job.lastAction
	job.mu.Unlock()

	resp := SubagentStatusResponse{
		ChildSession: job.childSession,
		Status:       st,
		Goal:         job.goal,
		LastAction:   last,
		Metrics:      res.Metrics,
	}
	if st != subagentStatusRunning {
		resp.Summary = res.Summary
		resp.Artifacts = res.Artifacts
		resp.Error = res.Err
	}
	if resp.Metrics.WallMS == 0 {
		resp.Metrics.WallMS = time.Since(job.startedAt).Milliseconds()
	}
	if reg.todos != nil {
		raw, err := reg.todos.LoadTodos(ctx, job.childSession)
		if err == nil && raw != "" && raw != "[]" {
			var items []subagentTodo
			if json.Unmarshal([]byte(raw), &items) == nil {
				resp.Todos = items
			}
		}
	}
	return resp
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
