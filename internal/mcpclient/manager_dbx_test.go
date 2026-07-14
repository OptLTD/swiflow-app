package mcpclient_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/OptLTD/swiflow/internal/mcpclient"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/testutil"
	"github.com/OptLTD/swiflow/internal/tool"
)

// TestManagerSyncStdioDBX connects to the local dbx MCP server (stdio).
// Skips when dbx-mcp-server is not installed.
//
// Equivalent Swiflow config:
//
//	{"name":"dbx","transport":"stdio","command":"dbx-mcp-server"}
func TestManagerSyncStdioDBX(t *testing.T) {
	path, err := exec.LookPath("dbx-mcp-server")
	if err != nil {
		t.Skip("dbx-mcp-server not in PATH")
	}

	st := testutil.OpenStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	reg := tool.NewRegistry()
	mgr := mcpclient.NewManager(st, reg)
	t.Cleanup(mgr.Close)

	srv := &store.MCPServer{
		ID: "dbx1", Name: "dbx", Transport: "stdio",
		Command: path, Enabled: true,
	}
	if err := st.CreateMCPServer(ctx, srv); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	names := reg.Names()
	if len(names) == 0 {
		t.Fatal("dbx sync succeeded but no tools registered")
	}
	t.Logf("dbx-mcp-server (%s) registered %d tools:", path, len(names))
	for _, n := range names {
		t.Logf("  - %s", n)
	}

	// Smoke-call the first registered tool with empty args if possible.
	tl, ok := reg.Get(names[0])
	if !ok {
		t.Fatalf("tool %q missing from registry", names[0])
	}
	callCtx, callCancel := context.WithTimeout(ctx, 30*time.Second)
	defer callCancel()
	if _, err := tl.Execute(callCtx, map[string]any{}); err != nil {
		t.Logf("tool %s execute (empty args): %v", names[0], err)
	}
}
