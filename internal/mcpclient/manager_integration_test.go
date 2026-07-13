package mcpclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/OptLTD/swiflow/internal/mcpclient"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/testutil"
	"github.com/OptLTD/swiflow/internal/tool"
)

func startTestMCPServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "test", Version: "1.0"}, nil)
	sdkmcp.AddTool(server, &sdkmcp.Tool{Name: "echo", Description: "echo back"}, func(_ context.Context, _ *sdkmcp.CallToolRequest, _ any) (*sdkmcp.CallToolResult, any, error) {
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: "pong"}},
		}, nil, nil
	})
	handler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server { return server }, nil)
	return httptest.NewServer(handler)
}

func TestManagerSyncStreamableRegistersTool(t *testing.T) {
	ts := startTestMCPServer(t)

	st := testutil.OpenStore(t)
	ctx := context.Background()
	reg := tool.NewRegistry()
	mgr := mcpclient.NewManager(st, reg)
	t.Cleanup(func() {
		mgr.Close()
		ts.CloseClientConnections()
		ts.Close()
	})

	srv := &store.MCPServer{
		ID: "s1", Name: "testsrv", Transport: "streamable",
		URL: ts.URL, Enabled: true,
	}
	if err := st.CreateMCPServer(ctx, srv); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	toolName := mcpclient.ToolName("testsrv", "echo")
	tl, ok := reg.Get(toolName)
	if !ok {
		t.Fatalf("tool %q not registered; have %v", toolName, reg.Names())
	}
	out, err := tl.Execute(ctx, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "pong" {
		t.Fatalf("got %q want pong", out)
	}
}

func TestManagerSyncSkipsDisabledAndReconnects(t *testing.T) {
	ts := startTestMCPServer(t)

	st := testutil.OpenStore(t)
	ctx := context.Background()
	reg := tool.NewRegistry()
	mgr := mcpclient.NewManager(st, reg)
	t.Cleanup(func() {
		mgr.Close()
		ts.CloseClientConnections()
		ts.Close()
	})

	disabled := &store.MCPServer{
		ID: "s0", Name: "off", Transport: "streamable",
		URL: "http://127.0.0.1:1", Enabled: false,
	}
	enabled := &store.MCPServer{
		ID: "s1", Name: "on", Transport: "streamable",
		URL: ts.URL, Enabled: true,
	}
	if err := st.CreateMCPServer(ctx, disabled); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateMCPServer(ctx, enabled); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Get(mcpclient.ToolName("off", "echo")); ok {
		t.Fatal("disabled server should not register tools")
	}
	if _, ok := reg.Get(mcpclient.ToolName("on", "echo")); !ok {
		t.Fatal("enabled server should register tools")
	}

	if err := st.DeleteMCPServer(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Get(mcpclient.ToolName("on", "echo")); ok {
		t.Fatal("tool should be unregistered after delete")
	}
}

func TestManagerSyncToleratesBadServer(t *testing.T) {
	st := testutil.OpenStore(t)
	ctx := context.Background()
	reg := tool.NewRegistry()
	mgr := mcpclient.NewManager(st, reg)
	t.Cleanup(mgr.Close)

	srv := &store.MCPServer{
		ID: "bad", Name: "bad", Transport: "streamable",
		URL: "http://127.0.0.1:1", Enabled: true,
	}
	if err := st.CreateMCPServer(ctx, srv); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Sync(ctx); err != nil {
		t.Fatalf("sync should tolerate connect failure: %v", err)
	}
	if len(reg.Names()) != 0 {
		t.Fatalf("expected no tools, got %v", reg.Names())
	}
}
