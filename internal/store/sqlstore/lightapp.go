package sqlstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/internal/store"
)

type lightAppRow struct {
	ID          string `db:"id"`
	Tid         string `db:"tid"`
	Name        string `db:"name"`
	Description string `db:"description"`
	Runtime     string `db:"runtime"`
	EntryPoint  string `db:"entry_point"`
	Status      string `db:"status"`
	Port        int    `db:"port"`
	CreatedAt   dbTime `db:"created_at"`
	UpdatedAt   dbTime `db:"updated_at"`
}

func (r lightAppRow) toLightApp() store.LightApp {
	return store.LightApp{
		ID: r.ID, Tid: r.Tid, Name: r.Name, Description: r.Description,
		Runtime: r.Runtime, EntryPoint: r.EntryPoint, Status: r.Status, Port: r.Port,
		CreatedAt: r.CreatedAt.String(), UpdatedAt: r.UpdatedAt.String(),
	}
}

func (s *Store) CreateLightApp(ctx context.Context, a *store.LightApp) error {
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO light_app (id, name, description, runtime, entry_point, status, port)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`), a.ID, a.Name, a.Description, a.Runtime, a.EntryPoint, a.Status, a.Port)
	return err
}

func (s *Store) ListLightApps(ctx context.Context) ([]store.LightApp, error) {
	var rows []lightAppRow
	if err := s.db.SelectContext(ctx, &rows, s.sql(`SELECT * FROM light_app ORDER BY created_at`)); err != nil {
		return nil, err
	}
	out := make([]store.LightApp, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toLightApp())
	}
	return out, nil
}

func (s *Store) GetLightAppByID(ctx context.Context, id string) (*store.LightApp, error) {
	var r lightAppRow
	if err := s.db.GetContext(ctx, &r, s.sql(`SELECT * FROM light_app WHERE id = ?`), id); err != nil {
		return nil, err
	}
	a := r.toLightApp()
	return &a, nil
}

func (s *Store) UpdateLightApp(ctx context.Context, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	allowed := map[string]bool{"name": true, "description": true, "runtime": true, "entry_point": true, "status": true, "port": true}
	setClauses := make([]string, 0, len(fields)+1)
	args := make([]any, 0, len(fields)+2)
	for k, v := range fields {
		if !allowed[k] {
			continue
		}
		setClauses = append(setClauses, k+" = ?")
		args = append(args, v)
	}
	if len(setClauses) == 0 {
		return nil
	}
	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now().UTC().Format("2006-01-02T15:04:05.000Z"))
	args = append(args, id)
	q := fmt.Sprintf("UPDATE light_app SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err := s.db.ExecContext(ctx, s.sql(q), args...)
	return err
}

func (s *Store) DeleteLightApp(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.sql(`DELETE FROM light_app WHERE id = ?`), id)
	return err
}

const lightAppEnvPrefix = "lightapp.env."

func (s *Store) ListLightAppEnv(ctx context.Context) (map[string]string, error) {
	var rows []struct {
		Key   string `db:"key"`
		Value string `db:"value"`
	}
	if err := s.db.SelectContext(ctx, &rows, s.sql(`SELECT key, value FROM sys_settings WHERE key LIKE 'lightapp.env.%'`)); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[strings.TrimPrefix(r.Key, lightAppEnvPrefix)] = r.Value
	}
	return out, nil
}

func (s *Store) SetLightAppEnv(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO sys_settings (key, value, updated_at) VALUES (?, ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`), lightAppEnvPrefix+key, value)
	return err
}

func (s *Store) DeleteLightAppEnv(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, s.sql(`DELETE FROM sys_settings WHERE key = ?`), lightAppEnvPrefix+key)
	return err
}
