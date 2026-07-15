// Package sesshub broadcasts agent run events to session watchers.
package sesshub

import (
	"encoding/json"
	"sync"

	"github.com/OptLTD/swiflow/internal/agent"
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

// Subscribe registers a watcher for sessionID. The returned cancel removes it.
func (h *Hub) Subscribe(sessionID string) (<-chan []byte, func()) {
	ch := make(chan []byte, 64)
	h.mu.Lock()
	if h.subs[sessionID] == nil {
		h.subs[sessionID] = map[chan []byte]struct{}{}
	}
	h.subs[sessionID][ch] = struct{}{}
	h.mu.Unlock()
	cancel := func() {
		h.mu.Lock()
		if m := h.subs[sessionID]; m != nil {
			delete(m, ch)
			if len(m) == 0 {
				delete(h.subs, sessionID)
			}
		}
		h.mu.Unlock()
	}
	return ch, cancel
}

// Publish sends an event to all watchers of sessionID. Nil hub is a no-op.
func (h *Hub) Publish(sessionID string, ev agent.Event) {
	if h == nil {
		return
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.Lock()
	targets := make([]chan []byte, 0, len(h.subs[sessionID]))
	for ch := range h.subs[sessionID] {
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
