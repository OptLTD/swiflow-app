package window_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/OptLTD/swiflow/internal/window"
)

func TestBridgeFallbackEmit(t *testing.T) {
	b := window.NewBridge()
	var got window.Event
	var mu sync.Mutex
	b.SetFallback(func(sessionKey string, ev window.Event) {
		mu.Lock()
		got = ev
		mu.Unlock()
		_ = b.Reply(ev.ID, `{"answer":"yes"}`, "")
	})

	out, err := b.RequestTimeout(context.Background(), "s1", "clarify", map[string]any{
		"question": "ok?",
	}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"answer":"yes"}` {
		t.Fatalf("out=%s", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if got.Name != "clarify" {
		t.Fatalf("name=%s", got.Name)
	}
}
