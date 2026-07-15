package sqlstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/OptLTD/swiflow/internal/store"
)

type mcpServerRow struct {
	ID        string `db:"id"`
	Tid       string `db:"tid"`
	Name      string `db:"name"`
	Type      string `db:"type"`
	Cmd       string `db:"cmd"`
	Args      dbJSON `db:"args"`
	URL       string `db:"url"`
	Env       dbJSON `db:"env"`
	Enabled   dbBool `db:"enabled"`
	CreatedAt dbTime `db:"created_at"`
	UpdatedAt dbTime `db:"updated_at"`
}

func (r mcpServerRow) toMCPServer() store.MCPServer {
	s := store.MCPServer{
		ID: r.ID, Tid: r.Tid, Name: r.Name,
		Type: r.Type, Cmd: r.Cmd, URL: r.URL,
		Enabled: r.Enabled.b, CreatedAt: r.CreatedAt.String(), UpdatedAt: r.UpdatedAt.String(),
	}
	argsRaw := r.Args.s
	if argsRaw == "" {
		argsRaw = "[]"
	}
	_ = json.Unmarshal([]byte(argsRaw), &s.Args)
	_ = json.Unmarshal([]byte(r.Env.envString()), &s.Env)
	if s.Args == nil {
		s.Args = []string{}
	}
	if s.Env == nil {
		s.Env = map[string]string{}
	}
	return s
}

func encodeMCPArgsEnv(s *store.MCPServer) (argsJSON, envJSON string, err error) {
	if s.Args == nil {
		s.Args = []string{}
	}
	if s.Env == nil {
		s.Env = map[string]string{}
	}
	args, err := json.Marshal(s.Args)
	if err != nil {
		return "", "", err
	}
	env, err := json.Marshal(s.Env)
	if err != nil {
		return "", "", err
	}
	return string(args), string(env), nil
}

func (s *Store) CreateMCPServer(ctx context.Context, srv *store.MCPServer) error {
	argsJSON, envJSON, err := encodeMCPArgsEnv(srv)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.sql(`
		INSERT INTO mcp_server (id, name, type, cmd, args, url, env, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`), srv.ID, srv.Name, srv.Type, srv.Cmd, argsJSON, srv.URL, envJSON, s.boolArg(srv.Enabled))
	return err
}

func (s *Store) ListMCPServers(ctx context.Context) ([]store.MCPServer, error) {
	var rows []mcpServerRow
	if err := s.db.SelectContext(ctx, &rows, s.sql(`SELECT * FROM mcp_server ORDER BY created_at`)); err != nil {
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
	if err := s.db.GetContext(ctx, &r, s.sql(`SELECT * FROM mcp_server WHERE id = ?`), id); err != nil {
		return nil, err
	}
	srv := r.toMCPServer()
	return &srv, nil
}

func (s *Store) GetMCPServerByName(ctx context.Context, name string) (*store.MCPServer, error) {
	var r mcpServerRow
	if err := s.db.GetContext(ctx, &r, s.sql(`SELECT * FROM mcp_server WHERE name = ?`), name); err != nil {
		return nil, err
	}
	srv := r.toMCPServer()
	return &srv, nil
}

func (s *Store) UpdateMCPServer(ctx context.Context, id string, fields map[string]any) error {
	allowed := map[string]bool{
		"type": true, "cmd": true,
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
			args = append(args, s.boolArg(b))
		case "args", "env":
			raw, err := json.Marshal(v)
			if err != nil {
				return err
			}
			sets = append(sets, k+" = ?")
			args = append(args, string(raw))
		case "type", "cmd", "url":
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
	sets = append(sets, "updated_at = "+nowToken)
	args = append(args, id)
	q := fmt.Sprintf("UPDATE mcp_server SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, s.sql(q), args...)
	return err
}

func (s *Store) DeleteMCPServer(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.sql(`DELETE FROM mcp_server WHERE id = ?`), id)
	return err
}
