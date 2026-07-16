// Package window bridges agent window_* tools to the connected Web UI via SSE RPC.
package window

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/OptLTD/swiflow/library/support"
)

const defaultTimeout = 8 * time.Second

// ClarifyTimeout is how long ask_user / clarify waits for a human reply.
const ClarifyTimeout = 15 * time.Minute

// Event is a ui_request payload streamed to the UI.
type Event struct {
	Type      string
	ID        string
	Name      string
	Arguments map[string]any
}

// EmitFunc streams a ui_request to the active UI client for a session.
type EmitFunc func(Event)

// FallbackEmit is used when no per-run BindEmit is registered.
type FallbackEmit func(sessionID string, ev Event)

type pending struct {
	ch chan reply
}

type reply struct {
	result string
	err    string
}

// Bridge multiplexes ui_request / reply RPCs between tool execution and the UI.
type Bridge struct {
	mu       sync.Mutex
	emits    map[string]EmitFunc
	pending  map[string]*pending
	fallback FallbackEmit
}

// NewBridge creates an empty bridge.
func NewBridge() *Bridge {
	return &Bridge{
		emits:   map[string]EmitFunc{},
		pending: map[string]*pending{},
	}
}

// SetFallback registers a fan-out when the session has no BindEmit.
func (b *Bridge) SetFallback(fn FallbackEmit) {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.fallback = fn
	b.mu.Unlock()
}

// BindEmit registers the SSE emitter for a session run.
func (b *Bridge) BindEmit(sessionID string, emit EmitFunc) {
	if b == nil || sessionID == "" || emit == nil {
		return
	}
	b.mu.Lock()
	b.emits[sessionID] = emit
	b.mu.Unlock()
}

// UnbindEmit clears the emitter for a session.
func (b *Bridge) UnbindEmit(sessionID string) {
	if b == nil || sessionID == "" {
		return
	}
	b.mu.Lock()
	delete(b.emits, sessionID)
	b.mu.Unlock()
}

// Request sends a ui_request to the bound UI and waits for Reply.
func (b *Bridge) Request(ctx context.Context, sessionID, op string, args map[string]any) (string, error) {
	return b.RequestTimeout(ctx, sessionID, op, args, defaultTimeout)
}

// RequestTimeout is like Request with a custom wait bound.
func (b *Bridge) RequestTimeout(ctx context.Context, sessionID, op string, args map[string]any, timeout time.Duration) (string, error) {
	if b == nil || sessionID == "" {
		return "", fmt.Errorf("ui client unavailable")
	}
	if args == nil {
		args = map[string]any{}
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	id := support.NewID()
	ch := make(chan reply, 1)

	b.mu.Lock()
	emit := b.emits[sessionID]
	fallback := b.fallback
	if emit == nil && fallback == nil {
		b.mu.Unlock()
		return "", fmt.Errorf("ui client unavailable")
	}
	b.pending[id] = &pending{ch: ch}
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
	}()

	ev := Event{Type: "ui_request", ID: id, Name: op, Arguments: args}
	if emit != nil {
		emit(ev)
	} else {
		fallback(sessionID, ev)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-timer.C:
		return "", fmt.Errorf("ui client timeout")
	case r := <-ch:
		if r.err != "" {
			return "", fmt.Errorf("%s", r.err)
		}
		return r.result, nil
	}
}

// Reply completes a pending Request identified by id.
func (b *Bridge) Reply(id, result, errMsg string) error {
	if b == nil {
		return fmt.Errorf("ui client unavailable")
	}
	if id == "" {
		return fmt.Errorf("id required")
	}
	b.mu.Lock()
	p := b.pending[id]
	b.mu.Unlock()
	if p == nil {
		return fmt.Errorf("unknown request id")
	}
	select {
	case p.ch <- reply{result: result, err: errMsg}:
		return nil
	default:
		return fmt.Errorf("reply already sent")
	}
}
