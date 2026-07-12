// Package sesshub broadcasts agent run events to session watchers.
package sesshub

import (
	"encoding/json"
	"sync"

	"mira/internal/agent"
)

// Hub fans out run events to SSE subscribers keyed by session.
type Hub struct {
	mu   sync.Mutex
	subs map[string]map[chan []byte]struct{}
}

// New creates an empty hub.
func New() *Hub {
	return &Hub{subs: map[string]map[chan []byte]struct{}{}}
}

// Subscribe registers a watcher for sessionKey. The returned cancel removes it.
func (h *Hub) Subscribe(sessionKey string) (<-chan []byte, func()) {
	ch := make(chan []byte, 64)
	h.mu.Lock()
	if h.subs[sessionKey] == nil {
		h.subs[sessionKey] = map[chan []byte]struct{}{}
	}
	h.subs[sessionKey][ch] = struct{}{}
	h.mu.Unlock()
	cancel := func() {
		h.mu.Lock()
		if m := h.subs[sessionKey]; m != nil {
			delete(m, ch)
			if len(m) == 0 {
				delete(h.subs, sessionKey)
			}
		}
		h.mu.Unlock()
	}
	return ch, cancel
}

// Publish sends an event to all watchers of sessionKey. Nil hub is a no-op.
func (h *Hub) Publish(sessionKey string, ev agent.Event) {
	if h == nil {
		return
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.Lock()
	targets := make([]chan []byte, 0, len(h.subs[sessionKey]))
	for ch := range h.subs[sessionKey] {
		targets = append(targets, ch)
	}
	h.mu.Unlock()
	for _, ch := range targets {
		select {
		case ch <- data:
		default:
			// Drop if a slow client blocks; they can reload messages.
		}
	}
}
