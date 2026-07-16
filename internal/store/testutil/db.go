// Package testutil provides shared helpers for store-related integration tests.
package testutil

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/OptLTD/swiflow/embed"
	"github.com/OptLTD/swiflow/internal/config"
	"github.com/OptLTD/swiflow/internal/migrate"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/store/sqlite"
)

const (
	TestEncryptionKey = "test-encryption-key-16"
	TestAuthToken     = "test-token"
)

// OpenStore opens a migrated SQLite store in a temp directory.
func OpenStore(t *testing.T) *sqlite.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := sqlite.Open(path, TestEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	upgrades, err := embed.UpgradesDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate.Apply(context.Background(), st.DB(), embed.SchemaSQL, upgrades); err != nil {
		t.Fatal(err)
	}
	return st
}

// TestConfig returns minimal server config for HTTP tests.
func TestConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		Host:          "127.0.0.1",
		Port:          8000,
		AuthToken:     TestAuthToken,
		EncryptionKey: TestEncryptionKey,
		WorkspaceDir:  t.TempDir(),
		UserSkillsDir: "",
	}
}

// SeedProviderAndAgent inserts a provider and default agent for cron/API tests.
func SeedProviderAndAgent(t *testing.T, st store.Store) {
	t.Helper()
	ctx := context.Background()
	p := &store.Provider{
		ID: "prov1", Name: "openai",
		ApiKey: "sk-test", Enabled: true,
		ApiBase: "http://127.0.0.1:9/v1",
		Model:   "gpt-4o-mini",
	}
	if err := st.CreateProvider(ctx, p); err != nil {
		t.Fatal(err)
	}
	ag := &store.Agent{
		ID: "ag1", Key: "default", Display: "Default",
		TxtModel: "openai",
	}
	if err := st.CreateAgent(ctx, ag); err != nil {
		t.Fatal(err)
	}
}
