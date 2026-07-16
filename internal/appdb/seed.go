package appdb

import (
	"context"
	"log/slog"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/library/support"
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
		ID: support.NewID(), Key: "default",
		TxtModel: prov, Display: "Default Agent",
	}
	if err := st.CreateAgent(ctx, ag); err != nil {
		return err
	}
	slog.Info("seed created default agent", "txt_model", prov)
	return nil
}
