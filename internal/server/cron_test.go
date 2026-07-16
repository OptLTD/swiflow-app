package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/OptLTD/swiflow/internal/store"
)

func TestCronHTTPValidationAndCRUD(t *testing.T) {
	e := newAPIEnv(t)
	ctx := context.Background()

	resp, _ := e.do(http.MethodPost, "/api/cron/jobs", map[string]any{
		"name": "job1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing fields: %d", resp.StatusCode)
	}

	resp, _ = e.do(http.MethodPost, "/api/cron/jobs", map[string]any{
		"name": "job1", "agent": "missing", "message": "hi", "schedule": "@hourly",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown agent: %d", resp.StatusCode)
	}

	resp, body := e.do(http.MethodPost, "/api/cron/jobs", map[string]any{
		"name": "job1", "agent": "default", "message": "hi", "schedule": "@hourly",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d %s", resp.StatusCode, body)
	}
	var created store.CronJob
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatal(err)
	}

	resp, body = e.do(http.MethodGet, "/api/cron/jobs", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: %d", resp.StatusCode)
	}
	var listOut struct {
		Jobs []store.CronJob `json:"jobs"`
	}
	if err := json.Unmarshal(body, &listOut); err != nil || len(listOut.Jobs) != 1 {
		t.Fatalf("list: %s", body)
	}

	resp, _ = e.do(http.MethodPut, "/api/cron/jobs/"+created.ID, map[string]any{
		"enabled": false,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update: %d", resp.StatusCode)
	}

	resp, _ = e.do(http.MethodPost, "/api/cron/reload", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reload: %d", resp.StatusCode)
	}

	got, _ := e.st.GetCronJobByID(ctx, created.ID)
	if got.Enabled {
		t.Fatal("expected disabled in db")
	}

	resp, _ = e.do(http.MethodDelete, "/api/cron/jobs/"+created.ID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	jobs, _ := e.st.ListCronJobs(ctx)
	if len(jobs) != 0 {
		t.Fatalf("db still has %d jobs", len(jobs))
	}
}
