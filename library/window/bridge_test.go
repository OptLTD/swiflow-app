package window_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/OptLTD/swiflow/library/window"
)

func TestBridgeRequestReply(t *testing.T) {
	b := window.NewBridge()
	var gotID, gotName string
	var wg sync.WaitGroup
	wg.Add(1)
	b.BindEmit("sess-1", func(ev window.Event) {
		gotID = ev.ID
		gotName = ev.Name
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
			if err := b.Reply(ev.ID, `{"ok":true}`, ""); err != nil {
				t.Errorf("Reply: %v", err)
			}
		}()
	})
	defer b.UnbindEmit("sess-1")

	out, err := b.Request(context.Background(), "sess-1", "window_opened", nil)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	wg.Wait()
	if gotName != "window_opened" {
		t.Fatalf("op = %q", gotName)
	}
	if gotID == "" {
		t.Fatal("empty request id")
	}
	if out != `{"ok":true}` {
		t.Fatalf("result = %q", out)
	}
}

func TestBridgeUnavailable(t *testing.T) {
	b := window.NewBridge()
	_, err := b.Request(context.Background(), "sess-x", "window_active", nil)
	if err == nil || err.Error() != "ui client unavailable" {
		t.Fatalf("err = %v", err)
	}
}

func TestBridgeReplyError(t *testing.T) {
	b := window.NewBridge()
	b.BindEmit("s", func(ev window.Event) {
		_ = b.Reply(ev.ID, "", "open failed")
	})
	defer b.UnbindEmit("s")
	_, err := b.Request(context.Background(), "s", "window_open", map[string]any{"path": "a"})
	if err == nil || err.Error() != "open failed" {
		t.Fatalf("err = %v", err)
	}
}

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
