// Package seed ensures default provider and agent exist on first run.
package seed

import (
	"context"
	"log/slog"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/util"
)

// EnsureDefaults creates a default agent if the database has none.
func EnsureDefaults(ctx context.Context, st store.Store) error {
	agents, err := st.ListAgents(ctx)
	if err != nil {
		return err
	}
	if len(agents) > 0 {
		return nil
	}
	providers, err := st.ListProviders(ctx)
	if err != nil {
		return err
	}
	if len(providers) == 0 {
		slog.Info("seed skipped: no providers configured")
		return nil
	}
	prov := providers[0].Name
	ag := &store.Agent{
		ID: util.NewID(), Key: "default",
		Provider: prov, Model: "gpt-4o-mini",
		DisplayName: "Default Agent",
	}
	if err := st.CreateAgent(ctx, ag); err != nil {
		return err
	}
	slog.Info("seed created default agent", "provider", prov)
	return nil
}
