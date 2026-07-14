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

// Event is a ui_request payload streamed to the UI (mapped to agent.Event by the server).
type Event struct {
	Type      string
	ID        string
	Name      string
	Arguments map[string]any
}

// EmitFunc streams a ui_request to the active UI client for a session.
type EmitFunc func(Event)

type pending struct {
	ch chan reply
}

type reply struct {
	result string
	err    string
}

// Bridge multiplexes ui_request / reply RPCs between tool execution and the UI.
type Bridge struct {
	mu      sync.Mutex
	emits   map[string]EmitFunc
	pending map[string]*pending
}

// NewBridge creates an empty bridge.
func NewBridge() *Bridge {
	return &Bridge{
		emits:   map[string]EmitFunc{},
		pending: map[string]*pending{},
	}
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

// Request sends a ui_request to the bound UI and waits for Reply.
func (b *Bridge) Request(ctx context.Context, sessionKey, op string, args map[string]any) (string, error) {
	if b == nil {
		return "", fmt.Errorf("ui client unavailable")
	}
	if sessionKey == "" {
		return "", fmt.Errorf("ui client unavailable")
	}
	if args == nil {
		args = map[string]any{}
	}
	id := util.NewID()
	ch := make(chan reply, 1)

	b.mu.Lock()
	emit := b.emits[sessionKey]
	if emit == nil {
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

	emit(Event{
		Type:      "ui_request",
		ID:        id,
		Name:      op,
		Arguments: args,
	})

	timer := time.NewTimer(defaultTimeout)
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
