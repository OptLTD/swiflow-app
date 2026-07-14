// Package window bridges agent window_* tools to the connected Web UI via SSE RPC.
package window

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/OptLTD/swiflow/internal/util"
)

const defaultTimeout = 8 * time.Second

// ClarifyTimeout is how long ask_user / clarify waits for a human reply.
const ClarifyTimeout = 15 * time.Minute

// Event is a ui_request payload streamed to the UI (mapped to agent.Event by the server).
type Event struct {
	Type      string
	ID        string
	Name      string
	Arguments map[string]any
}

// EmitFunc streams a ui_request to the active UI client for a session.
type EmitFunc func(Event)

// FallbackEmit is used when no per-run BindEmit is registered (e.g. queue drain → sesshub).
type FallbackEmit func(sessionKey string, ev Event)

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

// SetFallback registers a fan-out when the session has no BindEmit (auto-continue runs).
func (b *Bridge) SetFallback(fn FallbackEmit) {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.fallback = fn
	b.mu.Unlock()
}

// BindEmit registers the SSE emitter for a session run. Replaces any prior binding.
func (b *Bridge) BindEmit(sessionKey string, emit EmitFunc) {
	if b == nil || sessionKey == "" || emit == nil {
		return
	}
	b.mu.Lock()
	b.emits[sessionKey] = emit
	b.mu.Unlock()
}

// UnbindEmit clears the emitter for a session (call when the run ends).
func (b *Bridge) UnbindEmit(sessionKey string) {
	if b == nil || sessionKey == "" {
		return
	}
	b.mu.Lock()
	delete(b.emits, sessionKey)
	b.mu.Unlock()
}

// Request sends a ui_request to the bound UI and waits for Reply (default 8s).
func (b *Bridge) Request(ctx context.Context, sessionKey, op string, args map[string]any) (string, error) {
	return b.RequestTimeout(ctx, sessionKey, op, args, defaultTimeout)
}

// RequestTimeout is like Request with a custom wait bound.
func (b *Bridge) RequestTimeout(ctx context.Context, sessionKey, op string, args map[string]any, timeout time.Duration) (string, error) {
	if b == nil {
		return "", fmt.Errorf("ui client unavailable")
	}
	if sessionKey == "" {
		return "", fmt.Errorf("ui client unavailable")
	}
	if args == nil {
		args = map[string]any{}
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	id := util.NewID()
	ch := make(chan reply, 1)

	b.mu.Lock()
	emit := b.emits[sessionKey]
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

	ev := Event{
		Type:      "ui_request",
		ID:        id,
		Name:      op,
		Arguments: args,
	}
	if emit != nil {
		emit(ev)
	} else {
		fallback(sessionKey, ev)
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
