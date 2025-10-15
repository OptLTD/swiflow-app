package config

import (
	"fmt"
	"swiflow/errors"
)

func MySQLDSN() any {
	var host, port, name, user, pass string
	if host = Get("MYSQL_HOST"); host == "" {
		return fmt.Errorf("%w: %s", errors.ErrorConfig, "loss host")
	}
	if port = Get("MYSQL_PORT"); port == "" {
		return fmt.Errorf("%w: %s", errors.ErrorConfig, "loss db port")
	}
	if name = Get("MYSQL_NAME"); name == "" {
		return fmt.Errorf("%w: %s", errors.ErrorConfig, "loss db name")
	}
	if user = Get("MYSQL_USER"); user == "" {
		return fmt.Errorf("%w: %s", errors.ErrorConfig, "loss username")
	}
	if pass = Get("MYSQL_PASS"); pass == "" {
		return fmt.Errorf("%w: %s", errors.ErrorConfig, "loss password")
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, pass, host, port, name)
}
func SQLiteFile() string {
	return GetDataPath("swiflow.db")
}
