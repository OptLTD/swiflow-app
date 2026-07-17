package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OptLTD/swiflow/internal/agent"
	"github.com/OptLTD/swiflow/internal/mcpclient"
	"github.com/OptLTD/swiflow/internal/schedule"
	"github.com/OptLTD/swiflow/internal/server"
	"github.com/OptLTD/swiflow/internal/skill"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/store/testutil"
	"github.com/OptLTD/swiflow/internal/tool"
)

type apiEnv struct {
	t      *testing.T
	server *httptest.Server
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
	events := server.NewSessionHub()
	runner := agent.NewRunner(agent.RunnerDeps{
		Store: st, Tools: reg, Skills: skills,
	})
	cron := schedule.New(st, runner, events)

	srv := server.New(cfg, st, runner, reg, skills, mcpMgr, cron, events, nil, nil, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	return &apiEnv{t: t, server: ts, st: st}
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
