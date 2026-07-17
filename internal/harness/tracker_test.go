package harness

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/OptLTD/swiflow/internal/agent"
)

type memHub struct {
	mu   sync.Mutex
	evs  []pubEv
}

type pubEv struct {
	sid string
	ev  agent.Event
}

func (h *memHub) Publish(sessionID string, ev agent.Event) {
	h.mu.Lock()
	h.evs = append(h.evs, pubEv{sid: sessionID, ev: ev})
	h.mu.Unlock()
}

func (h *memHub) warns() []pubEv {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []pubEv
	for _, e := range h.evs {
		if e.ev.Type == "harness_warn" {
			out = append(out, e)
		}
	}
	return out
}

func TestTrackerGoalAndDelegateChild(t *testing.T) {
	hub := &memHub{}
	tr := NewTracker(hub, nil)
	defer tr.Close()

	tr.Publish("root-1", agent.Event{Type: "user", Content: "OCR @/a.png @/b.png 汇总成 csv"})
	tr.Publish("root-1", agent.Event{
		Type: "tool_call", Name: "delegate_task", ID: "c1",
		Arguments: map[string]any{"goal": "extract all images to result.csv", "max_rounds": float64(8)},
	})
	tr.Publish("root-1", agent.Event{
		Type: "tool_progress", ID: "c1", Child: "sub-root-1-abc", Content: "document_extract",
	})

	snap, ok := tr.Snapshot("root-1")
	if !ok {
		t.Fatal("missing root")
	}
	if snap.Goal == "" || snap.Status != StatusRunning {
		t.Fatalf("root=%+v", snap)
	}
	if len(snap.Children) != 1 || snap.Children[0] != "sub-root-1-abc" {
		t.Fatalf("children=%v", snap.Children)
	}
	child, ok := tr.Snapshot("sub-root-1-abc")
	if !ok || child.ParentID != "root-1" {
		t.Fatalf("child=%+v ok=%v", child, ok)
	}
	if child.MaxRounds != 8 {
		t.Fatalf("max_rounds=%d", child.MaxRounds)
	}
	if child.Goal == "" {
		t.Fatal("expected child goal from delegate")
	}
}

func TestDriftDoneWithOpenTodos(t *testing.T) {
	hub := &memHub{}
	tr := NewTracker(hub, nil)
	defer tr.Close()

	tr.Publish("s1", agent.Event{Type: "user", Content: "do stuff"})
	tr.mu.Lock()
	st := tr.runs["s1"]
	st.snap.Todos = []TodoItem{{ID: "1", Text: "open", Done: false}}
	tr.mu.Unlock()

	tr.Publish("s1", agent.Event{Type: "done"})

	warns := hub.warns()
	found := false
	for _, w := range warns {
		if w.ev.Name == DriftDoneOpenTodos {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected done_with_open_todos, warns=%v", warns)
	}
	snap, _ := tr.Snapshot("s1")
	if len(snap.Drift) == 0 {
		t.Fatal("drift should remain on snapshot")
	}
}

func TestDriftGoalToolMismatch(t *testing.T) {
	hub := &memHub{}
	tr := NewTracker(hub, nil)
	defer tr.Close()

	tr.Publish("s2", agent.Event{Type: "user", Content: "对 @/1.png @/2.png OCR 汇总表格"})
	for i := 0; i < 4; i++ {
		tr.Publish("s2", agent.Event{Type: "tool_call", Name: "fs_list", ID: "t"})
		tr.Publish("s2", agent.Event{Type: "tool_result", Name: "fs_list", ID: "t"})
		// force new rounds
		tr.mu.Lock()
		tr.runs["s2"].inToolRound = false
		tr.mu.Unlock()
	}

	warns := hub.warns()
	found := false
	for _, w := range warns {
		if w.ev.Name == DriftGoalToolMismatch {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected goal_tool_mismatch, warns=%+v", warns)
	}
}

func TestDriftStallRepeatTools(t *testing.T) {
	hub := &memHub{}
	tr := NewTracker(hub, nil)
	defer tr.Close()

	tr.Publish("s3", agent.Event{Type: "user", Content: "hi"})
	for i := 0; i < 3; i++ {
		tr.Publish("s3", agent.Event{Type: "tool_call", Name: "exec", ID: "e"})
	}
	warns := hub.warns()
	found := false
	for _, w := range warns {
		if w.ev.Name == DriftStallRepeatTools {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected stall_repeat_tools, got %#v", warns)
	}
}

func TestListRootsHideChildren(t *testing.T) {
	hub := &memHub{}
	tr := NewTracker(hub, nil)
	defer tr.Close()

	tr.Publish("root", agent.Event{Type: "user", Content: "g"})
	tr.Publish("root", agent.Event{Type: "tool_progress", Child: "sub-x", Content: "x"})
	list := tr.List(true)
	for _, s := range list {
		if s.SessionID == "sub-x" {
			t.Fatal("child must not appear as root in List")
		}
	}
	kids := tr.ListChildren("root")
	if len(kids) != 1 {
		t.Fatalf("children=%d", len(kids))
	}
}

func TestLooksLikeBatchGoal(t *testing.T) {
	if !looksLikeBatchGoal("汇总 @/a.png") {
		t.Fatal("expected true")
	}
	if looksLikeBatchGoal("hello") {
		t.Fatal("expected false")
	}
}

func TestDelegateResultJSON(t *testing.T) {
	hub := &memHub{}
	tr := NewTracker(hub, nil)
	defer tr.Close()

	tr.Publish("p", agent.Event{Type: "user", Content: "go"})
	tr.Publish("p", agent.Event{
		Type: "tool_call", Name: "delegate_task",
		Arguments: map[string]any{"goal": "child goal", "max_rounds": float64(6)},
	})
	body, _ := json.Marshal(map[string]any{
		"child_session": "sub-p-1",
		"status":        "budget",
		"summary":       "stopped",
		"metrics":       map[string]any{"rounds": 6, "tool_calls": 10, "failures": 1},
	})
	tr.Publish("p", agent.Event{Type: "tool_result", Name: "delegate_task", Result: string(body)})

	child, ok := tr.Snapshot("sub-p-1")
	if !ok || child.Status != StatusBudget {
		t.Fatalf("child=%+v", child)
	}
	if child.Metrics.Rounds != 6 {
		t.Fatalf("metrics=%+v", child.Metrics)
	}
}

func TestNoHarnessWarnReentry(t *testing.T) {
	hub := &memHub{}
	tr := NewTracker(hub, nil)
	defer tr.Close()
	// Publishing harness_warn through Tracker should not create run state noise.
	tr.Publish("x", agent.Event{Type: "harness_warn", Name: "test", Content: "ignore"})
	if _, ok := tr.Snapshot("x"); ok {
		// ensureLocked is NOT called for harness_warn
		t.Fatal("harness_warn should not create snapshot")
	}
	time.Sleep(10 * time.Millisecond)
}
