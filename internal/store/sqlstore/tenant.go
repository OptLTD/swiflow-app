package sqlstore

import (
	"context"
	"database/sql"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/tenant"
)

// tid returns the active tenant id from ctx.
func tid(ctx context.Context) string { return tenant.ID(ctx) }

const settingsSep = "\x1f"

// settingsKey builds a composite sys_settings key: tid + sep + logicalKey.
func settingsKey(ctx context.Context, logicalKey string) string {
	return tid(ctx) + settingsSep + logicalKey
}

// settingsPrefix is the tenant prefix for LIKE filters on sys_settings.
func settingsPrefix(ctx context.Context) string {
	return tid(ctx) + settingsSep
}

func (s *Store) affectedOrNoRows(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type tenantRow struct {
	ID           string `db:"id"`
	Name         string `db:"name"`
	Enabled      dbBool `db:"enabled"`
	PasswordHash string `db:"password_hash"`
	CreatedAt    dbTime `db:"created_at"`
}

func (r tenantRow) toTenant() store.Tenant {
	return store.Tenant{
		ID:           r.ID,
		Name:         r.Name,
		Enabled:      r.Enabled.b,
		PasswordHash: r.PasswordHash,
		CreatedAt:    r.CreatedAt.String(),
	}
}

// EnsureDefaultTenant upserts the default tenant (id=default, name=default).
func (s *Store) EnsureDefaultTenant(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO sys_tenant (id, name, enabled, password_hash)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			enabled = excluded.enabled
	`), tenant.DefaultID, "default", s.boolArg(true), "")
	return err
}

// CreateTenant inserts a new tenant with the given password hash.
func (s *Store) CreateTenant(ctx context.Context, id, name, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO sys_tenant (id, name, enabled, password_hash)
		VALUES (?, ?, ?, ?)
	`), id, name, s.boolArg(true), passwordHash)
	return err
}

// GetTenantByName returns a tenant by unique name.
func (s *Store) GetTenantByName(ctx context.Context, name string) (*store.Tenant, error) {
	var r tenantRow
	if err := s.db.GetContext(ctx, &r, s.sql(`
		SELECT id, name, enabled, password_hash, created_at
		FROM sys_tenant WHERE name = ?
	`), name); err != nil {
		return nil, err
	}
	t := r.toTenant()
	return &t, nil
}

// GetTenantByID returns a tenant by id.
func (s *Store) GetTenantByID(ctx context.Context, id string) (*store.Tenant, error) {
	var r tenantRow
	if err := s.db.GetContext(ctx, &r, s.sql(`
		SELECT id, name, enabled, password_hash, created_at
		FROM sys_tenant WHERE id = ?
	`), id); err != nil {
		return nil, err
	}
	t := r.toTenant()
	return &t, nil
}
