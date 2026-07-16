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
	"github.com/OptLTD/swiflow/internal/tool"
	"github.com/OptLTD/swiflow/library/support"
)

// maxRoundsDefault is only a safety fuse against runaway tool use.
const maxRoundsDefault = 32

const defaultToolTimeout = 120 * time.Second

const softBudgetNudge = "You are approaching the tool-call budget for this turn. Prefer finishing with a clear answer now if you already have enough information. Use another tool only if it is essential to complete the user's request."

const hardBudgetNudge = "You have hit the tool-call safety limit for this turn. Do not call tools. Summarize what you completed, what you found, what is blocked or unfinished, and the most useful next step for the user."

const stallNudge = "Progress has stalled (repeated tools or repeated failures). Do not call tools again. Tell the user what you tried, why it is stuck, and the best next step — ask them to continue if needed."

const childWrapUpNudge = "Sub-agent budget is running low. Finish the batch now: if the deliverable file (e.g. xlsx) exists, reply with ONLY its @/ path and row count — no more tools. If blocked, summarize what failed and stop."

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
	Type      string `json:"type"` // delta|thinking|tool_call|tool_result|tool_progress|done|error|ui_request|user|queued
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

	Workspace          string
	MaxHistoryMessages int

	// Publish is optional; when set, every emit is also published for watchers.
	Publish EventPublisher

	// MaxConcurrentRuns caps global in-flight runs; 0 = unlimited.
	MaxConcurrentRuns int
	// ToolTimeoutSec wraps each tool call; 0 = 120s.
	ToolTimeoutSec int
	// ToolTimeouts overrides per-tool timeouts (single source of truth), e.g.
	// {"document_extract": DocumentTimeout}. Falls back to ToolTimeoutSec.
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
	cancels   map[string]context.CancelFunc
	queue     map[string][]queuedMsg
	provMu    sync.Mutex
	provCache map[string]llmclient.Provider
}

// NewRunner constructs a Runner.
func NewRunner(deps RunnerDeps) *Runner {
	return &Runner{
		deps:      deps,
		busy:      map[string]struct{}{},
		cancels:   map[string]context.CancelFunc{},
		queue:     map[string][]queuedMsg{},
		provCache: map[string]llmclient.Provider{},
	}
}

// ErrBusy is returned when a session already has a run in flight.
var ErrBusy = fmt.Errorf("session busy")

// ErrConcurrent is returned when the global concurrent-run gate is full.
var ErrConcurrent = fmt.Errorf("too many concurrent runs")

// IsBusy reports whether a session has a run in flight.
func (r *Runner) IsBusy(sessionID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.busy[sessionID]
	return ok
}

// AtCapacity reports whether the global concurrent-run gate is full.
func (r *Runner) AtCapacity() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	max := r.deps.MaxConcurrentRuns
	return max > 0 && len(r.busy) >= max
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
	return true
}

// InvalidateAll drops cached LLM providers so subsequent runs re-read config.
func (r *Runner) InvalidateAll() {
	r.provMu.Lock()
	r.provCache = map[string]llmclient.Provider{}
	r.provMu.Unlock()
}

// InvalidateProvider drops a single cached provider.
func (r *Runner) InvalidateProvider(name string) {
	r.provMu.Lock()
	delete(r.provCache, name)
	r.provMu.Unlock()
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

	// Claim the session + optional global gate.
	r.mu.Lock()
	if _, busy := r.busy[sessionID]; busy {
		r.mu.Unlock()
		observe.BusyReject(sessionID)
		publisher(Event{Type: "error", Error: "session busy"})
		return runResult{status: "error"}, ErrBusy
	}
	if max := r.deps.MaxConcurrentRuns; max > 0 && len(r.busy) >= max {
		n := len(r.busy)
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
	r.cancels[sessionID] = cancel
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.busy, sessionID)
		delete(r.cancels, sessionID)
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

	llmMsgs := []llmclient.Message{{Role: "system", Content: r.buildSystem(ag)}}
	for _, m := range history {
		llmMsgs = append(llmMsgs, toLLM(m))
	}
	llmMsgs = append(llmMsgs, llmclient.Message{Role: "user", Content: userMessage})

	toolDefs := filterTools(r.deps.Tools.Definitions(), opts)
	firstUser := firstUserMessage(history, userMessage)

	// Deterministic batch routing (once): if the user attached >= threshold files,
	// the main agent must hand the whole batch to a child — heavy tools are gated
	// here and a single delegate_task is required. No runtime cost probing.
	if paths, forced := shouldForceBatchDelegate(userMessage, childRun); forced {
		denyDocumentExtract(&opts)
		toolDefs = filterTools(r.deps.Tools.Definitions(), opts)
		llmMsgs = append(llmMsgs, llmclient.Message{Role: "user", Content: batchDelegateNudge(paths)})
		slog.Info("agent.batch_delegate_forced", "session", sessionID, "paths", len(paths))
	}

	var (
		repeat       int
		lastKey      string
		consecErrors int
		softNudged   bool
		forceWrapUp  bool
		wrapReason   string
		toolsUsed    bool
		childNudged  bool
		llmRetries   int
	)

	for round := 0; round < rounds; {
		observe.RoundStart(sessionID, round)
		roundTools := toolDefs
		roundT0 := time.Now()

		switch {
		case forceWrapUp || round == rounds-1:
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
		case !softNudged && round >= rounds*3/4:
			softNudged = true
			llmMsgs = append(llmMsgs, llmclient.Message{Role: "user", Content: softBudgetNudge})
		case childRun && !childNudged && round >= rounds/2:
			childNudged = true
			llmMsgs = append(llmMsgs, llmclient.Message{Role: "user", Content: childWrapUpNudge})
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
				slog.Info("agent.run_end", "session", sessionID, "reason", "llm_error_wrapup", "round", round, "child", childRun, "error", err.Error())
				res.status = "error"
				res.summary = summary
				return res, nil
			}
			publisher(Event{Type: "error", Error: err.Error()})
			res.status = "error"
			return res, err
		}
		llmRetries = 0
		slog.Info("agent.llm_done", "session", sessionID, "round", round, "model", model, "ms", llmMS, "had_tools", len(resp.ToolCalls) > 0)

		if len(resp.ToolCalls) > 0 && len(roundTools) > 0 {
			toolsUsed = true
			res.rounds = round + 1
			observe.RoundEnd(sessionID, round, true)
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

			outcomes := r.executeToolCalls(runCtx, persistCtx, sessionID, agentKey, resp.ToolCalls, publisher)
			res.toolCalls += len(outcomes)
			for _, o := range outcomes {
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

		// Model produced a no-tool reply: this is the final answer / wrap-up.
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
		if content != "" && content != resp.Content {
			publisher(Event{Type: "delta", Content: content})
		}

		title := ""
		if sess.Title == "" {
			title = titleFromMessage(firstUser)
			if err := st.UpdateSessionTitle(runCtx, sessionID, title); err != nil {
				slog.Error("set session title", "error", err)
			}
		}
		publisher(Event{Type: "done", Title: title})
		slog.Info("agent.run_end", "session", sessionID, "reason", "done", "round", round, "max_rounds", rounds, "child", childRun)
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
	slog.Info("agent.run_end", "session", sessionID, "reason", "budget", "max_rounds", rounds, "child", childRun)
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

// resolveTxtModel looks up llm_provider by name (agent.txt_model) and returns
// the chat client plus the model id defined on that provider row.
func (r *Runner) resolveTxtModel(ctx context.Context, name string) (llmclient.Provider, string, error) {
	apiBase, apiKey, model, err := r.deps.Store.ProviderCreds(ctx, name)
	if err != nil {
		return nil, "", err
	}
	r.provMu.Lock()
	if p, ok := r.provCache[name]; ok {
		r.provMu.Unlock()
		return p, model, nil
	}
	r.provMu.Unlock()
	p := llmclient.NewOpenAIProvider(name, apiBase, apiKey, "")
	p.SetDisableThinking(r.deps.DisableThinking)
	r.provMu.Lock()
	if existing, ok := r.provCache[name]; ok {
		r.provMu.Unlock()
		return existing, model, nil
	}
	r.provCache[name] = p
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

func (r *Runner) buildSystem(ag *store.Agent) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are Swiflow agent %s.", ag.Key)
	if ag.SysPrompt != "" {
		b.WriteString("\n\n")
		b.WriteString(ag.SysPrompt)
	}
	if r.deps.Workspace != "" {
		b.WriteString("\n\n## Workspace\nWorkspace root: ")
		b.WriteString(r.deps.Workspace)
		b.WriteString(". File tools are restricted to it.\n")
		b.WriteString("User messages may cite workspace files as @/relative/path (e.g. @/notes.txt, @/docs/a.md). ")
		b.WriteString("@/ means the workspace root. Attached uploads appear in a block between [UPLOAD FILES START] and [UPLOAD FILES END] (one @/ path per line). ")
		b.WriteString("File tools accept both workspace-relative paths and @/… (equivalent). Prefer passing the path as given; do not invent a literal \"@\" directory. ")
		b.WriteString("When the user attaches or mentions @/…, resolve it to that relative path and use fs_* / document_extract / other file tools on it — do not treat @/ as a URL or package alias. ")
		b.WriteString("When you refer to workspace files in replies, prefer the same @/ form.")
	}
	disabled := map[string]bool{}
	if r.deps.Skills != nil {
		if list, err := r.deps.Store.DisabledSkills(context.Background()); err == nil {
			for _, s := range list {
				disabled[s] = true
			}
		}
		summary := r.deps.Skills.Summary(context.Background(), disabled)
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
	b.WriteString("\n\n## Delegation\n")
	b.WriteString("Use delegate_task ONCE for a batch of remaining work (own session + round budget). ")
	b.WriteString("List every remaining @/ path inside goal; the child chooses tools itself — do not pass path/tools args, and do not one-file-per-delegate. ")
	b.WriteString("You receive one structured result (status/summary/artifacts/metrics) — large artifacts stay in workspace files cited in the child goal.\n")
	b.WriteString("When many files or a table/Excel deliverable is obvious, delegate early. ")
	b.WriteString("When ≥3 @/ files are attached for a table, document_extract is disabled on the main agent by the runtime: do not ask which columns — pick sensible defaults and hand the whole batch to one delegate_task. ")
	b.WriteString("Never pretend you will continue later without calling tools.")
	b.WriteString("\n\n## Clarify\n")
	b.WriteString("When you need a user choice, confirmation, or missing info before continuing, call clarify. ")
	b.WriteString("Do not invent answers; wait for the tool result. Prefer short options when choices are discrete.")
	b.WriteString("\n\n## Learning & memory\n")
	b.WriteString("After completing a significant task (success or failure), call experience_write to record: a one-sentence summary, the outcome, and 1-3 topic tags.\n")
	b.WriteString("Before starting a complex or unfamiliar task, call experience_list to check for relevant prior experience that could shortcut the work.\n")
	b.WriteString("When the same pattern appears in multiple experiences, use skill_manage to promote it into a reusable skill.")
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
		for _, tc := range m.ToolCalls {
			if tc.ID != "" {
				needed[tc.ID] = tc
			}
		}
		found := make(map[string]store.Message, len(needed))
		j := i + 1
		for j < len(msgs) && msgs[j].Role == "tool" {
			id := msgs[j].ToolCallId
			if _, ok := needed[id]; ok {
				found[id] = msgs[j]
			}
			j++
		}

		out = append(out, m)
		for _, tc := range m.ToolCalls {
			if tc.ID == "" {
				continue
			}
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
		DenyTools:     map[string]bool{"delegate_task": true, "clarify": true},
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
	r.provCache[name] = p
}
