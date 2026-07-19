package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/OptLTD/swiflow/internal/store"
)

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
		"name": "stdio-srv", "type": "stdio",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing command: %d", resp.StatusCode)
	}

	disabled := false
	resp, body := e.do(http.MethodPost, "/api/mcp/servers", map[string]any{
		"name": "offline", "type": "stdio", "cmd": "false",
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

	resp, body = e.do(http.MethodPost, "/api/mcp/act", map[string]any{
		"act": "get", "id": created.ID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get: %d %s", resp.StatusCode, body)
	}

	resp, _ = e.do(http.MethodPost, "/api/mcp/act", map[string]any{
		"act": "set", "id": created.ID, "enabled": true,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update: %d", resp.StatusCode)
	}

	resp, _ = e.do(http.MethodPost, "/api/mcp/act", map[string]any{"act": "reload"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reload: %d", resp.StatusCode)
	}

	resp, _ = e.do(http.MethodPost, "/api/mcp/act", map[string]any{
		"act": "del", "id": created.ID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	servers, _ := e.st.ListMCPServers(ctx)
	if len(servers) != 0 {
		t.Fatalf("db still has %d servers", len(servers))
	}
}
