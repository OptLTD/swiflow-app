// Package agent implements the run loop. Spec §6.7, §7.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OptLTD/swiflow/internal/llmclient"
	"github.com/OptLTD/swiflow/internal/observe"
	"github.com/OptLTD/swiflow/internal/skill"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/tenant"
	"github.com/OptLTD/swiflow/internal/tool"
	"github.com/OptLTD/swiflow/library/support"
)

// TenantRoots holds per-tenant disk roots used by a run.
type TenantRoots struct {
	Workspace string
	Skills    string
	LightApps string
}

// maxRoundsDefault is only a safety fuse against runaway tool use.
const maxRoundsDefault = 32

const defaultToolTimeout = 120 * time.Second

const softBudgetNudge = "You are approaching the tool-call budget for this turn. Prefer finishing with a clear answer now if you already have enough information. Use another tool only if it is essential to complete the user's request."

const hardBudgetNudge = "You have hit the tool-call safety limit for this turn. Do not call tools. Summarize what you completed, what you found, what is blocked or unfinished, and the most useful next step for the user."

const stallNudge = "Progress has stalled (repeated tools or repeated failures). Do not call tools again. Tell the user what you tried, why it is stuck, and the best next step — ask them to continue if needed."

const childWrapUpNudge = "Sub-agent budget is running low. Finish the batch now: if the deliverable file (e.g. xlsx) exists, reply with ONLY its @/ path and row count — no more tools. If blocked, summarize what failed and stop."

const reflectExperienceNudge = "If this turn produced reusable handling logic (pitfalls / decision rules / recipes worth applying in other tasks), call experience_write for each distinct lesson (short ≤200 chars, outcome, 1-3 English tags). Do not save task diaries. Skip if nothing generalizable. If you applied past experiences, experience_use them (or used_ids)."


// maxLLMRetries caps how many times a single round retries after a transient
// LLM stall/network error before falling back to a graceful wrap-up.
const maxLLMRetries = 1

const continueHint = "I could not fully finish this turn. Tell me to continue and I will pick up from here."

// SoftAsyncPlaceholder is retained only for backward compatibility with older
// persisted messages / eval harnesses. The runtime no longer produces it: tool
// calls now run concurrently within a turn and the loop awaits them all.
const SoftAsyncPlaceholder = "工具运行中, 耗时较长, 执行完毕后补全执行结果"

// runResult is the terminal outcome of a run, surfaced to delegate_task parents.
type runResult struct {
	status    string // done|budget|stall|error|blocked
	summary   string
	artifacts []string
	rounds    int
	toolCalls int
	failures  int
}

type object = map[string]any

// Event is one event streamed to the client during a run.
type Event struct {
	Type      string `json:"type"` // delta|thinking|tool_call|tool_result|tool_progress|harness_warn|done|error|ui_request|user|queued
	Content   string `json:"content,omitempty"`
	Thinking  string `json:"thinking,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments object `json:"arguments,omitempty"`
	Result    string `json:"result,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
	Error     string `json:"error,omitempty"`
	Title     string `json:"title,omitempty"`
	Position  int    `json:"position,omitempty"` // queued position (1-based)
	// Child is the subagent session key carried by tool_progress events so the
	// parent UI can attach live progress (and later open the child) for a
	// delegate_task call.
	Child string `json:"child,omitempty"`
	// DurationMS is the measured execution time of a tool (tool_result events),
	// excluding time spent queued behind the concurrency limit.
	DurationMS int64 `json:"duration_ms,omitempty"`
}

// EventPublisher fans out events (typically server.SessionHub).
type EventPublisher interface {
	Publish(sessionID string, ev Event)
}

// RunnerDeps configures a Runner.
type RunnerDeps struct {
	Store  store.Store
	Tools  *tool.Registry
	Skills *skill.Catalog

	Workspace string
	// WorkspaceResolver returns the workspace root for a tenant id.
	// When nil, Workspace is used for all tenants.
	WorkspaceResolver func(tid string) string
	// RootsResolver returns workspace/skills/light-apps for a tenant.
	// When set, it takes precedence over WorkspaceResolver for workspace.
	RootsResolver func(tid string) TenantRoots
	// MCPOwns reports whether an mcp_* tool is owned by tid (cross-tenant hide).
	MCPOwns func(toolName, tid string) bool
	MaxHistoryMessages int
	// MaxContextChars is the soft character budget for in-memory LLM messages.
	// 0 disables proactive fitting; overflow still triggers emergency compact.
	// Negative values are treated as the package default (120_000).
	MaxContextChars int

	// Publish is optional; when set, every emit is also published for watchers.
	Publish EventPublisher

	// MaxConcurrentRuns caps in-flight runs per tenant; 0 = unlimited.
	MaxConcurrentRuns int
	// ToolTimeoutSec wraps each tool call; 0 = 120s.
	ToolTimeoutSec int
	// ToolTimeouts overrides per-tool timeouts (single source of truth), e.g.
	// {"content_extract": DocumentTimeout}. Falls back to ToolTimeoutSec.
	ToolTimeouts map[string]time.Duration
	// DisableThinking turns off model reasoning (GLM thinking:{type:disabled}).
	DisableThinking bool
}

type queuedMsg struct {
	AgentKey string
	Message  string
}

// RunOpts controls a single Run (used by subagents).
type RunOpts struct {
	MaxRounds int // 0 = default 32
	// MaxWallClock, when > 0, caps the run's wall clock independently of the
	// caller ctx (children get their own budget, decoupled from parent tool timeout).
	MaxWallClock time.Duration
	// AllowTools: if non-nil, only these tool names are offered.
	AllowTools map[string]bool
	// DenyTools: always excluded (e.g. delegate_task for children).
	DenyTools map[string]bool
	// ParentSession, when non-empty, marks this run's session as a subagent
	// child of the given parent session (hidden from the top-level list).
	ParentSession string
}

// Runner executes agent runs and enforces single-run-per-session.
type Runner struct {
	deps RunnerDeps

	mu        sync.Mutex
	busy      map[string]struct{}
	busyTid   map[string]int // tid -> in-flight run count
	cancels   map[string]context.CancelFunc
	queue     map[string][]queuedMsg
	provMu    sync.Mutex
	provCache map[string]llmclient.Provider

	subagents *SubagentRegistry
}

// NewRunner constructs a Runner.
func NewRunner(deps RunnerDeps) *Runner {
	r := &Runner{
		deps:      deps,
		busy:      map[string]struct{}{},
		busyTid:   map[string]int{},
		cancels:   map[string]context.CancelFunc{},
		queue:     map[string][]queuedMsg{},
		provCache: map[string]llmclient.Provider{},
	}
	if st, ok := deps.Store.(subagentTodoStore); ok {
		r.subagents = NewSubagentRegistry(r, deps.Publish, st)
	} else {
		r.subagents = NewSubagentRegistry(r, deps.Publish, nil)
	}
	return r
}

// ErrBusy is returned when a session already has a run in flight.
var ErrBusy = fmt.Errorf("session busy")

// ErrConcurrent is returned when the per-tenant concurrent-run gate is full.
var ErrConcurrent = fmt.Errorf("too many concurrent runs")

// IsBusy reports whether a session has a run in flight.
func (r *Runner) IsBusy(sessionID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.busy[sessionID]
	return ok
}

// AtCapacity reports whether the per-tenant concurrent-run gate is full for tid.
func (r *Runner) AtCapacity(tid string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	max := r.deps.MaxConcurrentRuns
	if max <= 0 {
		return false
	}
	if tid == "" {
		tid = tenant.DefaultID
	}
	return r.busyTid[tid] >= max
}

// QueueLen returns pending mid-run messages for a session.
func (r *Runner) QueueLen(sessionID string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.queue[sessionID])
}

// Enqueue adds a message to the session FIFO while a run is in flight.
// Returns 1-based queue position.
func (r *Runner) Enqueue(sessionID, agentKey, message string) (position int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queue[sessionID] = append(r.queue[sessionID], queuedMsg{AgentKey: agentKey, Message: message})
	position = len(r.queue[sessionID])
	observe.Queued(sessionID, position)
	return position
}

// Abort cancels an in-flight run for a session. Queue is retained.
func (r *Runner) Abort(sessionID string) bool {
	r.mu.Lock()
	cancel, ok := r.cancels[sessionID]
	r.mu.Unlock()
	if !ok {
		return false
	}
	observe.Abort(sessionID)
	cancel()
	if r.subagents != nil {
		r.subagents.CancelParent(sessionID)
	}
	return true
}

// InvalidateAll drops cached LLM providers so subsequent runs re-read config.
func (r *Runner) InvalidateAll() {
	r.provMu.Lock()
	r.provCache = map[string]llmclient.Provider{}
	r.provMu.Unlock()
}

// InvalidateProvider drops cached providers for name across all tenants.
func (r *Runner) InvalidateProvider(name string) {
	r.provMu.Lock()
	defer r.provMu.Unlock()
	suffix := "\x00" + name
	for k := range r.provCache {
		if k == name || strings.HasSuffix(k, suffix) {
			delete(r.provCache, k)
		}
	}
}

func provCacheKey(tid, name string) string {
	if tid == "" {
		tid = tenant.DefaultID
	}
	return tid + "\x00" + name
}

// Run executes one agent run with default options.
func (r *Runner) Run(ctx context.Context, sessionID, agentKey, userMessage string, onEvent func(Event)) error {
	return r.RunOpts(ctx, sessionID, agentKey, userMessage, onEvent, RunOpts{})
}

// RunOpts executes one agent run with options (subagent budgets / tool filters).
func (r *Runner) RunOpts(ctx context.Context, sessionID, agentKey, userMessage string, onEvent func(Event), opts RunOpts) error {
	_, err := r.run(ctx, sessionID, agentKey, userMessage, onEvent, opts)
	return err
}

// run is the core loop. It returns a structured runResult (surfaced to
// delegate_task parents) plus a hard error only when no usable work was produced.
// Tool calls in a turn run concurrently and are all awaited before the next LLM
// turn; messages are written only here (single writer) — no soft-async, no races.
func (r *Runner) run(ctx context.Context, sessionID, agentKey, userMessage string, onEvent func(Event), opts RunOpts) (runResult, error) {
	rounds := opts.MaxRounds
	if rounds <= 0 {
		rounds = maxRoundsDefault
	}
	childRun := isChildRun(opts)
	res := runResult{status: "done"}

	publisher := func(ev Event) {
		emit(onEvent, ev)
		if r.deps.Publish != nil {
			r.deps.Publish.Publish(sessionID, ev)
		}
	}

	// Claim the session + optional per-tenant concurrent gate.
	runTid := tenant.ID(ctx)
	r.mu.Lock()
	if _, busy := r.busy[sessionID]; busy {
		r.mu.Unlock()
		observe.BusyReject(sessionID)
		publisher(Event{Type: "error", Error: "session busy"})
		return runResult{status: "error"}, ErrBusy
	}
	if max := r.deps.MaxConcurrentRuns; max > 0 && r.busyTid[runTid] >= max {
		n := r.busyTid[runTid]
		r.mu.Unlock()
		observe.ConcurrentReject(sessionID, n, max)
		publisher(Event{Type: "error", Error: "too many concurrent runs"})
		return runResult{status: "error"}, ErrConcurrent
	}
	runCtx, cancel := context.WithCancel(ctx)
	if opts.MaxWallClock > 0 {
		runCtx, cancel = context.WithTimeout(runCtx, opts.MaxWallClock)
	}
	r.busy[sessionID] = struct{}{}
	r.busyTid[runTid]++
	r.cancels[sessionID] = cancel
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.busy, sessionID)
		delete(r.cancels, sessionID)
		if r.busyTid[runTid] > 0 {
			r.busyTid[runTid]--
		}
		if r.busyTid[runTid] == 0 {
			delete(r.busyTid, runTid)
		}
		r.mu.Unlock()
		cancel()
		r.drainQueue(sessionID)
	}()

	st := r.deps.Store

	sess, err := st.GetSessionByID(runCtx, sessionID)
	if err != nil {
		if agentKey == "" {
			agentKey = "default"
		}
		sess = &store.Session{ID: sessionID, Agent: agentKey, Parent: opts.ParentSession}
		if cerr := st.CreateSession(runCtx, sess); cerr != nil {
			sess, err = st.GetSessionByID(runCtx, sessionID)
			if err != nil {
				publisher(Event{Type: "error", Error: "session unavailable"})
				return runResult{status: "error"}, err
			}
		}
	} else {
		agentKey = sess.Agent
	}

	tid := sess.Tid
	if tid == "" {
		tid = tenant.ID(runCtx)
	}
	runCtx = tenant.WithID(runCtx, tid)
	roots := r.resolveRoots(tid)
	ws := roots.Workspace

	ag, err := st.GetAgentByKey(runCtx, agentKey)
	if err != nil {
		publisher(Event{Type: "error", Error: "agent not found: " + agentKey})
		return runResult{status: "error"}, err
	}

	if ag.TxtModel == "" {
		publisher(Event{Type: "error", Error: "agent has no txt_model"})
		return runResult{status: "error"}, fmt.Errorf("agent %q has no txt_model", agentKey)
	}
	prov, model, err := r.resolveTxtModel(runCtx, ag.TxtModel)
	if err != nil {
		publisher(Event{Type: "error", Error: err.Error()})
		return runResult{status: "error"}, err
	}
	slog.Info("agent.run_model", "session", sessionID, "provider", ag.TxtModel, "chat_model", model)
	observe.RunStart(sessionID, childRun, rounds, model)

	history, err := st.ListMessages(runCtx, sessionID)
	if err != nil {
		publisher(Event{Type: "error", Error: "load history failed"})
		return runResult{status: "error"}, err
	}
	history = truncateHistory(history, r.deps.MaxHistoryMessages)

	userMsg := store.Message{ID: support.NewID(), Role: "user", Content: userMessage}
	if _, err := st.AppendMessage(runCtx, sessionID, userMsg); err != nil {
		slog.Error("persist user message", "error", err)
	}

	// Persist tool pairing even if the run context is cancelled mid-loop.
	persistCtx := context.WithoutCancel(runCtx)

	llmMsgs := []llmclient.Message{{Role: "system", Content: r.buildSystem(runCtx, sessionID, ag, ws, roots.Skills)}}
	for _, m := range history {
		llmMsgs = append(llmMsgs, toLLM(m))
	}
	llmMsgs = append(llmMsgs, llmclient.Message{Role: "user", Content: userMessage})

	// Async calibration: preference-like follow-ups append into working charter.
	if len(history) > 0 && looksLikeCorrection(userMessage) {
		appendCharterPrinciple(persistCtx, st, sessionID, ag, userMessage, "correction")
	}

	toolDefs := r.toolDefinitions(runCtx, tid, opts)
	firstUser := firstUserMessage(history, userMessage)

	// Deterministic batch routing (once): if the user attached >= threshold files,
	// the main agent must hand the whole batch to a child — heavy tools are gated
	// here and a single delegate_task is required. No runtime cost probing.
	if paths, forced := shouldForceBatchDelegate(userMessage, childRun); forced {
		denyContentExtract(&opts)
		toolDefs = r.toolDefinitions(runCtx, tid, opts)
		llmMsgs = append(llmMsgs, llmclient.Message{Role: "user", Content: batchDelegateNudge(paths)})
		slog.Info("agent.batch_delegate_forced", "session", sessionID, "paths", len(paths))
	}

	var (
		repeat            int
		lastKey           string
		consecErrors      int
		softNudged        bool
		forceWrapUp       bool
		wrapReason        string
		toolsUsed         bool
		childNudged       bool
		llmRetries        int
		contextCompacts   int
		reflectUsed       int
		reflectPending    bool
		reflectAllowlist  bool
		claimReflectDone  bool
		stallReflectDone  bool
		experienceWritten bool
		experienceNudged  bool
	)

	contextBudget := r.deps.MaxContextChars
	if contextBudget < 0 {
		contextBudget = defaultMaxContextChars
	}

	for round := 0; round < rounds; {
		observe.RoundStart(sessionID, round)
		roundTools := toolDefs
		roundT0 := time.Now()

		switch {
		case reflectAllowlist:
			roundTools = reflectToolDefs(toolDefs, opts.DenyTools)
			reflectAllowlist = false
		case forceWrapUp || round == rounds-1:
			if forceWrapUp && wrapReason == "stall" && !stallReflectDone && reflectUsed < maxReflectPerRun {
				stallReflectDone = true
				reflectUsed++
				reflectPending = true
				observe.ReflectEnter(sessionID, round, "stall")
				llmMsgs = append(llmMsgs, llmclient.Message{Role: "user", Content: reflectNudge})
				roundTools = reflectToolDefs(toolDefs, opts.DenyTools)
				forceWrapUp = false
				wrapReason = ""
			} else {
				roundTools = nil
				nudge := hardBudgetNudge
				if forceWrapUp && wrapReason == "stall" {
					nudge = stallNudge
					res.status = "stall"
				} else {
					res.status = "budget"
				}
				forceWrapUp = false
				wrapReason = ""
				llmMsgs = append(llmMsgs, llmclient.Message{Role: "user", Content: nudge})
			}
		case !softNudged && round >= rounds*3/4:
			softNudged = true
			llmMsgs = append(llmMsgs, llmclient.Message{Role: "user", Content: softBudgetNudge})
		case childRun && !childNudged && round >= rounds/2:
			childNudged = true
			llmMsgs = append(llmMsgs, llmclient.Message{Role: "user", Content: childWrapUpNudge})
		}

		if contextBudget > 0 {
			llmMsgs = fitMessagesToBudget(llmMsgs, contextBudget, contextFitOpts{})
		}

		req := llmclient.ChatRequest{Model: model, Messages: llmMsgs, Tools: roundTools}
		ctxRemain := ctxRemaining(runCtx)
		slog.Info("agent.llm_wait", "session", sessionID, "round", round, "model", model, "ctx_remain", ctxRemain)
		llmDone := make(chan struct{})
		go r.llmWaitHeartbeat(runCtx, llmDone, sessionID, round, model, roundT0)
		resp, err := r.streamRound(runCtx, prov, req, publisher)
		close(llmDone)
		llmMS := time.Since(roundT0).Milliseconds()
		if err != nil {
			slog.Warn("agent.llm_error", "session", sessionID, "round", round, "model", model, "ms", llmMS, "error", err.Error())
			// Context overflow: compact in-memory messages and retry same round.
			if runCtx.Err() == nil && llmclient.IsContextOverflow(err) && contextCompacts < maxContextCompacts {
				contextCompacts++
				emergency := contextBudget
				if emergency <= 0 {
					emergency = defaultMaxContextChars
				}
				before := estimateChars(llmMsgs)
				llmMsgs = fitMessagesToBudget(llmMsgs, emergency, contextFitOpts{Aggressive: true, KeepTail: 8})
				slog.Info("agent.context_compact",
					"session", sessionID, "round", round, "attempt", contextCompacts,
					"chars_before", before, "chars_after", estimateChars(llmMsgs), "budget", emergency)
				continue
			}
			// Transient stall/network error: retry the same round once.
			if runCtx.Err() == nil && llmRetries < maxLLMRetries {
				llmRetries++
				slog.Info("agent.llm_retry", "session", sessionID, "round", round, "attempt", llmRetries, "error", err.Error())
				continue
			}
			// Terminal. If we produced tool work, exit cleanly with a summary
			// (emit done) instead of dropping results — sub-agent exits normally.
			if toolsUsed {
				summary := buildLLMErrorSummary(err)
				if _, aerr := st.AppendMessage(persistCtx, sessionID, store.Message{
					ID: support.NewID(), Role: "assistant", Content: summary,
				}); aerr != nil {
					slog.Error("persist llm-error wrap-up", "error", aerr)
				}
				publisher(Event{Type: "delta", Content: summary})
				publisher(Event{Type: "done"})
				observe.RunEnd(sessionID, "error", round, rounds, childRun)
				res.status = "error"
				res.summary = summary
				return res, nil
			}
			publisher(Event{Type: "error", Error: err.Error()})
			res.status = "error"
			observe.RunEnd(sessionID, "error", round, rounds, childRun)
			return res, err
		}
		llmRetries = 0
		slog.Info("agent.llm_done", "session", sessionID, "round", round, "model", model, "ms", llmMS, "had_tools", len(resp.ToolCalls) > 0)

		if len(resp.ToolCalls) > 0 && len(roundTools) > 0 {
			toolsUsed = true
			res.rounds = round + 1
			observe.RoundEnd(sessionID, round, true)
			if reflectPending {
				outcome := "fix"
				for _, tc := range resp.ToolCalls {
					if tc.Name == "clarify" {
						outcome = "clarify"
						break
					}
				}
				observe.ReflectExit(sessionID, round, outcome)
				reflectPending = false
			}
			key := toolCallKey(resp.ToolCalls)
			if key == lastKey {
				repeat++
				if repeat >= 3 {
					forceWrapUp = true
					wrapReason = "stall"
					observe.Stall(sessionID, "repeat_tool_calls", round)
					round++
					continue
				}
			} else {
				repeat = 0
				lastKey = key
			}

			assistantMsg := store.Message{
				ID:        support.NewID(),
				Role:      "assistant",
				Content:   resp.Content,
				Thinking:  resp.Thinking,
				ToolCalls: toStoreToolCalls(resp.ToolCalls),
			}
			if _, err := st.AppendMessage(persistCtx, sessionID, assistantMsg); err != nil {
				slog.Error("persist assistant (tool) message", "error", err)
			}
			llmMsgs = append(llmMsgs, toLLM(assistantMsg))

			outcomes := r.executeToolCalls(runCtx, persistCtx, sessionID, agentKey, tid, roots, resp.ToolCalls, publisher)
			res.toolCalls += len(outcomes)
			for _, o := range outcomes {
				if o.tc.Name == "experience_write" && !o.isErr {
					experienceWritten = true
				}
				if o.isErr {
					consecErrors++
					res.failures++
				} else {
					consecErrors = 0
				}
				if p := artifactPath(o.tc); p != "" {
					res.artifacts = appendUnique(res.artifacts, p)
				}
				llmMsgs = append(llmMsgs, llmclient.Message{
					Role:       "tool",
					Content:    o.result,
					ToolCallID: o.tc.ID,
					ToolName:   o.tc.Name,
				})
			}
			if consecErrors >= 3 {
				forceWrapUp = true
				wrapReason = "stall"
				observe.Stall(sessionID, "consecutive_tool_errors", round)
			}
			round++
			continue
		}

		// Model produced a no-tool reply: final answer, or reflect checkpoint.
		observe.RoundEnd(sessionID, round, false)
		content := resp.Content
		if strings.TrimSpace(content) == "" && res.status != "done" {
			content = continueHint
		}
		assistantMsg := store.Message{
			ID:       support.NewID(),
			Role:     "assistant",
			Content:  content,
			Thinking: resp.Thinking,
		}
		if _, err := st.AppendMessage(runCtx, sessionID, assistantMsg); err != nil {
			slog.Error("persist assistant message", "error", err)
		}
		llmMsgs = append(llmMsgs, toLLM(assistantMsg))
		if content != "" && content != resp.Content {
			publisher(Event{Type: "delta", Content: content})
		}

		if reflectPending {
			observe.ReflectExit(sessionID, round, "ship")
			reflectPending = false
		} else if isSignificantRun(toolsUsed, sessionHasOpenTodos(runCtx, st, sessionID)) &&
			!claimReflectDone && reflectUsed < maxReflectPerRun && round < rounds-1 {
			observe.ClaimRejected(sessionID, "significant_run_needs_reflect")
			observe.ReflectEnter(sessionID, round, "claim_done")
			claimReflectDone = true
			reflectUsed++
			reflectPending = true
			reflectAllowlist = true
			llmMsgs = append(llmMsgs, llmclient.Message{Role: "user", Content: reflectNudge})
			if toolsUsed && !experienceWritten && !experienceNudged {
				experienceNudged = true
				llmMsgs = append(llmMsgs, llmclient.Message{Role: "user", Content: reflectExperienceNudge})
			}
			round++
			continue
		}

		title := ""
		if sess.Title == "" {
			title = titleFromMessage(firstUser)
			if err := st.UpdateSessionTitle(runCtx, sessionID, title); err != nil {
				slog.Error("set session title", "error", err)
			}
		}
		if res.status == "" {
			res.status = "done"
		}
		publisher(Event{Type: "done", Title: title})
		observe.RunEnd(sessionID, res.status, round, rounds, childRun)
		res.summary = content
		for _, p := range extractAtPaths(content) {
			res.artifacts = appendUnique(res.artifacts, p)
		}
		return res, nil
	}

	// Budget exhausted without a natural stop.
	if _, err := st.AppendMessage(runCtx, sessionID, store.Message{
		ID: support.NewID(), Role: "assistant", Content: continueHint,
	}); err != nil {
		slog.Error("persist wrap-up message", "error", err)
	}
	publisher(Event{Type: "delta", Content: continueHint})
	publisher(Event{Type: "done"})
	observe.RunEnd(sessionID, "budget", rounds-1, rounds, childRun)
	res.status = "budget"
	res.summary = continueHint
	return res, nil
}

func (r *Runner) drainQueue(sessionID string) {
	r.mu.Lock()
	q := r.queue[sessionID]
	if len(q) == 0 {
		r.mu.Unlock()
		return
	}
	next := q[0]
	r.queue[sessionID] = q[1:]
	if len(r.queue[sessionID]) == 0 {
		delete(r.queue, sessionID)
	}
	r.mu.Unlock()

	go func() {
		time.Sleep(80 * time.Millisecond)
		ctx := context.Background()
		emit := func(ev Event) {
			if r.deps.Publish != nil {
				r.deps.Publish.Publish(sessionID, ev)
			}
		}
		if r.deps.Publish != nil {
			r.deps.Publish.Publish(sessionID, Event{Type: "user", Content: next.Message})
		}
		if err := r.Run(ctx, sessionID, next.AgentKey, next.Message, emit); err != nil {
			slog.Warn("queue drain run", "session", sessionID, "error", err)
		}
	}()
}

func filterTools(defs []llmclient.ToolDef, opts RunOpts) []llmclient.ToolDef {
	if opts.AllowTools == nil && len(opts.DenyTools) == 0 {
		return defs
	}
	out := make([]llmclient.ToolDef, 0, len(defs))
	for _, d := range defs {
		if opts.DenyTools[d.Name] {
			continue
		}
		if opts.AllowTools != nil && !opts.AllowTools[d.Name] {
			continue
		}
		out = append(out, d)
	}
	return out
}

func (r *Runner) resolveRoots(tid string) TenantRoots {
	if r.deps.RootsResolver != nil {
		roots := r.deps.RootsResolver(tid)
		if roots.Workspace == "" {
			roots.Workspace = r.deps.Workspace
		}
		return roots
	}
	ws := r.deps.Workspace
	if r.deps.WorkspaceResolver != nil {
		ws = r.deps.WorkspaceResolver(tid)
	}
	return TenantRoots{Workspace: ws}
}

func (r *Runner) toolDefinitions(ctx context.Context, tid string, opts RunOpts) []llmclient.ToolDef {
	if r.deps.Tools == nil {
		return nil
	}
	defs := r.deps.Tools.Definitions()
	out := make([]llmclient.ToolDef, 0, len(defs))
	for _, d := range defs {
		if strings.HasPrefix(d.Name, "mcp_") {
			if r.deps.MCPOwns == nil || !r.deps.MCPOwns(d.Name, tid) {
				continue
			}
		}
		if r.deps.Store != nil && !r.deps.Store.ToolEnabled(ctx, d.Name) {
			continue
		}
		out = append(out, d)
	}
	return filterTools(out, opts)
}

// resolveTxtModel looks up llm_provider by name (agent.txt_model) and returns
// the chat client plus the model id defined on that provider row.
func (r *Runner) resolveTxtModel(ctx context.Context, name string) (llmclient.Provider, string, error) {
	apiBase, apiKey, model, err := r.deps.Store.ProviderCreds(ctx, name)
	if err != nil {
		return nil, "", err
	}
	key := provCacheKey(tenant.ID(ctx), name)
	r.provMu.Lock()
	if p, ok := r.provCache[key]; ok {
		r.provMu.Unlock()
		return p, model, nil
	}
	r.provMu.Unlock()
	p := llmclient.NewAdaptiveProvider(name, apiBase, apiKey, "")
	p.SetDisableThinking(r.deps.DisableThinking)
	r.provMu.Lock()
	if existing, ok := r.provCache[key]; ok {
		r.provMu.Unlock()
		return existing, model, nil
	}
	r.provCache[key] = p
	r.provMu.Unlock()
	return p, model, nil
}

func (r *Runner) llmWaitHeartbeat(ctx context.Context, done <-chan struct{}, sessionID string, round int, model string, start time.Time) {
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-tick.C:
			observe.LLMStillWaiting(sessionID, round, model, time.Since(start), ctxRemaining(ctx))
		}
	}
}

func ctxRemaining(ctx context.Context) string {
	if dl, ok := ctx.Deadline(); ok {
		return time.Until(dl).Round(time.Second).String()
	}
	return ""
}

func (r *Runner) streamRound(ctx context.Context, p llmclient.Provider, req llmclient.ChatRequest, onEvent func(Event)) (*llmclient.ChatResponse, error) {
	return p.ChatStream(ctx, req, func(c llmclient.StreamChunk) {
		if c.Thinking != "" {
			emit(onEvent, Event{Type: "thinking", Content: c.Thinking})
		}
		if c.Content != "" {
			emit(onEvent, Event{Type: "delta", Content: c.Content})
		}
	})
}

func (r *Runner) buildSystem(ctx context.Context, sessionID string, ag *store.Agent, workspace, skillsDir string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are Swiflow agent %s.", ag.Key)
	if ag.Prompt != "" {
		b.WriteString("\n\n")
		b.WriteString(ag.Prompt)
	}
	charter := strings.TrimSpace(ag.Charter)
	emptySeed := charter == ""
	if emptySeed {
		charter = defaultCharterSeed
	}
	b.WriteString("\n\n## Ways of working\n")
	b.WriteString(charter)
	observe.CharterInjected(sessionID, ag.Key, len(charter), emptySeed)

	if workspace != "" {
		b.WriteString("\n\n## Workspace\nWorkspace root: ")
		b.WriteString(workspace)
		b.WriteString(". File tools are restricted to it.\n")
		b.WriteString("User messages may cite workspace files as @/relative/path (e.g. @/notes.txt, @/docs/a.md). ")
		b.WriteString("@/ means the workspace root. Attached uploads appear in a block between [UPLOAD FILES START] and [UPLOAD FILES END] (one @/ path per line). ")
		b.WriteString("User uploads land under @/uploads/… with unique paths; treat those as immutable originals so chat history keeps working.\n")
		b.WriteString("Keep the workspace tidy: never drop new deliverables (reports, spreadsheets, rewritten copies, notes, scripts) in the workspace root. ")
		b.WriteString("For each topic/task, create or reuse one short slug folder (lowercase, hyphens; e.g. @/q3-sales-recon/, @/meeting-notes-0812/) and put all related outputs under it. ")
		b.WriteString("Copy out of @/uploads/ into that topic folder when you need editable working copies — do not move or delete @/uploads/ originals. ")
		b.WriteString("If this session already has a topic folder, keep using it instead of starting another. ")
		b.WriteString("File tools accept both workspace-relative paths and @/… (equivalent). Prefer passing the path as given; do not invent a literal \"@\" directory. ")
		b.WriteString("When the user attaches or mentions @/…, resolve it to that relative path and use fs_* / content_extract / other file tools on it — do not treat @/ as a URL or package alias. ")
		b.WriteString("When you refer to workspace files in replies, prefer the same @/ form.")
	}
	disabled := map[string]bool{}
	if r.deps.Skills != nil {
		if r.deps.Store != nil {
			if list, err := r.deps.Store.DisabledSkills(ctx); err == nil {
				for _, s := range list {
					disabled[s] = true
				}
			}
		}
		cat := r.deps.Skills
		if skillsDir != "" {
			cat = cat.ForUserDir(skillsDir)
		}
		summary := cat.Summary(ctx, disabled)
		if summary != "" {
			b.WriteString("\n\n## Skills\n\n")
			b.WriteString(summary)
		}
	}
	b.WriteString("\n\n## When to stop\n")
	b.WriteString("Primary goal: satisfy the user's request, then answer in natural language without more tools.\n")
	b.WriteString("- Stop when you have enough information to answer clearly.\n")
	b.WriteString("- If blocked, looping, or results are not helping the goal, stop early: explain what failed and what the user should do next.\n")
	b.WriteString("- Do not keep calling tools hoping for a different outcome. Prefer short paths toward the goal over exhaustive exploration.\n")
	b.WriteString("\n\n## Scheduling\n")
	b.WriteString("Use schedule_run to re-invoke the agent in the current chat after a delay (delay_seconds + message as a new user turn). ")
	b.WriteString("Use schedule_create for recurring cron jobs (@hourly, 0 9 * * *, @every 5m).")
	b.WriteString("\n\n## Skill authoring\n")
	b.WriteString("Use skill_manage to save reusable workflows: action create with full SKILL.md content for new skills; ")
	b.WriteString("action patch with old_string/new_string for small edits (preferred). User skills override built-ins by slug. ")
	b.WriteString("For skill *drafts* that need human review, use skill_draft instead of writing directly when unsure.")
	b.WriteString("\n\n## Task tracking\n")
	b.WriteString("For multi-step work, maintain a checklist with todo_write / todo_read. Prefer marking items done before claiming the overall goal is finished. ")
	b.WriteString("When verification matters, run tests via exec (if enabled) before the final answer.")
	b.WriteString("\n\n## Parallel tools\n")
	b.WriteString("You may request several independent tool calls in one turn; they run concurrently and all of their real results are returned before your next turn. ")
	b.WriteString("Do not invent tool output, and never say you will continue later and stop: either keep calling tools for remaining work, or produce the final deliverable (e.g. Excel) with completed results only.")
	b.WriteString("\n\n## Subagents\n")
	b.WriteString("Use subagent_spawn for batch work (own session + round budget). List every remaining @/ path inside goal; the child chooses tools itself.\n")
	b.WriteString("Tools: subagent_spawn (start, returns immediately), subagent_status (non-blocking progress), subagent_wait (blocking collect — use sparingly).\n")
	b.WriteString("Rules: (1) Spawn before wait — never subagent_wait while you still need to subagent_spawn. ")
	b.WriteString("(2) When multiple subagents are running, use subagent_status only; wait is rejected unless exactly one is running. ")
	b.WriteString("(3) Do not mix subagent_spawn and subagent_wait in the same tool-call batch. ")
	b.WriteString("(4) Maintain a parent checklist with todo_write; read child progress via subagent_status (child maintains its own todos). ")
	b.WriteString("(5) Claim completion only after terminal status (done|budget|stall|error|blocked|timeout).\n")
	b.WriteString("When many files or a table/Excel deliverable is obvious, spawn early. ")
	b.WriteString("When ≥3 @/ files are attached for a table, content_extract is disabled on the main agent: pick sensible columns and subagent_spawn the whole batch once.")
	b.WriteString("\n\n## Clarify\n")
	b.WriteString("When you need a user choice, confirmation, or missing info before continuing, call clarify. ")
	b.WriteString("Do not invent answers; wait for the tool result. Prefer short options when choices are discrete.")
	b.WriteString("\n\n## Learning & memory\n")
	b.WriteString("Experiences are reusable handling logic (pitfalls, decision rules, recipes) — not one entry per task. ")
	b.WriteString("Call experience_write whenever you discover a lesson worth reusing elsewhere; a single task may yield several distinct lessons, or none. ")
	b.WriteString("Each write: one short sentence, outcome, 1-3 English tags. Never save a task diary or product changelog. ")
	b.WriteString("Before complex work, call experience_list (sorted by weight) and apply relevant lessons; when you do, call experience_use on those ids (or pass used_ids on experience_write) so useful lessons rank higher next time.\n")
	b.WriteString("When the same pattern appears in multiple experiences, promote it: use skill_manage for reusable workflows, or refine the Ways of working charter via clear user corrections.")
	return b.String()
}

func emit(onEvent func(Event), ev Event) {
	if onEvent == nil {
		return
	}
	defer func() { recover() }()
	onEvent(ev)
}

func toLLM(m store.Message) llmclient.Message {
	out := llmclient.Message{
		Role:       m.Role,
		Content:    m.Content,
		Thinking:   m.Thinking,
		ToolCallID: m.ToolCallId,
		ToolName:   m.ToolName,
	}
	if len(m.ToolCalls) > 0 {
		out.ToolCalls = make([]llmclient.ToolCall, len(m.ToolCalls))
		for i, tc := range m.ToolCalls {
			out.ToolCalls[i] = llmclient.ToolCall{
				ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments,
			}
		}
	}
	return out
}

func toStoreToolCalls(tcs []llmclient.ToolCall) []store.ToolCall {
	if len(tcs) == 0 {
		return nil
	}
	out := make([]store.ToolCall, len(tcs))
	for i, tc := range tcs {
		out[i] = store.ToolCall{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments}
	}
	return out
}

func toolCallKey(tcs []llmclient.ToolCall) string {
	if len(tcs) == 0 {
		return ""
	}
	sorted := append([]llmclient.ToolCall(nil), tcs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	b, _ := json.Marshal(sorted)
	return string(b)
}

func truncateToolResult(s string) string {
	s = support.SanitizeUTF8(s)
	const max = 4000
	if len(s) <= max {
		return s
	}
	// Truncate on rune boundary so Postgres UTF8 columns stay valid.
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "\n...[truncated]"
}

// formatToolError keeps command/tool output (stdout/stderr) when execution fails.
func formatToolError(output string, err error) string {
	msg := "error: " + err.Error()
	output = strings.TrimSpace(output)
	if output == "" {
		return msg
	}
	return output + "\n" + msg
}

func firstUserMessage(history []store.Message, current string) string {
	for _, m := range history {
		if m.Role == "user" && strings.TrimSpace(m.Content) != "" {
			return m.Content
		}
	}
	return current
}

func truncateHistory(msgs []store.Message, max int) []store.Message {
	if max > 0 && len(msgs) > max {
		msgs = msgs[len(msgs)-max:]
		// Drop leading orphan tool messages created by a mid-pair cut.
		for len(msgs) > 0 && msgs[0].Role == "tool" {
			msgs = msgs[1:]
		}
	}
	return sanitizeToolHistory(msgs)
}

// sanitizeToolHistory ensures every assistant tool_calls message is followed by
// a tool result for each tool_call_id (required by OpenAI-compatible APIs).
func sanitizeToolHistory(msgs []store.Message) []store.Message {
	if len(msgs) == 0 {
		return msgs
	}
	out := make([]store.Message, 0, len(msgs))
	for i := 0; i < len(msgs); {
		m := msgs[i]
		if m.Role == "tool" {
			// Orphan tool result without a preceding assistant tool_calls.
			i++
			continue
		}
		if m.Role != "assistant" || len(m.ToolCalls) == 0 {
			out = append(out, m)
			i++
			continue
		}

		needed := make(map[string]store.ToolCall, len(m.ToolCalls))
		uniqCalls := make([]store.ToolCall, 0, len(m.ToolCalls))
		for _, tc := range m.ToolCalls {
			if tc.ID == "" {
				continue
			}
			if _, ok := needed[tc.ID]; ok {
				continue
			}
			needed[tc.ID] = tc
			uniqCalls = append(uniqCalls, tc)
		}
		m.ToolCalls = uniqCalls
		found := make(map[string]store.Message, len(needed))
		j := i + 1
		for j < len(msgs) && msgs[j].Role == "tool" {
			id := msgs[j].ToolCallId
			if _, ok := needed[id]; ok {
				if _, have := found[id]; !have {
					found[id] = msgs[j]
				}
			}
			j++
		}

		out = append(out, m)
		for _, tc := range uniqCalls {
			if tm, ok := found[tc.ID]; ok {
				out = append(out, tm)
				continue
			}
			out = append(out, store.Message{
				Role:       "tool",
				Content:    "error: tool call interrupted or result missing",
				ToolCallId: tc.ID,
				ToolName:   tc.Name,
			})
		}
		i = j
	}
	return out
}

func titleFromMessage(msg string) string {
	msg = strings.ReplaceAll(msg, "\n", " ")
	msg = strings.TrimSpace(msg)
	if len(msg) > 60 {
		return msg[:60] + "…"
	}
	return msg
}

// buildLLMErrorSummary produces an honest wrap-up when the model call fails
// after tool work already ran. It does not claim success (the deliverable may be
// missing); completed tool results remain persisted in the session.
func buildLLMErrorSummary(cause error) string {
	if llmclient.IsContextOverflow(cause) {
		msg := "未能完成本次任务：上下文过长，已尝试压缩仍失败"
		if cause != nil {
			msg += "（" + cause.Error() + "）"
		}
		msg += "。已完成的工具结果已保存在会话中；请开新会话或让我继续（continue）并尽量缩小范围。"
		return msg
	}
	msg := "未能完成本次任务：调用模型时出错"
	if cause != nil {
		msg += "（" + cause.Error() + "）"
	}
	msg += "。已完成的工具结果已保存在会话中；如需继续整理，请让我继续（continue）。"
	return msg
}

// RunChild implements tool.ChildRunner for delegate_task. It returns a structured
// result so the parent gets status/artifacts/metrics (context stays isolated).
func (r *Runner) RunChild(ctx context.Context, opts tool.ChildRunOpts, onDelta func(string)) (tool.ChildResult, error) {
	allow := map[string]bool(nil)
	if opts.AllowTools != nil {
		allow = make(map[string]bool, len(opts.AllowTools))
		for _, n := range opts.AllowTools {
			allow[n] = true
		}
	}
	res, err := r.run(ctx, opts.SessionID, opts.AgentKey, opts.UserMessage, func(ev Event) {
		if ev.Type == "delta" && onDelta != nil && ev.Content != "" {
			onDelta(ev.Content)
		}
		if opts.OnProgress == nil {
			return
		}
		switch ev.Type {
		case "tool_call":
			if ev.Name != "" {
				opts.OnProgress(tool.ToolProgress{Child: opts.SessionID, Content: ev.Name})
			}
		case "delta":
			if s := strings.TrimSpace(ev.Content); s != "" {
				opts.OnProgress(tool.ToolProgress{Child: opts.SessionID, Content: progressSnippet(s)})
			}
		}
	}, RunOpts{
		MaxRounds:     opts.MaxRounds,
		MaxWallClock:  opts.MaxWallClock,
		AllowTools:    allow,
		DenyTools: map[string]bool{
			"subagent_spawn":  true,
			"subagent_status": true,
			"subagent_wait":   true,
			"clarify":         true,
		},
		ParentSession: opts.ParentSessionID,
	})
	cr := tool.ChildResult{
		Status:    res.status,
		Summary:   res.summary,
		Artifacts: res.artifacts,
		Metrics: tool.ChildMetrics{
			Rounds:    res.rounds,
			ToolCalls: res.toolCalls,
			Failures:  res.failures,
		},
	}
	if err != nil {
		cr.Err = err.Error()
		if cr.Status == "" || cr.Status == "done" {
			cr.Status = "error"
		}
	}
	return cr, err
}

// progressSnippet collapses a streamed text chunk into a short single-line hint
// suitable for the parent UI's live progress label.
func progressSnippet(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	const max = 120
	if r := []rune(s); len(r) > max {
		s = string(r[:max]) + "…"
	}
	return s
}

// LastAssistantContent returns the latest non-empty assistant message in a session.
func (r *Runner) LastAssistantContent(ctx context.Context, sessionID string) string {
	if r.deps.Store == nil || sessionID == "" {
		return ""
	}
	msgs, err := r.deps.Store.ListMessages(ctx, sessionID)
	if err != nil {
		return ""
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "assistant" && strings.TrimSpace(msgs[i].Content) != "" {
			return msgs[i].Content
		}
	}
	return ""
}

// SetProvider injects a cached LLM provider (tests / overrides).
func (r *Runner) SetProvider(name string, p llmclient.Provider) {
	r.provMu.Lock()
	defer r.provMu.Unlock()
	r.provCache[provCacheKey(tenant.DefaultID, name)] = p
}

// SpawnSubagent implements tool.SubagentBackend.
func (r *Runner) SpawnSubagent(ctx context.Context, rc tool.RunContext, goal, contextHint string, maxRounds int) (string, error) {
	if r.subagents == nil {
		return "", fmt.Errorf("subagent_spawn unavailable")
	}
	parent := rc.SessionID
	if parent == "" {
		parent = "unknown"
	}
	agentKey := rc.Agent
	if agentKey == "" {
		agentKey = "default"
	}
	userMsg := goal
	if contextHint != "" {
		userMsg = "Context:\n" + contextHint + "\n\nGoal:\n" + goal
	}
	tid := rc.Tid
	if tid == "" {
		tid = tenant.ID(ctx)
	}
	return r.subagents.Spawn(SpawnOpts{
		ParentSession:   parent,
		SpawnToolCallID: rc.ToolCallID,
		AgentKey:        agentKey,
		UserMessage:     userMsg,
		Goal:            goal,
		MaxRounds:       maxRounds,
		Tid:             tid,
		OnProgress:      rc.Emit,
	})
}

// SubagentStatusJSON implements tool.SubagentBackend.
func (r *Runner) SubagentStatusJSON(ctx context.Context, parentSession, childSession string) (string, error) {
	if r.subagents == nil {
		return "", fmt.Errorf("subagent_status unavailable")
	}
	resp, err := r.subagents.Status(ctx, parentSession, childSession)
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(resp)
	return string(b), nil
}

// SubagentWaitJSON implements tool.SubagentBackend.
func (r *Runner) SubagentWaitJSON(ctx context.Context, parentSession, childSession string, timeoutSec int) (string, error) {
	if r.subagents == nil {
		return "", fmt.Errorf("subagent_wait unavailable")
	}
	resp, err := r.subagents.Wait(ctx, parentSession, childSession, time.Duration(timeoutSec)*time.Second)
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(resp)
	return string(b), nil
}
