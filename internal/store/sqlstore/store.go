// Package sqlstore is a shared SQL implementation of store.Store for SQLite and Postgres.
package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/OptLTD/swiflow/internal/store"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const (
	DialectSQLite   = "sqlite"
	DialectPostgres = "postgres"
)

// Store is a sqlx-backed store.Store.
type Store struct {
	db     *sqlx.DB
	now    string
	driver string
}

// OpenSQLite opens (creating if needed) the SQLite database at path.
func OpenSQLite(path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(8)
	return &Store{
		db: db, now: nowSQLite, driver: DialectSQLite,
	}, nil
}

// OpenPostgres opens a Postgres database using a pgx DSN
// (e.g. postgres://user:pass@localhost:5432/swiflow?sslmode=disable).
func OpenPostgres(dsn string) (*Store, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(16)
	return &Store{
		db: db, now: nowPostgres, driver: DialectPostgres,
	}, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// DB exposes the underlying connection for the migrator.
func (s *Store) DB() *sql.DB { return s.db.DB }

// Driver returns "sqlite" or "postgres".
func (s *Store) Driver() string { return s.driver }

// sql rewrites dialect tokens and rebinds ?.
func (s *Store) sql(q string) string {
	q = strings.ReplaceAll(q, nowToken, s.now)
	if s.driver == DialectPostgres {
		q = quotePGTypeColumn(q)
		q = strings.ReplaceAll(q, "ON CONFLICT(", "ON CONFLICT (")
	}
	return s.db.Rebind(q)
}

func (s *Store) sqlTx(tx *sqlx.Tx, q string) string {
	q = strings.ReplaceAll(q, nowToken, s.now)
	if s.driver == DialectPostgres {
		q = quotePGTypeColumn(q)
		q = strings.ReplaceAll(q, "ON CONFLICT(", "ON CONFLICT (")
	}
	return tx.Rebind(q)
}

// quotePGTypeColumn quotes the reserved "type" column used by mcp_server.
func quotePGTypeColumn(q string) string {
	repls := []struct{ old, new string }{
		{", type,", `, "type",`},
		{", type = ?", `, "type" = ?`},
		{" type = ?", ` "type" = ?`},
		{"(type,", `("type",`},
	}
	for _, r := range repls {
		q = strings.ReplaceAll(q, r.old, r.new)
	}
	return q
}

// --- Providers ---

func (s *Store) CreateProvider(ctx context.Context, p *store.Provider) error {
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO llm_provider (id, name, display, api_base, api_key, model, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`), p.ID, p.Name, p.Display, p.ApiBase, []byte(p.ApiKey), p.Model, s.boolArg(p.Enabled))
	return err
}

func (s *Store) ListProviders(ctx context.Context) ([]store.Provider, error) {
	var rows []providerRow
	if err := s.db.SelectContext(ctx, &rows, s.sql(`SELECT * FROM llm_provider ORDER BY created_at`)); err != nil {
		return nil, err
	}
	out := make([]store.Provider, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toProvider())
	}
	return out, nil
}

func (s *Store) GetProviderByName(ctx context.Context, name string) (*store.Provider, error) {
	var r providerRow
	if err := s.db.GetContext(ctx, &r, s.sql(`SELECT * FROM llm_provider WHERE name = ?`), name); err != nil {
		return nil, err
	}
	p := r.toProvider()
	return &p, nil
}

// ProviderAPIKey returns the API key for a provider by name.
func (s *Store) ProviderAPIKey(ctx context.Context, name string) (string, error) {
	var blob []byte
	if err := s.db.GetContext(ctx, &blob, s.sql(`SELECT api_key FROM llm_provider WHERE name = ?`), name); err != nil {
		return "", err
	}
	return string(blob), nil
}

func (s *Store) GetProviderByID(ctx context.Context, id string) (*store.Provider, error) {
	var r providerRow
	if err := s.db.GetContext(ctx, &r, s.sql(`SELECT * FROM llm_provider WHERE id = ?`), id); err != nil {
		return nil, err
	}
	p := r.toProvider()
	return &p, nil
}

// ProviderCreds returns the api_base, api_key, and model for an enabled provider.
func (s *Store) ProviderCreds(ctx context.Context, name string) (apiBase, apiKey, model string, err error) {
	var r providerRow
	if err := s.db.GetContext(ctx, &r, s.sql(`SELECT * FROM llm_provider WHERE name = ? AND enabled = 1`), name); err != nil {
		if err == sql.ErrNoRows {
			var disabled providerRow
			if e := s.db.GetContext(ctx, &disabled, s.sql(`SELECT * FROM llm_provider WHERE name = ?`), name); e == nil {
				return "", "", "", fmt.Errorf("provider %q is disabled", name)
			}
		}
		return "", "", "", err
	}
	return r.ApiBase, string(r.ApiKeyBlob), r.Model, nil
}

func (s *Store) UpdateProvider(ctx context.Context, id string, fields map[string]any) error {
	allowed := map[string]bool{
		"display": true, "api_base": true, "api_key": true, "model": true, "enabled": true,
	}
	sets := []string{}
	args := []any{}
	for k, v := range fields {
		if !allowed[k] {
			continue
		}
		switch k {
		case "api_key":
			keyStr, ok := v.(string)
			if !ok {
				return fmt.Errorf("api_key must be a string")
			}
			sets = append(sets, "api_key = ?")
			args = append(args, []byte(keyStr))
		case "enabled":
			b, ok := v.(bool)
			if !ok {
				return fmt.Errorf("enabled must be a boolean")
			}
			sets = append(sets, "enabled = ?")
			args = append(args, s.boolArg(b))
		case "display", "api_base", "model":
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
	q := fmt.Sprintf("UPDATE llm_provider SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, s.sql(q), args...)
	return err
}

func (s *Store) DeleteProvider(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.sql(`DELETE FROM llm_provider WHERE id = ?`), id)
	return err
}

// --- Agents ---

func (s *Store) CreateAgent(ctx context.Context, a *store.Agent) error {
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO agent_config (id, key, display, txt_model, img_model, sys_prompt)
		VALUES (?, ?, ?, ?, ?, ?)
	`), a.ID, a.Key, a.Display, a.TxtModel, a.ImgModel, a.SysPrompt)
	return err
}

func (s *Store) ListAgents(ctx context.Context) ([]store.Agent, error) {
	var rows []agentRow
	if err := s.db.SelectContext(ctx, &rows, s.sql(`SELECT * FROM agent_config ORDER BY created_at`)); err != nil {
		return nil, err
	}
	out := make([]store.Agent, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toAgent())
	}
	return out, nil
}

func (s *Store) GetAgentByKey(ctx context.Context, key string) (*store.Agent, error) {
	var r agentRow
	if err := s.db.GetContext(ctx, &r, s.sql(`SELECT * FROM agent_config WHERE key = ?`), key); err != nil {
		return nil, err
	}
	a := r.toAgent()
	return &a, nil
}

func (s *Store) UpdateAgent(ctx context.Context, id string, fields map[string]any) error {
	allowed := map[string]bool{
		"display": true, "txt_model": true, "img_model": true, "sys_prompt": true,
	}
	sets := []string{}
	args := []any{}
	for k, v := range fields {
		if !allowed[k] {
			continue
		}
		sets = append(sets, k+" = ?")
		args = append(args, v)
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at = datetime('now')")
	args = append(args, id)
	q := fmt.Sprintf("UPDATE agent_config SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, s.sql(q), args...)
	return err
}

// --- Sessions + messages ---

func (s *Store) CreateSession(ctx context.Context, sess *store.Session) error {
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO agent_session (id, agent, title, parent) VALUES (?, ?, ?, ?)
	`), sess.ID, sess.Agent, sess.Title, sess.Parent)
	return err
}

func (s *Store) GetSessionByID(ctx context.Context, id string) (*store.Session, error) {
	var r sessionRow
	if err := s.db.GetContext(ctx, &r, s.sql(`SELECT * FROM agent_session WHERE id = ?`), id); err != nil {
		return nil, err
	}
	sess := r.toSession()
	return &sess, nil
}

func (s *Store) ListSessions(ctx context.Context) ([]store.Session, error) {
	var rows []sessionRow
	if err := s.db.SelectContext(ctx, &rows, s.sql(`SELECT * FROM agent_session ORDER BY updated_at DESC`)); err != nil {
		return nil, err
	}
	out := make([]store.Session, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toSession())
	}
	return out, nil
}

func (s *Store) UpdateSessionTitle(ctx context.Context, id, title string) error {
	_, err := s.db.ExecContext(ctx, s.sql(`UPDATE agent_session SET title = ?, updated_at = datetime('now') WHERE id = ?`), title, id)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("session id required")
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	ids := []string{id}
	var children []string
	if err := tx.SelectContext(ctx, &children, s.sqlTx(tx, `SELECT id FROM agent_session WHERE parent = ?`), id); err != nil {
		return err
	}
	ids = append(ids, children...)

	for _, sid := range ids {
		if _, err := tx.ExecContext(ctx, s.sqlTx(tx, `DELETE FROM agent_message WHERE sid = ?`), sid); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, s.sqlTx(tx, `DELETE FROM agent_experience WHERE sid = ?`), sid); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, s.sqlTx(tx, `DELETE FROM agent_todo WHERE sid = ?`), sid); err != nil {
			return err
		}
	}
	// Children first, then the root.
	for i := len(ids) - 1; i >= 0; i-- {
		if _, err := tx.ExecContext(ctx, s.sqlTx(tx, `DELETE FROM agent_session WHERE id = ?`), ids[i]); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type messageRow struct {
	ID         string `db:"id"`
	Tid        string `db:"tid"`
	Sid        string `db:"sid"`
	Seq        int    `db:"seq"`
	Role       string `db:"role"`
	Content    string `db:"content"`
	Thinking   string `db:"thinking"`
	ToolCalls  dbJSON `db:"tool_calls"`
	ToolCallId string `db:"tool_call_id"`
	ToolName   string `db:"tool_name"`
	CreatedAt  dbTime `db:"created_at"`
}

func (r messageRow) toMessage() store.Message {
	m := store.Message{
		ID: r.ID, Tid: r.Tid, Sid: r.Sid, Seq: r.Seq,
		Role: r.Role, Content: r.Content, Thinking: r.Thinking,
		ToolCallId: r.ToolCallId, ToolName: r.ToolName, CreatedAt: r.CreatedAt.String(),
	}
	raw := r.ToolCalls.s
	if raw == "" {
		raw = "[]"
	}
	if raw != "[]" {
		_ = json.Unmarshal([]byte(raw), &m.ToolCalls)
	}
	return m
}

func encodeToolCalls(calls []store.ToolCall) (string, error) {
	if len(calls) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(calls)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// AppendMessage inserts a message with the next seq for its session, atomically.
func (s *Store) AppendMessage(ctx context.Context, sessionID string, msg store.Message) (store.Message, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return msg, err
	}
	defer tx.Rollback() //nolint:errcheck

	var maxSeq sql.NullInt64
	if err := tx.GetContext(ctx, &maxSeq, s.sqlTx(tx, `SELECT MAX(seq) FROM agent_message WHERE sid = ?`), sessionID); err != nil {
		return msg, err
	}
	msg.Sid = sessionID
	msg.Seq = int(maxSeq.Int64) + 1

	toolCallsJSON, err := encodeToolCalls(msg.ToolCalls)
	if err != nil {
		return msg, err
	}
	_, err = tx.ExecContext(ctx, s.sqlTx(tx, `
		INSERT INTO agent_message (id, sid, seq, role, content, thinking, tool_calls, tool_call_id, tool_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`), msg.ID, msg.Sid, msg.Seq, msg.Role, msg.Content, msg.Thinking, toolCallsJSON, msg.ToolCallId, msg.ToolName)
	if err != nil {
		return msg, err
	}
	if _, err := tx.ExecContext(ctx, s.sqlTx(tx, `UPDATE agent_session SET updated_at = datetime('now') WHERE id = ?`), sessionID); err != nil {
		return msg, err
	}
	if err := tx.Commit(); err != nil {
		return msg, err
	}
	return msg, nil
}

// UpdateToolMessageByCallID updates the content of a tool-role message identified by tool_call_id.
func (s *Store) UpdateToolMessageByCallID(ctx context.Context, sessionID, toolCallID, content string) error {
	if sessionID == "" || toolCallID == "" {
		return fmt.Errorf("sessionID and toolCallID required")
	}
	res, err := s.db.ExecContext(ctx, s.sql(`
		UPDATE agent_message SET content = ? WHERE sid = ? AND tool_call_id = ? AND role = 'tool'
	`), content, sessionID, toolCallID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tool message not found: %s", toolCallID)
	}
	_, _ = s.db.ExecContext(ctx, s.sql(`UPDATE agent_session SET updated_at = datetime('now') WHERE id = ?`), sessionID)
	return nil
}

func (s *Store) ListMessages(ctx context.Context, sessionID string) ([]store.Message, error) {
	var rows []messageRow
	if err := s.db.SelectContext(ctx, &rows, s.sql(`
		SELECT * FROM agent_message WHERE sid = ? ORDER BY seq
	`), sessionID); err != nil {
		return nil, err
	}
	out := make([]store.Message, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toMessage())
	}
	return out, nil
}

// --- Settings ---

func (s *Store) ToolEnabled(ctx context.Context, name string) bool {
	var val string
	err := s.db.GetContext(ctx, &val, s.sql(`SELECT value FROM sys_settings WHERE key = ?`), "tool."+name)
	if err == sql.ErrNoRows {
		return true
	}
	if err != nil {
		return true
	}
	return val == "1"
}

func (s *Store) SetToolEnabled(ctx context.Context, name string, enabled bool) error {
	v := "0"
	if enabled {
		v = "1"
	}
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO sys_settings (key, value, updated_at) VALUES (?, ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`), "tool."+name, v)
	return err
}

func (s *Store) ListToolPolicy(ctx context.Context) ([]store.ToolPolicy, error) {
	var rows []struct {
		Key   string `db:"key"`
		Value string `db:"value"`
	}
	if err := s.db.SelectContext(ctx, &rows, s.sql(`SELECT key, value FROM sys_settings WHERE key LIKE 'tool.%'`)); err != nil {
		return nil, err
	}
	out := make([]store.ToolPolicy, 0, len(rows))
	for _, r := range rows {
		out = append(out, store.ToolPolicy{
			ToolName: strings.TrimPrefix(r.Key, "tool."),
			Enabled:  r.Value == "1",
		})
	}
	return out, nil
}

func (s *Store) DisabledSkills(ctx context.Context) ([]string, error) {
	var rows []struct {
		Key   string `db:"key"`
		Value string `db:"value"`
	}
	if err := s.db.SelectContext(ctx, &rows, s.sql(`SELECT key, value FROM sys_settings WHERE key LIKE 'skill.%' AND value = '0'`)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, strings.TrimPrefix(r.Key, "skill."))
	}
	return out, nil
}

func (s *Store) SetSkillEnabled(ctx context.Context, slug string, enabled bool) error {
	v := "0"
	if enabled {
		v = "1"
	}
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO sys_settings (key, value, updated_at) VALUES (?, ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`), "skill."+slug, v)
	return err
}

func (s *Store) GetSysSetting(ctx context.Context, key string) (string, bool, error) {
	var val string
	err := s.db.GetContext(ctx, &val, s.sql(`SELECT value FROM sys_settings WHERE key = ?`), key)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func (s *Store) SetSysSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO sys_settings (key, value, updated_at) VALUES (?, ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`), key, value)
	return err
}

type providerRow struct {
	ID         string `db:"id"`
	Tid        string `db:"tid"`
	Name       string `db:"name"`
	Display    string `db:"display"`
	ApiBase    string `db:"api_base"`
	ApiKeyBlob []byte `db:"api_key"`
	Model      string `db:"model"`
	Enabled    dbBool `db:"enabled"`
	CreatedAt  dbTime `db:"created_at"`
	UpdatedAt  dbTime `db:"updated_at"`
}

func (r providerRow) toProvider() store.Provider {
	return store.Provider{
		ID:        r.ID,
		Tid:       r.Tid,
		Name:      r.Name,
		Display:   r.Display,
		ApiBase:   r.ApiBase,
		ApiKey:    "",
		Model:     r.Model,
		Enabled:   r.Enabled.b,
		CreatedAt: r.CreatedAt.String(),
		UpdatedAt: r.UpdatedAt.String(),
	}
}

type agentRow struct {
	ID        string `db:"id"`
	Tid       string `db:"tid"`
	Key       string `db:"key"`
	Display   string `db:"display"`
	TxtModel  string `db:"txt_model"`
	ImgModel  string `db:"img_model"`
	SysPrompt string `db:"sys_prompt"`
	CreatedAt dbTime `db:"created_at"`
	UpdatedAt dbTime `db:"updated_at"`
}

func (r agentRow) toAgent() store.Agent {
	return store.Agent{
		ID: r.ID, Tid: r.Tid, Key: r.Key, Display: r.Display,
		TxtModel: r.TxtModel, ImgModel: r.ImgModel, SysPrompt: r.SysPrompt,
		CreatedAt: r.CreatedAt.String(), UpdatedAt: r.UpdatedAt.String(),
	}
}

type sessionRow struct {
	ID        string `db:"id"`
	Tid       string `db:"tid"`
	Agent     string `db:"agent"`
	Title     string `db:"title"`
	Parent    string `db:"parent"`
	CreatedAt dbTime `db:"created_at"`
	UpdatedAt dbTime `db:"updated_at"`
}

func (r sessionRow) toSession() store.Session {
	return store.Session{
		ID: r.ID, Tid: r.Tid, Agent: r.Agent, Title: r.Title, Parent: r.Parent,
		CreatedAt: r.CreatedAt.String(), UpdatedAt: r.UpdatedAt.String(),
	}
}
