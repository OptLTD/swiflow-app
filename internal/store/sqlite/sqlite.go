package sqlite

import "github.com/OptLTD/swiflow/internal/store/sqlstore"

type Store = sqlstore.Store

func Open(path string) (*Store, error) {
	return sqlstore.OpenSQLite(path)
}
