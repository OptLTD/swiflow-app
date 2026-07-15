// Package sqlite opens a SQLite-backed store (thin wrapper over sqlstore).
package sqlite

import "github.com/OptLTD/swiflow/internal/store/sqlstore"

// Store is the SQLite-backed store.Store.
type Store = sqlstore.Store

// Open opens (creating if needed) the SQLite database at path.
func Open(path string, encryptionKey string) (*Store, error) {
	return sqlstore.OpenSQLite(path, encryptionKey)
}
