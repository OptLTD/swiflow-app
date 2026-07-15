package schedule_test

import (
	"context"
	"testing"
	"time"

	"github.com/OptLTD/swiflow/internal/agent"
	"github.com/OptLTD/swiflow/internal/schedule"
	"github.com/OptLTD/swiflow/internal/skill"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/testutil"
	"github.com/OptLTD/swiflow/internal/tool"
)

func newTestScheduler(t *testing.T, st store.Store) *schedule.Scheduler {
	t.Helper()
	skills := skill.NewCatalog("", "")
	reg := tool.NewRegistry()
	runner := agent.NewRunner(agent.RunnerDeps{
		Store: st, Tools: reg, Skills: skills,
	})
	return schedule.New(st, runner, nil)
}

func TestSchedulerReloadValidAndInvalidSchedules(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	ctx := context.Background()
	sched := newTestScheduler(t, st)

	if err := st.CreateCronJob(ctx, &store.CronJob{
		ID: "ok", Name: "ok", Agent: "default",
		Message: "hi", Schedule: "@hourly", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateCronJob(ctx, &store.CronJob{
		ID: "bad", Name: "bad", Agent: "default",
		Message: "hi", Schedule: "not-valid", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateCronJob(ctx, &store.CronJob{
		ID: "off", Name: "off", Agent: "default",
		Message: "hi", Schedule: "@hourly", Enabled: false,
	}); err != nil {
		t.Fatal(err)
	}

	if err := sched.Reload(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestSchedulerRunsJobAndSetsLastRunAt(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	ctx := context.Background()
	sched := newTestScheduler(t, st)

	job := &store.CronJob{
		ID: "tick", Name: "tick", Agent: "default",
		Message: "cron ping", Schedule: "@every 100ms", Enabled: true,
	}
	if err := st.CreateCronJob(ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := sched.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer sched.Stop()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := st.GetCronJobByID(ctx, "tick")
		if err != nil {
			t.Fatal(err)
		}
		if got.LastRunAt != "" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("last_run_at was not set after scheduled runs")
}

func TestSchedulerReloadAfterDelete(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	ctx := context.Background()
	sched := newTestScheduler(t, st)

	if err := st.CreateCronJob(ctx, &store.CronJob{
		ID: "gone", Name: "gone", Agent: "default",
		Message: "x", Schedule: "@hourly", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := sched.Reload(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteCronJob(ctx, "gone"); err != nil {
		t.Fatal(err)
	}
	if err := sched.Reload(ctx); err != nil {
		t.Fatal(err)
	}
}
