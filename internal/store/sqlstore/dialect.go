package sqlstore

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Placeholder in SQL for “now”; rewritten per dialect in Store.sql.
const nowToken = "datetime('now')"

const (
	nowSQLite   = "strftime('%Y-%m-%dT%H:%M:%fZ', 'now')"
	nowPostgres = "NOW()"
)

// boolArg returns 0/1 for both SQLite and Postgres SMALLINT enabled flags.
func (s *Store) boolArg(b bool) any {
	if b {
		return 1
	}
	return 0
}

// dbTime scans SQLite TEXT / Postgres TIMESTAMPTZ into an RFC3339 string.
type dbTime struct {
	s string
}

func (t *dbTime) Scan(src any) error {
	if src == nil {
		t.s = ""
		return nil
	}
	switch v := src.(type) {
	case time.Time:
		t.s = v.UTC().Format(time.RFC3339Nano)
	case string:
		t.s = normalizeTimeString(v)
	case []byte:
		t.s = normalizeTimeString(string(v))
	default:
		return fmt.Errorf("cannot scan %T into dbTime", src)
	}
	return nil
}

func (t dbTime) String() string { return t.s }

func normalizeTimeString(v string) string {
	if v == "" {
		return ""
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, v); err == nil {
			return ts.UTC().Format(time.RFC3339Nano)
		}
	}
	return v
}

// dbNullTime is like dbTime but nullable.
type dbNullTime struct {
	s     string
	valid bool
}

func (t *dbNullTime) Scan(src any) error {
	if src == nil {
		t.s, t.valid = "", false
		return nil
	}
	var d dbTime
	if err := d.Scan(src); err != nil {
		return err
	}
	t.s, t.valid = d.s, true
	return nil
}

// dbBool scans SQLite INTEGER or Postgres BOOLEAN.
type dbBool struct {
	b bool
}

func (b *dbBool) Scan(src any) error {
	if src == nil {
		b.b = false
		return nil
	}
	switch v := src.(type) {
	case bool:
		b.b = v
	case int64:
		b.b = v != 0
	case int32:
		b.b = v != 0
	case int:
		b.b = v != 0
	case []byte:
		s := string(v)
		b.b = s == "1" || s == "t" || s == "true" || s == "TRUE"
	case string:
		b.b = v == "1" || v == "t" || v == "true" || v == "TRUE"
	default:
		return fmt.Errorf("cannot scan %T into dbBool", src)
	}
	return nil
}

// dbJSON scans JSON/JSONB text (or driver-decoded values) into a JSON string.
type dbJSON struct {
	s string
}

func (j *dbJSON) Scan(src any) error {
	if src == nil {
		j.s = ""
		return nil
	}
	switch v := src.(type) {
	case string:
		j.s = v
	case []byte:
		j.s = string(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		j.s = string(b)
	}
	return nil
}

func (j dbJSON) String() string {
	return j.s
}

// envString returns JSON object default for empty values.
func (j dbJSON) envString() string {
	if j.s == "" {
		return "{}"
	}
	return j.s
}

// Value implements driver.Valuer for inserts that need a string.
func (j dbJSON) Value() (driver.Value, error) {
	if j.s == "" {
		return "[]", nil
	}
	return j.s, nil
}
