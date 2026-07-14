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

	"github.com/OptLTD/swiflow/internal/util"
	"github.com/OptLTD/swiflow/internal/llm"
	"github.com/OptLTD/swiflow/internal/skill"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/tool"
)

const maxRounds = 12

type object = map[string]any

// Event is one event streamed to the client during a run.
type Event struct {
	Type      string `json:"type"` // delta|thinking|tool_call|tool_result|done|error
	Content   string `json:"content,omitempty"`
	Thinking  string `json:"thinking,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments object `json:"arguments,omitempty"`
	Result    string `json:"result,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
	Error     string `json:"error,omitempty"`
	Title     string `json:"title,omitempty"`
}

// RunnerDeps configures a Runner.
type RunnerDeps struct {
	Store  store.Store
	Tools  *tool.Registry
	Skills *skill.Catalog

	Workspace            string
	MaxHistoryMessages   int
}

// Runner executes agent runs and enforces single-run-per-session.
type Runner struct {
	deps RunnerDeps

	mu        sync.Mutex
	busy      map[string]struct{}
	cancels   map[string]context.CancelFunc
	provMu    sync.Mutex
	provCache map[string]llm.Provider
}

// NewRunner constructs a Runner.
func NewRunner(deps RunnerDeps) *Runner {
	return &Runner{
		deps:      deps,
		busy:      map[string]struct{}{},
		cancels:   map[string]context.CancelFunc{},
		provCache: map[string]llm.Provider{},
	}
}

// ErrBusy is returned when a session already has a run in flight.
var ErrBusy = fmt.Errorf("session busy")

// IsBusy reports whether a session has a run in flight.
func (r *Runner) IsBusy(sessionKey string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.busy[sessionKey]
	return ok
}

// Abort cancels an in-flight run for a session. Returns true if a run was
// aborted.
func (r *Runner) Abort(sessionKey string) bool {
	r.mu.Lock()
	cancel, ok := r.cancels[sessionKey]
	r.mu.Unlock()
	if !ok {
		return false
	}
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

// Run executes one agent run, streaming events via onEvent.
func (r *Runner) Run(ctx context.Context, sessionKey, agentKey, userMessage string, onEvent func(Event)) error {
	// Claim the session (single-run guard).
	r.mu.Lock()
	if _, busy := r.busy[sessionKey]; busy {
		r.mu.Unlock()
		emit(onEvent, Event{Type: "error", Error: "session busy"})
		return ErrBusy
	}
	runCtx, cancel := context.WithCancel(ctx)
	r.busy[sessionKey] = struct{}{}
	r.cancels[sessionKey] = cancel
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.busy, sessionKey)
		delete(r.cancels, sessionKey)
		r.mu.Unlock()
		cancel()
	}()

	st := r.deps.Store

	// Get-or-create session.
	sess, err := st.GetSessionByKey(runCtx, sessionKey)
	if err != nil {
		if agentKey == "" {
			agentKey = "default"
		}
		sess = &store.Session{ID: util.NewID(), Key: sessionKey, AgentKey: agentKey}
		if cerr := st.CreateSession(runCtx, sess); cerr != nil {
			// Maybe created concurrently; try once more.
			sess, err = st.GetSessionByKey(runCtx, sessionKey)
			if err != nil {
				emit(onEvent, Event{Type: "error", Error: "session unavailable"})
				return err
			}
		}
	} else {
		agentKey = sess.AgentKey
	}

	// Resolve agent.
	ag, err := st.GetAgentByKey(runCtx, agentKey)
	if err != nil {
		emit(onEvent, Event{Type: "error", Error: "agent not found: " + agentKey})
		return err
	}

	// Resolve provider (cached, invalidate on config change).
	prov, err := r.provider(runCtx, ag.Provider)
	if err != nil {
		emit(onEvent, Event{Type: "error", Error: err.Error()})
		return err
	}

	// Load history.
	history, err := st.ListMessages(runCtx, sessionKey)
	if err != nil {
		emit(onEvent, Event{Type: "error", Error: "load history failed"})
		return err
	}
	history = truncateHistory(history, r.deps.MaxHistoryMessages)

	// Persist the user message immediately (survives first-round failure).
	userMsg := store.Message{ID: util.NewID(), Role: "user", Content: userMessage}
	if _, err := st.AppendMessage(runCtx, sessionKey, userMsg); err != nil {
		slog.Error("persist user message", "error", err)
	}

	// Build LLM messages: system + history + user.
	llmMsgs := []llm.Message{{Role: "system", Content: r.buildSystem(ag)}}
	for _, m := range history {
		llmMsgs = append(llmMsgs, toLLM(m))
	}
	llmMsgs = append(llmMsgs, llm.Message{Role: "user", Content: userMessage})

	toolDefs := r.deps.Tools.Definitions()

	firstUser := firstUserMessage(history, userMessage)

	var lastKey string
	repeat := 0
	for round := 0; round < maxRounds; round++ {
		req := llm.ChatRequest{Model: ag.Model, Messages: llmMsgs, Tools: toolDefs}
		resp, err := r.streamRound(runCtx, prov, req, onEvent)
		if err != nil {
			emit(onEvent, Event{Type: "error", Error: err.Error()})
			return err
		}

		if len(resp.ToolCalls) > 0 {
			key := toolCallKey(resp.ToolCalls)
			if key == lastKey {
				repeat++
				if repeat >= 3 {
					e := fmt.Errorf("tool loop detected: repeated %s", resp.ToolCalls[0].Name)
					emit(onEvent, Event{Type: "error", Error: e.Error()})
					return e
				}
			} else {
				repeat = 0
				lastKey = key
			}

			assistantMsg := store.Message{
				ID:            util.NewID(),
				Role:          "assistant",
				Content:       resp.Content,
				Thinking:      resp.Thinking,
				ToolCallsJSON: marshalToolCalls(resp.ToolCalls),
			}
			if _, err := st.AppendMessage(runCtx, sessionKey, assistantMsg); err != nil {
				slog.Error("persist assistant (tool) message", "error", err)
			}
			llmMsgs = append(llmMsgs, toLLM(assistantMsg))

			for _, tc := range resp.ToolCalls {
				emit(onEvent, Event{Type: "tool_call", ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments})
				var result string
				var execErr error
				if !st.ToolEnabled(runCtx, tc.Name) {
					execErr = fmt.Errorf("tool %q is disabled", tc.Name)
				} else {
					result, execErr = r.deps.Tools.Execute(tool.WithRunContext(runCtx, tool.RunContext{
						SessionKey: sessionKey,
						AgentKey:   agentKey,
					}), tc.Name, tc.Arguments)
				}
				isErr := execErr != nil
				if isErr {
					result = "error: " + execErr.Error()
				}
				if result == "" {
					result = "(no output)"
				}
				truncated := truncateToolResult(result)
				emit(onEvent, Event{Type: "tool_result", ID: tc.ID, Name: tc.Name, Result: truncated, IsError: isErr})

				toolMsg := store.Message{
					ID:         util.NewID(),
					Role:       "tool",
					Content:    truncated,
					ToolCallID: tc.ID,
					ToolName:   tc.Name,
				}
				if _, err := st.AppendMessage(runCtx, sessionKey, toolMsg); err != nil {
					slog.Error("persist tool message", "error", err)
				}
				llmMsgs = append(llmMsgs, llm.Message{
					Role:       "tool",
					Content:    truncated,
					ToolCallID: tc.ID,
					ToolName:   tc.Name,
				})
			}
			continue
		}

		// Final answer.
		assistantMsg := store.Message{
			ID:       util.NewID(),
			Role:     "assistant",
			Content:  resp.Content,
			Thinking: resp.Thinking,
		}
		if _, err := st.AppendMessage(runCtx, sessionKey, assistantMsg); err != nil {
			slog.Error("persist assistant message", "error", err)
		}

		title := ""
		if sess.Title == "" {
			title = titleFromMessage(firstUser)
			if err := st.UpdateSessionTitle(runCtx, sessionKey, title); err != nil {
				slog.Error("set session title", "error", err)
			}
		}
		emit(onEvent, Event{Type: "done", Title: title})
		return nil
	}

	e := fmt.Errorf("run exceeded max rounds (%d)", maxRounds)
	emit(onEvent, Event{Type: "error", Error: e.Error()})
	return e
}

func (r *Runner) provider(ctx context.Context, name string) (llm.Provider, error) {
	r.provMu.Lock()
	if p, ok := r.provCache[name]; ok {
		r.provMu.Unlock()
		return p, nil
	}
	r.provMu.Unlock()
	apiBase, apiKey, err := r.deps.Store.ProviderCreds(ctx, name)
	if err != nil {
		return nil, err
	}
	p := llm.NewOpenAIProvider(name, apiBase, apiKey, "")
	r.provMu.Lock()
	// Another goroutine may have cached; keep first.
	if existing, ok := r.provCache[name]; ok {
		r.provMu.Unlock()
		return existing, nil
	}
	r.provCache[name] = p
	r.provMu.Unlock()
	return p, nil
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

// buildSystem builds the system prompt. Spec §7.1.
func (r *Runner) buildSystem(ag *store.Agent) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are Swiflow agent %s.", ag.Key)
	if ag.SystemExtra != "" {
		b.WriteString("\n\n")
		b.WriteString(ag.SystemExtra)
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
	b.WriteString("\n\n## Scheduling\n")
	b.WriteString("Use schedule_run to re-invoke the agent in the current chat after a delay (delay_seconds + message as a new user turn). ")
	b.WriteString("Use schedule_create for recurring cron jobs (@hourly, 0 9 * * *, @every 5m).")
	b.WriteString("\n\n## Skill authoring\n")
	b.WriteString("Use skill_manage to save reusable workflows: action create with full SKILL.md content for new skills; ")
	b.WriteString("action patch with old_string/new_string for small edits (preferred). User skills override built-ins by slug.")
	return b.String()
}

// --- helpers ---

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
		ToolCallID: m.ToolCallID,
		ToolName:   m.ToolName,
	}
	if m.ToolCallsJSON != "" {
		var tcs []llm.ToolCall
		if json.Unmarshal([]byte(m.ToolCallsJSON), &tcs) == nil {
			out.ToolCalls = tcs
		}
	}
	return out
}

func marshalToolCalls(tcs []llm.ToolCall) string {
	b, _ := json.Marshal(tcs)
	return string(b)
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
	if max <= 0 || len(msgs) <= max {
		return msgs
	}
	return msgs[len(msgs)-max:]
}

func titleFromMessage(msg string) string {
	msg = strings.ReplaceAll(msg, "\n", " ")
	msg = strings.TrimSpace(msg)
	if len(msg) > 60 {
		return msg[:60] + "…"
	}
	return msg
}
