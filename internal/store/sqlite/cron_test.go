package sqlite_test

import (
	"context"
	"testing"

	"mira/internal/store"
	"mira/internal/testutil"
)

func TestCronJobCRUD(t *testing.T) {
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	ctx := context.Background()

	job := &store.CronJob{
		ID: "j1", Name: "daily", AgentKey: "default",
		Message: "ping", Schedule: "0 9 * * *", Enabled: true,
	}
	if err := st.CreateCronJob(ctx, job); err != nil {
		t.Fatal(err)
	}

	list, err := st.ListCronJobs(ctx)
	if err != nil || len(list) != 1 || list[0].Name != "daily" {
		t.Fatalf("list: %+v err %v", list, err)
	}

	if err := st.UpdateCronJob(ctx, "j1", map[string]any{
		"schedule": "@hourly",
		"message":  "updated",
		"enabled":  false,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetCronJobByID(ctx, "j1")
	if err != nil || got.Schedule != "@hourly" || got.Enabled || got.Message != "updated" {
		t.Fatalf("updated: %+v err %v", got, err)
	}

	if err := st.SetCronJobLastRun(ctx, "j1", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	got, _ = st.GetCronJobByID(ctx, "j1")
	if got.LastRunAt != "2026-01-01T00:00:00Z" {
		t.Fatalf("last_run_at: %q", got.LastRunAt)
	}

	dup := &store.CronJob{ID: "j2", Name: "daily", AgentKey: "default", Message: "x", Schedule: "* * * * *"}
	if err := st.CreateCronJob(ctx, dup); err == nil {
		t.Fatal("expected unique name conflict")
	}

	if err := st.DeleteCronJob(ctx, "j1"); err != nil {
		t.Fatal(err)
	}
	list, _ = st.ListCronJobs(ctx)
	if len(list) != 0 {
		t.Fatalf("after delete: %d", len(list))
	}
}
