// Package seed ensures default provider and agent exist on first run.
package seed

import (
	"context"
	"log/slog"

	"mira/internal/id"
	"mira/internal/store"
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
		ID:          id.New(),
		Key:         "default",
		DisplayName: "Default Agent",
		Provider:    prov,
		Model:       "gpt-4o-mini",
	}
	if err := st.CreateAgent(ctx, ag); err != nil {
		return err
	}
	slog.Info("seed created default agent", "provider", prov)
	return nil
}
