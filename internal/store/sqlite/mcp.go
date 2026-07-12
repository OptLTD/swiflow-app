package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mira/internal/store"
)

type mcpServerRow struct {
	ID          string `db:"id"`
	Name        string `db:"name"`
	DisplayName string `db:"display_name"`
	Transport   string `db:"transport"`
	Command     string `db:"command"`
	ArgsJSON    string `db:"args_json"`
	URL         string `db:"url"`
	EnvJSON     string `db:"env_json"`
	Enabled     int    `db:"enabled"`
	CreatedAt   string `db:"created_at"`
	UpdatedAt   string `db:"updated_at"`
}

func (r mcpServerRow) toMCPServer() store.MCPServer {
	s := store.MCPServer{
		ID: r.ID, Name: r.Name, DisplayName: r.DisplayName,
		Transport: r.Transport, Command: r.Command, URL: r.URL,
		ArgsJSON: r.ArgsJSON, EnvJSON: r.EnvJSON,
		Enabled: r.Enabled == 1, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	_ = json.Unmarshal([]byte(r.ArgsJSON), &s.Args)
	_ = json.Unmarshal([]byte(r.EnvJSON), &s.Env)
	if s.Args == nil {
		s.Args = []string{}
	}
	if s.Env == nil {
		s.Env = map[string]string{}
	}
	return s
}

func encodeMCPServer(s *store.MCPServer) error {
	if s.Args == nil {
		s.Args = []string{}
	}
	if s.Env == nil {
		s.Env = map[string]string{}
	}
	args, err := json.Marshal(s.Args)
	if err != nil {
		return err
	}
	env, err := json.Marshal(s.Env)
	if err != nil {
		return err
	}
	s.ArgsJSON = string(args)
	s.EnvJSON = string(env)
	return nil
}

func (s *Store) CreateMCPServer(ctx context.Context, srv *store.MCPServer) error {
	if err := encodeMCPServer(srv); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mcp_servers (id, name, display_name, transport, command, args_json, url, env_json, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, srv.ID, srv.Name, srv.DisplayName, srv.Transport, srv.Command, srv.ArgsJSON, srv.URL, srv.EnvJSON, boolToInt(srv.Enabled))
	return err
}

func (s *Store) ListMCPServers(ctx context.Context) ([]store.MCPServer, error) {
	var rows []mcpServerRow
	if err := s.db.SelectContext(ctx, &rows, `SELECT * FROM mcp_servers ORDER BY created_at`); err != nil {
		return nil, err
	}
	out := make([]store.MCPServer, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toMCPServer())
	}
	return out, nil
}

func (s *Store) GetMCPServerByID(ctx context.Context, id string) (*store.MCPServer, error) {
	var r mcpServerRow
	if err := s.db.GetContext(ctx, &r, `SELECT * FROM mcp_servers WHERE id = ?`, id); err != nil {
		return nil, err
	}
	srv := r.toMCPServer()
	return &srv, nil
}

func (s *Store) GetMCPServerByName(ctx context.Context, name string) (*store.MCPServer, error) {
	var r mcpServerRow
	if err := s.db.GetContext(ctx, &r, `SELECT * FROM mcp_servers WHERE name = ?`, name); err != nil {
		return nil, err
	}
	srv := r.toMCPServer()
	return &srv, nil
}

func (s *Store) UpdateMCPServer(ctx context.Context, id string, fields map[string]any) error {
	allowed := map[string]bool{
		"display_name": true, "transport": true, "command": true,
		"args": true, "url": true, "env": true, "enabled": true,
	}
	sets := []string{}
	args := []any{}
	for k, v := range fields {
		if !allowed[k] {
			continue
		}
		switch k {
		case "enabled":
			b, ok := v.(bool)
			if !ok {
				return fmt.Errorf("enabled must be a boolean")
			}
			sets = append(sets, "enabled = ?")
			args = append(args, boolToInt(b))
		case "args":
			raw, err := json.Marshal(v)
			if err != nil {
				return err
			}
			sets = append(sets, "args_json = ?")
			args = append(args, string(raw))
		case "env":
			raw, err := json.Marshal(v)
			if err != nil {
				return err
			}
			sets = append(sets, "env_json = ?")
			args = append(args, string(raw))
		case "display_name", "transport", "command", "url":
			str, ok := v.(string)
			if !ok {
				return fmt.Errorf("%s must be a string", k)
			}
			sets = append(sets, k+" = ?")
			args = append(args, str)
		}
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at = datetime('now')")
	args = append(args, id)
	q := fmt.Sprintf("UPDATE mcp_servers SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, q, args...)
	return err
}

func (s *Store) DeleteMCPServer(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM mcp_servers WHERE id = ?`, id)
	return err
}
