package tool_test

import (
	"context"
	"testing"
	"time"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/tool"
)

type fakeSched struct {
	runs []struct {
		session  string
		agentKey string
		message  string
		after    time.Duration
		tid      string
	}
}

func (f *fakeSched) ScheduleRun(sessionKey, agentKey, message string, after time.Duration, tid string) {
	f.runs = append(f.runs, struct {
		session  string
		agentKey string
		message  string
		after    time.Duration
		tid      string
	}{sessionKey, agentKey, message, after, tid})
}

func (f *fakeSched) AddJob(ctx context.Context, job *store.CronJob) error {
	return nil
}

type fakeStore struct {
	agents map[string]bool
}

func (f *fakeStore) GetAgentByKey(ctx context.Context, key string) (*store.Agent, error) {
	if f.agents[key] {
		return &store.Agent{Key: key}, nil
	}
	return nil, context.Canceled
}

func TestScheduleRunTool(t *testing.T) {
	sched := &fakeSched{}
	reg := tool.NewRegistry()
	tool.RegisterSchedule(reg, &fakeStore{agents: map[string]bool{"default": true}}, sched)

	tl, ok := reg.Get("schedule_run")
	if !ok {
		t.Fatal("schedule_run not registered")
	}
	ctx := tool.WithRunContext(context.Background(), tool.RunContext{
		SessionID: "chat-1",
		Agent:     "default",
	})
	out, err := tl.Execute(ctx, map[string]any{
		"delay_seconds": float64(30),
		"message":       "请回复用户：你好吗,天气好吗",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sched.runs) != 1 {
		t.Fatalf("expected 1 scheduled run, got %d", len(sched.runs))
	}
	run := sched.runs[0]
	if run.session != "chat-1" || run.agentKey != "default" {
		t.Fatalf("run: %+v", run)
	}
	if run.message != "请回复用户：你好吗,天气好吗" {
		t.Fatalf("message = %q", run.message)
	}
	if run.after != 30*time.Second {
		t.Fatalf("after = %v", run.after)
	}
	if out == "" {
		t.Fatal("expected confirmation message")
	}
}
