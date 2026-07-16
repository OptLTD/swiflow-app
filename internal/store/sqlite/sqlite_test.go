package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/OptLTD/swiflow/embed"
	"github.com/OptLTD/swiflow/internal/migrate"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/store/sqlite"
)

func openTestDB(t *testing.T) *sqlite.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	upgrades, err := embed.UpgradesDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate.Apply(context.Background(), st.DB(), embed.SchemaSQL, upgrades); err != nil {
		t.Fatal(err)
	}
	return st
}

func TestProviderCreds(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	p := &store.Provider{
		ID: "p1", Name: "openai", ApiBase: "https://api.openai.com/v1",
		ApiKey: "sk-test", Enabled: true,
	}
	if err := st.CreateProvider(ctx, p); err != nil {
		t.Fatal(err)
	}
	base, key, _, err := st.ProviderCreds(ctx, "openai")
	if err != nil {
		t.Fatal(err)
	}
	if base != p.ApiBase || key != "sk-test" {
		t.Fatalf("creds mismatch: %s %s", base, key)
	}
	if err := st.UpdateProvider(ctx, "p1", map[string]any{"enabled": false}); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := st.ProviderCreds(ctx, "openai"); err == nil {
		t.Fatal("expected disabled provider error")
	}
}

func TestMessageSeqMonotonic(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	sess := &store.Session{ID: "s1", Agent: "default"}
	if err := st.CreateSession(ctx, sess); err != nil {
		t.Fatal(err)
	}
	for i, mid := range []string{"m1", "m2", "m3"} {
		if _, err := st.AppendMessage(ctx, "s1", store.Message{ID: mid, Role: "user", Content: "hi"}); err != nil {
			t.Fatal(err)
		}
		_ = i
	}
	msgs, err := st.ListMessages(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 || msgs[0].Seq != 1 || msgs[2].Seq != 3 {
		t.Fatalf("seq: %+v", msgs)
	}
}

func TestGetProviderByID(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	p := &store.Provider{ID: "pid", Name: "local", ApiBase: "http://127.0.0.1:8080/v1", ApiKey: "k", Enabled: true}
	if err := st.CreateProvider(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetProviderByID(ctx, "pid")
	if err != nil || got.Name != "local" {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestUpdateToolMessageByCallID(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	if err := st.CreateSession(ctx, &store.Session{ID: "s1", Agent: "default"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendMessage(ctx, "s1", store.Message{
		ID: "t1", Role: "tool", Content: "running", ToolCallId: "call-9", ToolName: "x",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateToolMessageByCallID(ctx, "s1", "call-9", "done"); err != nil {
		t.Fatal(err)
	}
	msgs, err := st.ListMessages(ctx, "s1")
	if err != nil || len(msgs) != 1 || msgs[0].Content != "done" {
		t.Fatalf("got %+v err %v", msgs, err)
	}
}
