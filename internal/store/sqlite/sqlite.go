// Package sqlite is the SQLite implementation of store.Store. Spec §6.2, §5.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/OptLTD/swiflow/internal/secure"
	"github.com/OptLTD/swiflow/internal/store"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Store is a SQLite-backed store.Store.
type Store struct {
	db  *sqlx.DB
	key []byte // AES-256 key for provider API keys
}

// Open opens (creating if needed) the SQLite database at path.
func Open(path string, encryptionKey string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(8)
	s := &Store{db: db, key: secure.DeriveKey(encryptionKey)}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// DB exposes the underlying connection for the migrator.
func (s *Store) DB() *sql.DB { return s.db.DB }

// --- Providers ---

func (s *Store) CreateProvider(ctx context.Context, p *store.Provider) error {
	enc, err := secure.Encrypt(s.key, []byte(p.APIKey))
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO providers (id, name, display_name, api_base, api_key_enc, enabled)
		VALUES (?, ?, ?, ?, ?, ?)
	`, p.ID, p.Name, p.DisplayName, p.APIBase, enc, boolToInt(p.Enabled))
	return err
}

func (s *Store) ListProviders(ctx context.Context) ([]store.Provider, error) {
	var rows []providerRow
	if err := s.db.SelectContext(ctx, &rows, `SELECT * FROM providers ORDER BY created_at`); err != nil {
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
	if err := s.db.GetContext(ctx, &r, `SELECT * FROM providers WHERE name = ?`, name); err != nil {
		return nil, err
	}
	p := r.toProvider()
	return &p, nil
}

// DecryptAPIKey returns the plaintext API key for a provider by name.
func (s *Store) DecryptAPIKey(ctx context.Context, name string) (string, error) {
	var enc []byte
	if err := s.db.GetContext(ctx, &enc, `SELECT api_key_enc FROM providers WHERE name = ?`, name); err != nil {
		return "", err
	}
	pt, err := secure.Decrypt(s.key, enc)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

func (s *Store) GetProviderByID(ctx context.Context, id string) (*store.Provider, error) {
	var r providerRow
	if err := s.db.GetContext(ctx, &r, `SELECT * FROM providers WHERE id = ?`, id); err != nil {
		return nil, err
	}
	p := r.toProvider()
	return &p, nil
}

// ProviderCreds returns the api_base and plaintext api_key for an enabled provider.
func (s *Store) ProviderCreds(ctx context.Context, name string) (apiBase, apiKey string, err error) {
	var r providerRow
	if err := s.db.GetContext(ctx, &r, `SELECT * FROM providers WHERE name = ? AND enabled = 1`, name); err != nil {
		if err == sql.ErrNoRows {
			var disabled providerRow
			if e := s.db.GetContext(ctx, &disabled, `SELECT * FROM providers WHERE name = ?`, name); e == nil {
				return "", "", fmt.Errorf("provider %q is disabled", name)
			}
		}
		return "", "", err
	}
	pt, err := secure.Decrypt(s.key, r.APIKeyEnc)
	if err != nil {
		return "", "", err
	}
	return r.APIBase, string(pt), nil
}

func (s *Store) UpdateProvider(ctx context.Context, id string, fields map[string]any) error {
	allowed := map[string]bool{
		"display_name": true, "api_base": true, "api_key": true, "enabled": true,
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
			enc, err := secure.Encrypt(s.key, []byte(keyStr))
			if err != nil {
				return err
			}
			sets = append(sets, "api_key_enc = ?")
			args = append(args, enc)
		case "enabled":
			b, ok := v.(bool)
			if !ok {
				return fmt.Errorf("enabled must be a boolean")
			}
			sets = append(sets, "enabled = ?")
			args = append(args, boolToInt(b))
		case "display_name", "api_base":
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("%s must be a string", k)
			}
			sets = append(sets, k+" = ?")
			args = append(args, s)
		}
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at = datetime('now')")
	args = append(args, id)
	q := fmt.Sprintf("UPDATE providers SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, q, args...)
	return err
}

func (s *Store) DeleteProvider(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM providers WHERE id = ?`, id)
	return err
}

// --- Agents ---

func (s *Store) CreateAgent(ctx context.Context, a *store.Agent) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agents (id, key, display_name, provider, model, system_extra)
		VALUES (?, ?, ?, ?, ?, ?)
	`, a.ID, a.Key, a.DisplayName, a.Provider, a.Model, a.SystemExtra)
	return err
}

func (s *Store) ListAgents(ctx context.Context) ([]store.Agent, error) {
	var out []store.Agent
	err := s.db.SelectContext(ctx, &out, `SELECT * FROM agents ORDER BY created_at`)
	return out, err
}

func (s *Store) GetAgentByKey(ctx context.Context, key string) (*store.Agent, error) {
	var a store.Agent
	if err := s.db.GetContext(ctx, &a, `SELECT * FROM agents WHERE key = ?`, key); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) UpdateAgent(ctx context.Context, id string, fields map[string]any) error {
	allowed := map[string]bool{
		"display_name": true, "provider": true, "model": true, "system_extra": true,
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
	q := fmt.Sprintf("UPDATE agents SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, q, args...)
	return err
}

// --- Sessions + messages ---

func (s *Store) CreateSession(ctx context.Context, sess *store.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, key, agent_key, title) VALUES (?, ?, ?, ?)
	`, sess.ID, sess.Key, sess.AgentKey, sess.Title)
	return err
}

func (s *Store) GetSessionByKey(ctx context.Context, key string) (*store.Session, error) {
	var sess store.Session
	if err := s.db.GetContext(ctx, &sess, `SELECT * FROM sessions WHERE key = ?`, key); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) ListSessions(ctx context.Context) ([]store.Session, error) {
	var out []store.Session
	err := s.db.SelectContext(ctx, &out, `SELECT * FROM sessions ORDER BY updated_at DESC`)
	return out, err
}

func (s *Store) UpdateSessionTitle(ctx context.Context, key, title string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET title = ?, updated_at = datetime('now') WHERE key = ?`, title, key)
	return err
}

// AppendMessage inserts a message with the next seq for its session, atomically.
func (s *Store) AppendMessage(ctx context.Context, sessionKey string, msg store.Message) (store.Message, error) {
	var sessID string
	if err := s.db.GetContext(ctx, &sessID, `SELECT id FROM sessions WHERE key = ?`, sessionKey); err != nil {
		return msg, err
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return msg, err
	}
	defer tx.Rollback() //nolint:errcheck

	var maxSeq sql.NullInt64
	if err := tx.GetContext(ctx, &maxSeq, `SELECT MAX(seq) FROM messages WHERE session_id = ?`, sessID); err != nil {
		return msg, err
	}
	msg.SessionID = sessID
	msg.Seq = int(maxSeq.Int64) + 1

	_, err = tx.ExecContext(ctx, `
		INSERT INTO messages (id, session_id, seq, role, content, thinking, tool_calls_json, tool_call_id, tool_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.SessionID, msg.Seq, msg.Role, msg.Content, msg.Thinking, msg.ToolCallsJSON, msg.ToolCallID, msg.ToolName)
	if err != nil {
		return msg, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sessions SET updated_at = datetime('now') WHERE id = ?`, sessID); err != nil {
		return msg, err
	}
	if err := tx.Commit(); err != nil {
		return msg, err
	}
	return msg, nil
}

func (s *Store) ListMessages(ctx context.Context, sessionKey string) ([]store.Message, error) {
	var out []store.Message
	err := s.db.SelectContext(ctx, &out, `
		SELECT m.* FROM messages m
		JOIN sessions s ON s.id = m.session_id
		WHERE s.key = ? ORDER BY m.seq
	`, sessionKey)
	return out, err
}

// --- Policy ---

func (s *Store) ToolEnabled(ctx context.Context, name string) bool {
	var enabled int
	err := s.db.GetContext(ctx, &enabled, `SELECT enabled FROM tool_policy WHERE tool_name = ?`, name)
	if err == sql.ErrNoRows {
		return true // absent => enabled
	}
	if err != nil {
		return true
	}
	return enabled == 1
}

func (s *Store) SetToolEnabled(ctx context.Context, name string, enabled bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tool_policy (tool_name, enabled) VALUES (?, ?)
		ON CONFLICT(tool_name) DO UPDATE SET enabled = EXCLUDED.enabled
	`, name, boolToInt(enabled))
	return err
}

func (s *Store) ListToolPolicy(ctx context.Context) ([]store.ToolPolicy, error) {
	var rows []struct {
		ToolName string `db:"tool_name"`
		Enabled  int    `db:"enabled"`
	}
	if err := s.db.SelectContext(ctx, &rows, `SELECT tool_name, enabled FROM tool_policy`); err != nil {
		return nil, err
	}
	out := make([]store.ToolPolicy, 0, len(rows))
	for _, r := range rows {
		out = append(out, store.ToolPolicy{ToolName: r.ToolName, Enabled: r.Enabled == 1})
	}
	return out, nil
}

func (s *Store) DisabledSkills(ctx context.Context) ([]string, error) {
	var out []string
	err := s.db.SelectContext(ctx, &out, `SELECT slug FROM skill_disabled`)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return out, err
}

func (s *Store) SetSkillEnabled(ctx context.Context, slug string, enabled bool) error {
	if enabled {
		_, err := s.db.ExecContext(ctx, `DELETE FROM skill_disabled WHERE slug = ?`, slug)
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO skill_disabled (slug) VALUES (?)`, slug)
	return err
}

// --- helpers ---

type providerRow struct {
	ID          string `db:"id"`
	Name        string `db:"name"`
	DisplayName string `db:"display_name"`
	APIBase     string `db:"api_base"`
	APIKeyEnc   []byte `db:"api_key_enc"`
	Enabled     int    `db:"enabled"`
	CreatedAt   string `db:"created_at"`
	UpdatedAt   string `db:"updated_at"`
}

func (r providerRow) toProvider() store.Provider {
	return store.Provider{
		ID:          r.ID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		APIBase:     r.APIBase,
		APIKey:      "", // never expose
		Enabled:     r.Enabled == 1,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Provider is re-exported as store.Provider for convenience.
type Provider = store.Provider
