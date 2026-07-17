package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/OptLTD/swiflow/internal/store"
)

// Child (subagent) sessions carry a non-empty parent and must not appear in the
// top-level session list, while still being fetchable individually.
func TestListSessionsHidesChildren(t *testing.T) {
	e := newAPIEnv(t)
	ctx := context.Background()

	if err := e.st.CreateSession(ctx, &store.Session{ID: "root-1", Agent: "default"}); err != nil {
		t.Fatal(err)
	}
	if err := e.st.CreateSession(ctx, &store.Session{ID: "sub-root-1-abcd", Agent: "default", Parent: "root-1"}); err != nil {
		t.Fatal(err)
	}

	resp, data := e.do("GET", "/api/sessions", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, data)
	}
	var out struct {
		Sessions []store.Session `json:"sessions"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, s := range out.Sessions {
		if s.Parent != "" {
			t.Fatalf("child session leaked into list: %+v", s)
		}
		if s.ID == "sub-root-1-abcd" {
			t.Fatalf("subagent session must be hidden from list")
		}
	}
	var sawRoot bool
	for _, s := range out.Sessions {
		if s.ID == "root-1" {
			sawRoot = true
		}
	}
	if !sawRoot {
		t.Fatal("top-level session missing from list")
	}

	// Child still reachable directly.
	resp2, data2 := e.do("GET", "/api/sessions/sub-root-1-abcd", nil)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("getSession child status=%d body=%s", resp2.StatusCode, data2)
	}
}

func TestDeleteSession(t *testing.T) {
	e := newAPIEnv(t)
	ctx := context.Background()

	if err := e.st.CreateSession(ctx, &store.Session{ID: "del-root", Agent: "default", Title: "to delete"}); err != nil {
		t.Fatal(err)
	}
	if err := e.st.CreateSession(ctx, &store.Session{ID: "del-child", Agent: "default", Parent: "del-root"}); err != nil {
		t.Fatal(err)
	}

	resp, data := e.do("DELETE", "/api/sessions/del-root", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, data)
	}

	resp2, _ := e.do("GET", "/api/sessions/del-root", nil)
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("root still present status=%d", resp2.StatusCode)
	}
	resp3, _ := e.do("GET", "/api/sessions/del-child", nil)
	if resp3.StatusCode != http.StatusNotFound {
		t.Fatalf("child still present status=%d", resp3.StatusCode)
	}

	resp4, data4 := e.do("DELETE", "/api/sessions/missing", nil)
	if resp4.StatusCode != http.StatusNotFound {
		t.Fatalf("missing delete status=%d body=%s", resp4.StatusCode, data4)
	}
}
