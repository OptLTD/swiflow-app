package appdb_test

import (
	"context"
	"os"
	"testing"

	"github.com/OptLTD/swiflow/internal/appdb"
	"github.com/OptLTD/swiflow/internal/config"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/library/support"
)

func TestPostgresCRUDSmoke(t *testing.T) {
	dsn := os.Getenv("SWIFLOW_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("set SWIFLOW_TEST_PG_DSN to run Postgres smoke test")
	}
	cfg := config.Config{
		DBDriver:      "postgres",
		DBDSN:         dsn,
		EncryptionKey: "test-encryption-key-16",
	}
	ctx := context.Background()
	st, err := appdb.MigrateAndOpen(ctx, cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	p := &store.Provider{
		ID: support.NewID(), Name: "pg-smoke",
		ApiBase: "http://127.0.0.1:9/v1", ApiKey: "sk-test",
		Model: "gpt-4o-mini", Enabled: true,
	}
	if err := st.CreateProvider(ctx, p); err != nil {
		t.Fatal(err)
	}
	base, key, model, err := st.ProviderCreds(ctx, "pg-smoke")
	if err != nil || base != p.ApiBase || key != "sk-test" || model != "gpt-4o-mini" {
		t.Fatalf("creds: %s %s %s err=%v", base, key, model, err)
	}
	a := &store.Agent{
		ID: support.NewID(), Key: "pg-default", TxtModel: "pg-smoke", Display: "PG",
	}
	if err := st.CreateAgent(ctx, a); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetAgentByKey(ctx, "pg-default")
	if err != nil || got.TxtModel != "pg-smoke" {
		t.Fatalf("agent: %+v err %v", got, err)
	}
}
