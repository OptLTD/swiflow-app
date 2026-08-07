package harness

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/OptLTD/swiflow/internal/agent"
	"github.com/OptLTD/swiflow/internal/store"
)

// EventPublisher is satisfied by server.SessionHub (and Tracker itself).
type EventPublisher interface {
	Publish(sessionID string, ev agent.Event)
}

// TodoLoader loads checklist JSON for a session (store.Store).
type TodoLoader interface {
	LoadTodos(ctx context.Context, sessionID string) (string, error)
}

// SessionTidLoader resolves a session's tenant id without tenant filtering.
type SessionTidLoader interface {
	SessionTid(ctx context.Context, sessionID string) (string, error)
}

// Tracker observes agent events, maintains RunSnapshots, and emits harness_warn.
// Phase 1: observe only — does not inject into the LLM loop.
type Tracker struct {
	inner EventPublisher
	todos TodoLoader

	mu    sync.Mutex
	runs  map[string]*runState
	order []string // recent session ids for List

	// warnDedup: sessionID|code → last warn time
	warnDedup map[string]time.Time

	stopCh chan struct{}

	watchers map[chan []byte]struct{}
}

type runState struct {
	snap RunSnapshot

	lastProgress                 time.Time
	consecErrors                 int
	repeatToolStreak             int
	lastToolKey                  string
	onlyExploreTools             bool
	sawAnyTool                   bool
	roundsWithTools              int
	inToolRound                  bool
	todosFingerprint             string
	todosFingerprintAtRoundStart string
	hadProgressSinceHalf         bool
	halfBudgetNoted              bool
	seenWarnCodes                map[string]bool
	pendingDelegateGoal          string
	pendingDelegateMax           int
}

// NewTracker wraps inner (typically SessionHub). todos may be nil (skips checklist).
func NewTracker(inner EventPublisher, todos TodoLoader) *Tracker {
	t := &Tracker{
		inner:     inner,
		todos:     todos,
		runs:      map[string]*runState{},
		warnDedup: map[string]time.Time{},
		stopCh:    make(chan struct{}),
		watchers:  map[chan []byte]struct{}{},
	}
	go t.noProgressLoop()
	return t
}

// Close stops background timers.
func (t *Tracker) Close() {
	select {
	case <-t.stopCh:
	default:
		close(t.stopCh)
	}
}

// Publish implements agent.EventPublisher.
func (t *Tracker) Publish(sessionID string, ev agent.Event) {
	if t == nil {
		return
	}
	if ev.Type != "harness_warn" {
		t.observe(sessionID, ev)
	}
	if t.inner != nil {
		t.inner.Publish(sessionID, ev)
	}
}

func (t *Tracker) noProgressLoop() {
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-t.stopCh:
			return
		case now := <-tick.C:
			t.mu.Lock()
			ids := make([]string, 0, len(t.runs))
			for id, st := range t.runs {
				if st.snap.Status == StatusRunning {
					ids = append(ids, id)
				}
			}
			t.mu.Unlock()
			for _, id := range ids {
				t.mu.Lock()
				st := t.runs[id]
				if st == nil {
					t.mu.Unlock()
					continue
				}
				sigs := evalDrift(st, now)
				t.emitNewWarnsLocked(id, st, sigs)
				t.mu.Unlock()
			}
		}
	}
}

func (t *Tracker) observe(sessionID string, ev agent.Event) {
	now := time.Now()
	t.mu.Lock()
	defer t.mu.Unlock()

	st := t.ensureLocked(sessionID, now)
	st.snap.UpdatedAt = now
	if st.snap.Tid == "" {
		st.snap.Tid = t.lookupTidLocked(sessionID)
	}

	switch ev.Type {
	case "user":
		st.snap.Status = StatusRunning
		if st.snap.StartedAt.IsZero() {
			st.snap.StartedAt = now
		}
		if g := strings.TrimSpace(ev.Content); g != "" && st.snap.Goal == "" {
			st.snap.Goal = truncate(g, 500)
		}
		st.lastProgress = now

	case "queued":
		st.snap.Status = StatusQueued

	case "tool_call":
		st.snap.Status = StatusRunning
		st.snap.CurrentTool = ev.Name
		st.snap.LastAction = ev.Name
		st.snap.Metrics.ToolCalls++
		if !st.sawAnyTool {
			st.onlyExploreTools = isExploreOnlyTool(ev.Name)
			st.sawAnyTool = true
		} else if !isExploreOnlyTool(ev.Name) {
			st.onlyExploreTools = false
		}
		st.lastProgress = now
		if !st.inToolRound {
			st.inToolRound = true
			st.snap.Round++
			st.roundsWithTools++
			st.todosFingerprintAtRoundStart = st.todosFingerprint
		}
		key := ev.Name
		if key == st.lastToolKey {
			st.repeatToolStreak++
		} else {
			st.lastToolKey = key
			st.repeatToolStreak = 1
		}
		if isProgressTool(ev.Name) {
			st.hadProgressSinceHalf = true
		}
		if ev.Name == "subagent_spawn" {
			t.onSubagentSpawnCallLocked(sessionID, st, ev, now)
		}
		if st.snap.ParentID != "" && st.snap.MaxRounds > 0 &&
			st.snap.Round >= st.snap.MaxRounds/2 && !st.halfBudgetNoted {
			st.halfBudgetNoted = true
		}

	case "tool_result":
		st.snap.CurrentTool = ""
		st.snap.LastAction = ev.Name + " done"
		st.lastProgress = now
		if ev.IsError {
			st.consecErrors++
			st.snap.Metrics.Failures++
		} else {
			st.consecErrors = 0
		}
		if ev.Name == "todo_write" || ev.Name == "todo_read" {
			t.refreshTodosLocked(sessionID, st)
		}
		if ev.Name == "subagent_spawn" && ev.Result != "" {
			t.onSubagentSpawnResultLocked(sessionID, st, ev.Result, now)
		}
		// End of tool round when we stop seeing tool_call — heuristic: leave inToolRound
		// until delta/done; clear on next non-tool event below.

	case "tool_progress", "subagent_progress":
		st.snap.LastAction = truncate(ev.Content, 120)
		st.lastProgress = now
		parentID := sessionID
		if ev.Type == "subagent_progress" && ev.Name == "subagent_spawn" {
			st.snap.LastAction = "subagent: " + truncate(ev.Content, 80)
		}
		if ev.Child != "" {
			child := t.ensureLocked(ev.Child, now)
			if child.snap.ParentID == "" {
				child.snap.ParentID = parentID
			}
			child.snap.Status = StatusRunning
			child.snap.LastAction = truncate(ev.Content, 120)
			child.lastProgress = now
			if child.snap.Goal == "" && st.pendingDelegateGoal != "" {
				child.snap.Goal = truncate(st.pendingDelegateGoal, 500)
			}
			if child.snap.MaxRounds == 0 && st.pendingDelegateMax > 0 {
				child.snap.MaxRounds = st.pendingDelegateMax
			}
			if child.snap.StartedAt.IsZero() {
				child.snap.StartedAt = now
			}
			t.linkChildLocked(st, ev.Child)
		}

	case "subagent_done":
		st.lastProgress = now
		if ev.Result != "" {
			t.onSubagentDoneLocked(sessionID, st, ev.Result, now)
		} else if ev.Child != "" {
			child := t.ensureLocked(ev.Child, now)
			child.snap.ParentID = sessionID
			child.snap.Status = StatusDone
			child.snap.UpdatedAt = now
			t.linkChildLocked(st, ev.Child)
		}

	case "delta", "thinking":
		st.inToolRound = false
		st.lastProgress = now
		if ev.Content != "" {
			st.snap.LastAction = truncate(ev.Content, 80)
		}

	case "done":
		st.inToolRound = false
		st.snap.CurrentTool = ""
		if st.snap.Status != StatusStall && st.snap.Status != StatusError {
			st.snap.Status = StatusDone
		}
		st.snap.Metrics.WallMS = now.Sub(st.snap.StartedAt).Milliseconds()
		st.snap.Metrics.Rounds = st.snap.Round
		t.refreshTodosLocked(sessionID, st)

	case "error":
		st.inToolRound = false
		st.snap.Status = StatusError
		st.snap.LastAction = ev.Error
		st.snap.Metrics.WallMS = now.Sub(st.snap.StartedAt).Milliseconds()
	}

	if !st.snap.StartedAt.IsZero() && st.snap.Status == StatusRunning {
		st.snap.Metrics.WallMS = now.Sub(st.snap.StartedAt).Milliseconds()
	}
	st.snap.Metrics.Rounds = st.snap.Round

	sigs := evalDrift(st, now)
	t.emitNewWarnsLocked(sessionID, st, sigs)
	t.notifyWatchersLocked(sessionID, st)
}

func (t *Tracker) notifyWatchersLocked(sessionID string, st *runState) {
	if len(t.watchers) == 0 {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"type":   "run_snapshot",
		"run":    cloneSnap(st.snap),
		"session": sessionID,
	})
	if err != nil {
		return
	}
	for ch := range t.watchers {
		select {
		case ch <- payload:
		default:
		}
	}
}

// SubscribeRuns registers for run_snapshot SSE payloads. Cancel removes the watcher.
func (t *Tracker) SubscribeRuns() (<-chan []byte, func()) {
	ch := make(chan []byte, 32)
	t.mu.Lock()
	if t.watchers == nil {
		t.watchers = map[chan []byte]struct{}{}
	}
	t.watchers[ch] = struct{}{}
	t.mu.Unlock()
	cancel := func() {
		t.mu.Lock()
		delete(t.watchers, ch)
		t.mu.Unlock()
	}
	return ch, cancel
}

func (t *Tracker) ensureLocked(sessionID string, now time.Time) *runState {
	st := t.runs[sessionID]
	if st != nil {
		return st
	}
	st = &runState{
		snap: RunSnapshot{
			SessionID: sessionID,
			Status:    StatusIdle,
			StartedAt: now,
			UpdatedAt: now,
			Children:  nil,
		},
		onlyExploreTools: true,
		seenWarnCodes:    map[string]bool{},
		lastProgress:     now,
	}
	t.runs[sessionID] = st
	t.order = append(t.order, sessionID)
	if len(t.order) > 200 {
		// Drop oldest idle/done entries from index only; keep map until overwritten.
		t.order = t.order[len(t.order)-150:]
	}
	return st
}

func (t *Tracker) linkChildLocked(parent *runState, childID string) {
	for _, c := range parent.snap.Children {
		if c == childID {
			return
		}
	}
	parent.snap.Children = append(parent.snap.Children, childID)
}

func (t *Tracker) onSubagentSpawnCallLocked(_ string, parent *runState, ev agent.Event, _ time.Time) {
	goal, _ := ev.Arguments["goal"].(string)
	maxRounds := 10
	switch v := ev.Arguments["max_rounds"].(type) {
	case float64:
		maxRounds = int(v)
	case int:
		maxRounds = v
	}
	parent.snap.LastAction = "subagent_spawn: " + truncate(goal, 80)
	parent.pendingDelegateMax = maxRounds
	parent.pendingDelegateGoal = goal
}

func (t *Tracker) onSubagentSpawnResultLocked(parentID string, parent *runState, result string, now time.Time) {
	var parsed struct {
		ChildSession string `json:"child_session"`
		Status       string `json:"status"`
		Goal         string `json:"goal"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		return
	}
	if parsed.ChildSession == "" {
		return
	}
	t.linkChildLocked(parent, parsed.ChildSession)
	child := t.ensureLocked(parsed.ChildSession, now)
	child.snap.ParentID = parentID
	if child.snap.Tid == "" {
		child.snap.Tid = parent.snap.Tid
	}
	if child.snap.Goal == "" {
		if parsed.Goal != "" {
			child.snap.Goal = truncate(parsed.Goal, 500)
		} else if parent.pendingDelegateGoal != "" {
			child.snap.Goal = truncate(parent.pendingDelegateGoal, 500)
		}
	}
	if child.snap.MaxRounds == 0 && parent.pendingDelegateMax > 0 {
		child.snap.MaxRounds = parent.pendingDelegateMax
	}
	child.snap.Status = StatusRunning
	if child.snap.StartedAt.IsZero() {
		child.snap.StartedAt = now
	}
	child.snap.UpdatedAt = now
	_ = parsed.Status
}

func (t *Tracker) onSubagentDoneLocked(parentID string, parent *runState, result string, now time.Time) {
	var parsed struct {
		ChildSession string `json:"child_session"`
		Status       string `json:"status"`
		Summary      string `json:"summary"`
		Metrics      struct {
			Rounds    int `json:"rounds"`
			ToolCalls int `json:"tool_calls"`
			Failures  int `json:"failures"`
		} `json:"metrics"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		return
	}
	if parsed.ChildSession == "" {
		return
	}
	t.linkChildLocked(parent, parsed.ChildSession)
	child := t.ensureLocked(parsed.ChildSession, now)
	child.snap.ParentID = parentID
	if child.snap.Goal == "" && parent.pendingDelegateGoal != "" {
		child.snap.Goal = truncate(parent.pendingDelegateGoal, 500)
	}
	if child.snap.MaxRounds == 0 && parent.pendingDelegateMax > 0 {
		child.snap.MaxRounds = parent.pendingDelegateMax
	}
	switch parsed.Status {
	case "budget":
		child.snap.Status = StatusBudget
	case "stall":
		child.snap.Status = StatusStall
	case "error", "blocked":
		child.snap.Status = StatusError
	default:
		child.snap.Status = StatusDone
	}
	child.snap.Metrics.Rounds = parsed.Metrics.Rounds
	child.snap.Metrics.ToolCalls = parsed.Metrics.ToolCalls
	child.snap.Metrics.Failures = parsed.Metrics.Failures
	child.snap.UpdatedAt = now
	if !child.snap.StartedAt.IsZero() {
		child.snap.Metrics.WallMS = now.Sub(child.snap.StartedAt).Milliseconds()
	}
}

func (t *Tracker) refreshTodosLocked(sessionID string, st *runState) {
	if t.todos == nil {
		return
	}
	raw, err := t.todos.LoadTodos(context.Background(), sessionID)
	if err != nil || raw == "" || raw == "[]" {
		return
	}
	var items []TodoItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return
	}
	st.snap.Todos = items
	fp := todosFingerprint(items)
	if fp != st.todosFingerprint {
		st.hadProgressSinceHalf = true
	}
	st.todosFingerprint = fp
}

func (t *Tracker) emitNewWarnsLocked(sessionID string, st *runState, sigs []DriftSignal) {
	for _, sig := range sigs {
		dedupKey := sessionID + "|" + sig.Code
		if last, ok := t.warnDedup[dedupKey]; ok && time.Since(last) < 30*time.Second {
			continue
		}
		// Allow re-warn for no_progress periodically; others once until status changes.
		if sig.Code != DriftNoProgress && st.seenWarnCodes[sig.Code] {
			continue
		}
		st.seenWarnCodes[sig.Code] = true
		t.warnDedup[dedupKey] = time.Now()

		// Keep on snapshot for GET /api/runs review.
		st.snap.Drift = appendDrift(st.snap.Drift, sig)

		ev := agent.Event{
			Type:    "harness_warn",
			Name:    sig.Code,
			Content: sig.Message,
		}
		if st.snap.ParentID != "" {
			ev.Child = sessionID
		}
		// Forward warn without re-entering observe.
		if t.inner != nil {
			t.inner.Publish(sessionID, ev)
			if st.snap.ParentID != "" {
				parentEv := ev
				parentEv.Child = sessionID
				t.inner.Publish(st.snap.ParentID, parentEv)
			}
		}
	}
}

func appendDrift(list []DriftSignal, sig DriftSignal) []DriftSignal {
	for i := range list {
		if list[i].Code == sig.Code {
			list[i] = sig
			return list
		}
	}
	return append(list, sig)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// --- query API ---

// Snapshot returns a copy of one run, or false if unknown.
func (t *Tracker) Snapshot(sessionID string) (RunSnapshot, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	st := t.runs[sessionID]
	if st == nil {
		return RunSnapshot{}, false
	}
	return cloneSnap(st.snap), true
}

// List returns root runs (no parent) plus in-flight children summaries via Children.
// includeFinished: when false, only running/queued.
func (t *Tracker) List(includeFinished bool) []RunSnapshot {
	return t.ListForTenant("", includeFinished)
}

// ListForTenant is like List but keeps only runs whose Tid matches.
// Empty tid returns all tenants.
func (t *Tracker) ListForTenant(tid string, includeFinished bool) []RunSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]RunSnapshot, 0)
	for _, id := range t.order {
		st := t.runs[id]
		if st == nil || st.snap.ParentID != "" {
			continue
		}
		if tid != "" && st.snap.Tid != "" && st.snap.Tid != tid {
			continue
		}
		if !includeFinished && st.snap.Status != StatusRunning && st.snap.Status != StatusQueued {
			continue
		}
		out = append(out, cloneSnap(st.snap))
	}
	return out
}

func (t *Tracker) lookupTidLocked(sessionID string) string {
	if t.todos == nil {
		return ""
	}
	tl, ok := t.todos.(SessionTidLoader)
	if !ok {
		return ""
	}
	tid, err := tl.SessionTid(context.Background(), sessionID)
	if err != nil {
		return ""
	}
	return tid
}

// ListChildren returns snapshots whose ParentID matches.
func (t *Tracker) ListChildren(parentID string) []RunSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]RunSnapshot, 0)
	for _, st := range t.runs {
		if st.snap.ParentID == parentID {
			out = append(out, cloneSnap(st.snap))
		}
	}
	return out
}

func cloneSnap(s RunSnapshot) RunSnapshot {
	cp := s
	if s.Todos != nil {
		cp.Todos = append([]TodoItem(nil), s.Todos...)
	}
	if s.Children != nil {
		cp.Children = append([]string(nil), s.Children...)
	}
	if s.Drift != nil {
		cp.Drift = append([]DriftSignal(nil), s.Drift...)
	}
	return cp
}

// Compile-time check: store.Store implements TodoLoader.
var _ TodoLoader = (store.Store)(nil)
