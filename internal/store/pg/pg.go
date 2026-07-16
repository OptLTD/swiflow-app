// Package pg opens a Postgres-backed store (thin wrapper over sqlstore).
package pg

import "github.com/OptLTD/swiflow/internal/store/sqlstore"

// Store is the Postgres-backed store.Store.
type Store = sqlstore.Store

// Open opens a Postgres database using a pgx DSN.
func Open(dsn string) (*Store, error) {
	return sqlstore.OpenPostgres(dsn)
}
