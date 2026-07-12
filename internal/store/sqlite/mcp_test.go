package sqlite_test

import (
	"context"
	"testing"

	"mira/internal/store"
	"mira/internal/testutil"
)

func TestMCPServerCRUD(t *testing.T) {
	st := testutil.OpenStore(t)
	ctx := context.Background()

	srv := &store.MCPServer{
		ID: "m1", Name: "fs", DisplayName: "Filesystem",
		Transport: "stdio", Command: "echo", Args: []string{"hi"},
		Env: map[string]string{"K": "V"}, Enabled: true,
	}
	if err := st.CreateMCPServer(ctx, srv); err != nil {
		t.Fatal(err)
	}

	list, err := st.ListMCPServers(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %d err %v", len(list), err)
	}
	if list[0].Args[0] != "hi" || list[0].Env["K"] != "V" {
		t.Fatalf("decoded: %+v", list[0])
	}

	got, err := st.GetMCPServerByID(ctx, "m1")
	if err != nil || got.Name != "fs" {
		t.Fatalf("get: %+v err %v", got, err)
	}

	if err := st.UpdateMCPServer(ctx, "m1", map[string]any{
		"enabled": false,
		"args":    []any{"--flag"},
		"env":     map[string]any{"X": "Y"},
	}); err != nil {
		t.Fatal(err)
	}
	got, _ = st.GetMCPServerByID(ctx, "m1")
	if got.Enabled || len(got.Args) != 1 || got.Args[0] != "--flag" || got.Env["X"] != "Y" {
		t.Fatalf("updated: %+v", got)
	}

	dup := &store.MCPServer{ID: "m2", Name: "fs", Transport: "stdio", Command: "x", Enabled: true}
	if err := st.CreateMCPServer(ctx, dup); err == nil {
		t.Fatal("expected unique name conflict")
	}

	if err := st.DeleteMCPServer(ctx, "m1"); err != nil {
		t.Fatal(err)
	}
	list, _ = st.ListMCPServers(ctx)
	if len(list) != 0 {
		t.Fatalf("after delete: %d", len(list))
	}
}
