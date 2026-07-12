package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mira/internal/agent"
	"mira/internal/schedule"
	"mira/internal/sesshub"
	"mira/internal/mcpclient"
	"mira/internal/server"
	"mira/internal/skill"
	"mira/internal/store"
	"mira/internal/testutil"
	"mira/internal/tool"
)

type apiEnv struct {
	t      *testing.T
	server *httptest.Server
	token  string
	st     store.Store
}

func newAPIEnv(t *testing.T) *apiEnv {
	t.Helper()
	st := testutil.OpenStore(t)
	testutil.SeedProviderAndAgent(t, st)
	cfg := testutil.TestConfig(t)

	skills := skill.NewCatalog("", "")
	reg := tool.NewRegistry()
	tool.RegisterSkill(reg, skills, st)
	mcpMgr := mcpclient.NewManager(st, reg)
	t.Cleanup(mcpMgr.Close)
	runner := agent.NewRunner(agent.RunnerDeps{Store: st, Tools: reg, Skills: skills})
	events := sesshub.New()
	cron := schedule.New(st, runner, events)

	srv := server.New(cfg, st, runner, reg, skills, mcpMgr, cron, events)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	return &apiEnv{t: t, server: ts, token: cfg.AuthToken, st: st}
}

func (e *apiEnv) do(method, path string, body any) (*http.Response, []byte) {
	e.t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.server.URL+path, r)
	if err != nil {
		e.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+e.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		e.t.Fatal(err)
	}
	return resp, data
}

func TestMCPHTTPValidationAndCRUD(t *testing.T) {
	e := newAPIEnv(t)
	ctx := context.Background()

	resp, _ := e.do(http.MethodPost, "/api/mcp/servers", map[string]any{
		"name": "x",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing transport: %d", resp.StatusCode)
	}

	resp, _ = e.do(http.MethodPost, "/api/mcp/servers", map[string]any{
		"name": "stdio-srv", "transport": "stdio",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing command: %d", resp.StatusCode)
	}

	disabled := false
	resp, body := e.do(http.MethodPost, "/api/mcp/servers", map[string]any{
		"name": "offline", "transport": "stdio", "command": "false",
		"enabled": &disabled,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d %s", resp.StatusCode, body)
	}
	var created store.MCPServer
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatal(err)
	}

	resp, body = e.do(http.MethodGet, "/api/mcp/servers", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: %d", resp.StatusCode)
	}
	var listOut struct {
		Servers []store.MCPServer `json:"servers"`
	}
	if err := json.Unmarshal(body, &listOut); err != nil || len(listOut.Servers) != 1 {
		t.Fatalf("list body: %s err %v", body, err)
	}

	resp, body = e.do(http.MethodGet, "/api/mcp/servers/"+created.ID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get: %d %s", resp.StatusCode, body)
	}

	resp, _ = e.do(http.MethodPut, "/api/mcp/servers/"+created.ID, map[string]any{
		"display_name": "Offline MCP",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update: %d", resp.StatusCode)
	}

	resp, _ = e.do(http.MethodPost, "/api/mcp/reload", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reload: %d", resp.StatusCode)
	}

	resp, _ = e.do(http.MethodDelete, "/api/mcp/servers/"+created.ID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	servers, _ := e.st.ListMCPServers(ctx)
	if len(servers) != 0 {
		t.Fatalf("db still has %d servers", len(servers))
	}
}

func TestChatSupportsSSE(t *testing.T) {
	e := newAPIEnv(t)
	body := []byte(`{"message":"hi","agent_key":"default"}`)
	req, err := http.NewRequest(http.MethodPost, e.server.URL+"/api/sessions/sse-test/chat", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusInternalServerError && bytes.Contains(data, []byte("streaming unsupported")) {
		t.Fatalf("chat handler lost http.Flusher through middleware: %s", data)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type = %q body = %s", ct, data)
	}
}

func TestMCPHTTPUnauthorized(t *testing.T) {
	e := newAPIEnv(t)
	req, _ := http.NewRequest(http.MethodGet, e.server.URL+"/api/mcp/servers", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

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
		"name": "job1", "agent_key": "missing", "message": "hi", "schedule": "@hourly",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown agent: %d", resp.StatusCode)
	}

	resp, body := e.do(http.MethodPost, "/api/cron/jobs", map[string]any{
		"name": "job1", "agent_key": "default", "message": "hi", "schedule": "@hourly",
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
