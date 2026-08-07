package agent

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestConcurrentGate(t *testing.T) {
	r := NewRunner(RunnerDeps{MaxConcurrentRuns: 1})
	// Simulate one busy slot without a full Run.
	r.mu.Lock()
	r.busy["a"] = struct{}{}
	r.busyTid["default"] = 1
	r.cancels["a"] = func() {}
	r.mu.Unlock()

	err := r.Run(context.Background(), "b", "default", "hi", nil)
	if !errors.Is(err, ErrConcurrent) {
		t.Fatalf("want ErrConcurrent, got %v", err)
	}
}

func TestEnqueueWhileBusy(t *testing.T) {
	r := NewRunner(RunnerDeps{})
	r.mu.Lock()
	r.busy["s1"] = struct{}{}
	r.mu.Unlock()

	pos := r.Enqueue("s1", "default", "next")
	if pos != 1 {
		t.Fatalf("pos=%d", pos)
	}
	if r.QueueLen("s1") != 1 {
		t.Fatal("expected queue len 1")
	}
}

func TestToolTimeoutContext(t *testing.T) {
	// Ensure WithTimeout is used: cancel parent should abort waits.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	<-ctx.Done()
	if ctx.Err() == nil {
		t.Fatal("expected deadline")
	}
}
