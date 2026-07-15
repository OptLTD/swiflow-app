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

	"github.com/OptLTD/swiflow/internal/llm"
	"github.com/OptLTD/swiflow/internal/observe"
	"github.com/OptLTD/swiflow/internal/skill"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/tool"
	"github.com/OptLTD/swiflow/internal/util"
)

// maxRoundsDefault is only a safety fuse against runaway tool use.
const maxRoundsDefault = 32

const defaultToolTimeout = 120 * time.Second

const softBudgetNudge = "You are approaching the tool-call budget for this turn. Prefer finishing with a clear answer now if you already have enough information. Use another tool only if it is essential to complete the user's request."

const hardBudgetNudge = "You have hit the tool-call safety limit for this turn. Do not call tools. Summarize what you completed, what you found, what is blocked or unfinished, and the most useful next step for the user."

const stallNudge = "Progress has stalled (repeated tools or repeated failures). Do not call tools again. Tell the user what you tried, why it is stuck, and the best next step — ask them to continue if needed."

const continueHint = "I could not fully finish this turn. Tell me to continue and I will pick up from here."

type object = map[string]any

// Event is one event streamed to the client during a run.
type Event struct {
	Type      string `json:"type"` // delta|thinking|tool_call|tool_result|done|error|ui_request|user|queued
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
}

// EventPublisher fans out events (typically sesshub.Hub).
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
}

type queuedMsg struct {
	AgentKey string
	Message  string
}

// RunOpts controls a single Run (used by subagents).
type RunOpts struct {
	MaxRounds int // 0 = default 32
	// AllowTools: if non-nil, only these tool names are offered.
	AllowTools map[string]bool
	// DenyTools: always excluded (e.g. delegate_task for children).
	DenyTools map[string]bool
}

// Runner executes agent runs and enforces single-run-per-session.
type Runner struct {
	deps RunnerDeps

	mu        sync.Mutex
	busy      map[string]struct{}
	cancels   map[string]context.CancelFunc
	queue     map[string][]queuedMsg
	provMu    sync.Mutex
	provCache map[string]llm.Provider
}

// NewRunner constructs a Runner.
func NewRunner(deps RunnerDeps) *Runner {
	return &Runner{
		deps:      deps,
		busy:      map[string]struct{}{},
		cancels:   map[string]context.CancelFunc{},
		queue:     map[string][]queuedMsg{},
		provCache: map[string]llm.Provider{},
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
	r.provCache = map[string]llm.Provider{}
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
	rounds := opts.MaxRounds
	if rounds <= 0 {
		rounds = maxRoundsDefault
	}

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
		return ErrBusy
	}
	if max := r.deps.MaxConcurrentRuns; max > 0 && len(r.busy) >= max {
		n := len(r.busy)
		r.mu.Unlock()
		observe.ConcurrentReject(sessionID, n, max)
		publisher(Event{Type: "error", Error: "too many concurrent runs"})
		return ErrConcurrent
	}
	runCtx, cancel := context.WithCancel(ctx)
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
		sess = &store.Session{ID: sessionID, Agent: agentKey}
		if cerr := st.CreateSession(runCtx, sess); cerr != nil {
			sess, err = st.GetSessionByID(runCtx, sessionID)
			if err != nil {
				publisher(Event{Type: "error", Error: "session unavailable"})
				return err
			}
		}
	} else {
		agentKey = sess.Agent
	}

	ag, err := st.GetAgentByKey(runCtx, agentKey)
	if err != nil {
		publisher(Event{Type: "error", Error: "agent not found: " + agentKey})
		return err
	}

	if ag.TxtModel == "" {
		publisher(Event{Type: "error", Error: "agent has no txt_model"})
		return fmt.Errorf("agent %q has no txt_model", agentKey)
	}
	prov, model, err := r.resolveTxtModel(runCtx, ag.TxtModel)
	if err != nil {
		publisher(Event{Type: "error", Error: err.Error()})
		return err
	}

	history, err := st.ListMessages(runCtx, sessionID)
	if err != nil {
		publisher(Event{Type: "error", Error: "load history failed"})
		return err
	}
	history = truncateHistory(history, r.deps.MaxHistoryMessages)

	userMsg := store.Message{ID: util.NewID(), Role: "user", Content: userMessage}
	if _, err := st.AppendMessage(runCtx, sessionID, userMsg); err != nil {
		slog.Error("persist user message", "error", err)
	}

	// Persist tool pairing even if the run context is cancelled mid-loop.
	persistCtx := context.WithoutCancel(runCtx)

	llmMsgs := []llm.Message{{Role: "system", Content: r.buildSystem(ag)}}
	for _, m := range history {
		llmMsgs = append(llmMsgs, toLLM(m))
	}
	llmMsgs = append(llmMsgs, llm.Message{Role: "user", Content: userMessage})

	toolDefs := filterTools(r.deps.Tools.Definitions(), opts)

	firstUser := firstUserMessage(history, userMessage)
	toolTimeout := time.Duration(r.deps.ToolTimeoutSec) * time.Second
	if toolTimeout <= 0 {
		toolTimeout = defaultToolTimeout
	}

	var (
		repeat       int
		lastKey      string
		consecErrors int
		softNudged   bool
		forceWrapUp  bool
		wrapReason   string
	)

	for round := 0; round < rounds; round++ {
		observe.RoundStart(sessionID, round)
		roundTools := toolDefs

		switch {
		case forceWrapUp || round == rounds-1:
			roundTools = nil
			nudge := hardBudgetNudge
			if forceWrapUp && wrapReason == "stall" {
				nudge = stallNudge
			}
			forceWrapUp = false
			wrapReason = ""
			llmMsgs = append(llmMsgs, llm.Message{Role: "user", Content: nudge})
		case !softNudged && round >= rounds*3/4:
			softNudged = true
			llmMsgs = append(llmMsgs, llm.Message{Role: "user", Content: softBudgetNudge})
		}

		req := llm.ChatRequest{Model: model, Messages: llmMsgs, Tools: roundTools}
		resp, err := r.streamRound(runCtx, prov, req, publisher)
		if err != nil {
			publisher(Event{Type: "error", Error: err.Error()})
			return err
		}

		if len(resp.ToolCalls) > 0 && len(roundTools) > 0 {
			observe.RoundEnd(sessionID, round, true)
			key := toolCallKey(resp.ToolCalls)
			if key == lastKey {
				repeat++
				if repeat >= 3 {
					forceWrapUp = true
					wrapReason = "stall"
					continue
				}
			} else {
				repeat = 0
				lastKey = key
			}

			assistantMsg := store.Message{
				ID:        util.NewID(),
				Role:      "assistant",
				Content:   resp.Content,
				Thinking:  resp.Thinking,
				ToolCalls: toStoreToolCalls(resp.ToolCalls),
			}
			if _, err := st.AppendMessage(persistCtx, sessionID, assistantMsg); err != nil {
				slog.Error("persist assistant (tool) message", "error", err)
			}
			llmMsgs = append(llmMsgs, toLLM(assistantMsg))

			for _, tc := range resp.ToolCalls {
				publisher(Event{Type: "tool_call", ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments})
				var result string
				var execErr error
				t0 := time.Now()
				observe.ToolStart(sessionID, tc.Name)
				if err := runCtx.Err(); err != nil {
					execErr = fmt.Errorf("cancelled: %w", err)
				} else if !st.ToolEnabled(runCtx, tc.Name) {
					execErr = fmt.Errorf("tool %q is disabled", tc.Name)
				} else {
					timeout := toolTimeout
					if tc.Name == "clarify" {
						timeout = 15 * time.Minute
					}
					tctx, tcancel := context.WithTimeout(runCtx, timeout)
					result, execErr = r.deps.Tools.Execute(tool.WithRunContext(tctx, tool.RunContext{
						SessionID: sessionID,
						Agent:     agentKey,
					}), tc.Name, tc.Arguments)
					tcancel()
				}
				observe.ToolEnd(sessionID, tc.Name, time.Since(t0), execErr)
				isErr := execErr != nil
				if isErr {
					result = "error: " + execErr.Error()
					consecErrors++
				} else {
					consecErrors = 0
				}
				if result == "" {
					result = "(no output)"
				}
				truncated := truncateToolResult(result)
				publisher(Event{Type: "tool_result", ID: tc.ID, Name: tc.Name, Result: truncated, IsError: isErr})

				toolMsg := store.Message{
					ID:         util.NewID(),
					Role:       "tool",
					Content:    truncated,
					ToolCallId: tc.ID,
					ToolName:   tc.Name,
				}
				if _, err := st.AppendMessage(persistCtx, sessionID, toolMsg); err != nil {
					slog.Error("persist tool message", "error", err)
				}
				llmMsgs = append(llmMsgs, llm.Message{
					Role:       "tool",
					Content:    truncated,
					ToolCallID: tc.ID,
					ToolName:   tc.Name,
				})
			}
			if consecErrors >= 3 {
				forceWrapUp = true
				wrapReason = "stall"
			}
			continue
		}

		observe.RoundEnd(sessionID, round, false)
		content := resp.Content
		if strings.TrimSpace(content) == "" && (forceWrapUp || round == rounds-1) {
			content = continueHint
		}
		assistantMsg := store.Message{
			ID:       util.NewID(),
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
		return nil
	}

	if _, err := st.AppendMessage(runCtx, sessionID, store.Message{
		ID: util.NewID(), Role: "assistant", Content: continueHint,
	}); err != nil {
		slog.Error("persist wrap-up message", "error", err)
	}
	publisher(Event{Type: "delta", Content: continueHint})
	publisher(Event{Type: "done"})
	return nil
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

func filterTools(defs []llm.ToolDef, opts RunOpts) []llm.ToolDef {
	if opts.AllowTools == nil && len(opts.DenyTools) == 0 {
		return defs
	}
	out := make([]llm.ToolDef, 0, len(defs))
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
func (r *Runner) resolveTxtModel(ctx context.Context, name string) (llm.Provider, string, error) {
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
	p := llm.NewOpenAIProvider(name, apiBase, apiKey, "")
	r.provMu.Lock()
	if existing, ok := r.provCache[name]; ok {
		r.provMu.Unlock()
		return existing, model, nil
	}
	r.provCache[name] = p
	r.provMu.Unlock()
	return p, model, nil
}

func (r *Runner) streamRound(ctx context.Context, p llm.Provider, req llm.ChatRequest, onEvent func(Event)) (*llm.ChatResponse, error) {
	return p.ChatStream(ctx, req, func(c llm.StreamChunk) {
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
		b.WriteString(". File tools are restricted to it.")
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
	b.WriteString("\n\n## Delegation\n")
	b.WriteString("Use delegate_task to spawn an isolated sub-agent with its own session and round budget. ")
	b.WriteString("You receive only a final summary — put large artifacts in workspace files and cite paths in the child goal if the parent will need them.")
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

func toLLM(m store.Message) llm.Message {
	out := llm.Message{
		Role:       m.Role,
		Content:    m.Content,
		Thinking:   m.Thinking,
		ToolCallID: m.ToolCallId,
		ToolName:   m.ToolName,
	}
	if len(m.ToolCalls) > 0 {
		out.ToolCalls = make([]llm.ToolCall, len(m.ToolCalls))
		for i, tc := range m.ToolCalls {
			out.ToolCalls[i] = llm.ToolCall{
				ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments,
			}
		}
	}
	return out
}

func toStoreToolCalls(tcs []llm.ToolCall) []store.ToolCall {
	if len(tcs) == 0 {
		return nil
	}
	out := make([]store.ToolCall, len(tcs))
	for i, tc := range tcs {
		out[i] = store.ToolCall{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments}
	}
	return out
}

func toolCallKey(tcs []llm.ToolCall) string {
	if len(tcs) == 0 {
		return ""
	}
	sorted := append([]llm.ToolCall(nil), tcs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	b, _ := json.Marshal(sorted)
	return string(b)
}

func truncateToolResult(s string) string {
	const max = 4000
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...[truncated]"
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

// RunChild implements tool.ChildRunner for delegate_task.
func (r *Runner) RunChild(ctx context.Context, opts tool.ChildRunOpts, onDelta func(string)) error {
	allow := map[string]bool(nil)
	if opts.AllowTools != nil {
		allow = make(map[string]bool, len(opts.AllowTools))
		for _, n := range opts.AllowTools {
			allow[n] = true
		}
	}
	return r.RunOpts(ctx, opts.SessionID, opts.AgentKey, opts.UserMessage, func(ev Event) {
		if ev.Type == "delta" && onDelta != nil && ev.Content != "" {
			onDelta(ev.Content)
		}
	}, RunOpts{
		MaxRounds:  opts.MaxRounds,
		AllowTools: allow,
		DenyTools:  map[string]bool{"delegate_task": true, "clarify": true},
	})
}

// SetProvider injects a cached LLM provider (tests / overrides).
func (r *Runner) SetProvider(name string, p llm.Provider) {
	r.provMu.Lock()
	defer r.provMu.Unlock()
	r.provCache[name] = p
}
