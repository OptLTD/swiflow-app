package sesshub_test

import (
	"testing"
	"time"

	"github.com/OptLTD/swiflow/internal/agent"
	"github.com/OptLTD/swiflow/internal/sesshub"
)

func TestHubPublishSubscribe(t *testing.T) {
	h := sesshub.New()
	ch, cancel := h.Subscribe("sess-1")
	defer cancel()

	h.Publish("sess-1", agent.Event{Type: "delta", Content: "hi"})
	select {
	case data := <-ch:
		if string(data) == "" {
			t.Fatal("empty payload")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	h.Publish("sess-2", agent.Event{Type: "delta", Content: "other"})
	select {
	case <-ch:
		t.Fatal("should not receive other session events")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestNilHubPublish(t *testing.T) {
	var h *sesshub.Hub
	h.Publish("x", agent.Event{Type: "done"})
}
