package sqlstore

import (
	"context"
	"encoding/json"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/library/support"
)

func (s *Store) CreateExperience(ctx context.Context, e *store.Experience) error {
	if e.ID == "" {
		e.ID = support.NewID()
	}
	tags, _ := json.Marshal(e.Tags)
	if len(tags) == 0 {
		tags = []byte("[]")
	}
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO agent_experience (id, sid, agent, summary, outcome, tags)
		VALUES (?, ?, ?, ?, ?, ?)
	`), e.ID, e.Sid, e.Agent, e.Summary, e.Outcome, string(tags))
	return err
}

func (s *Store) ListExperiences(ctx context.Context, agentKey string, limit int) ([]store.Experience, error) {
	if limit <= 0 {
		limit = 10
	}
	var rows []struct {
		ID        string `db:"id"`
		Sid       string `db:"sid"`
		Agent     string `db:"agent"`
		Summary   string `db:"summary"`
		Outcome   string `db:"outcome"`
		Tags      dbJSON `db:"tags"`
		CreatedAt dbTime `db:"created_at"`
	}
	if err := s.db.SelectContext(ctx, &rows, s.sql(`
		SELECT id, sid, agent, summary, outcome, tags, created_at
		FROM agent_experience
		WHERE agent = ?
		ORDER BY created_at DESC
		LIMIT ?
	`), agentKey, limit); err != nil {
		return nil, err
	}
	out := make([]store.Experience, 0, len(rows))
	for _, r := range rows {
		var tags []string
		_ = json.Unmarshal([]byte(r.Tags.String()), &tags)
		if tags == nil {
			tags = []string{}
		}
		out = append(out, store.Experience{
			ID:        r.ID,
			Sid:       r.Sid,
			Agent:     r.Agent,
			Summary:   r.Summary,
			Outcome:   r.Outcome,
			Tags:      tags,
			CreatedAt: r.CreatedAt.String(),
		})
	}
	return out, nil
}

func (s *Store) DeleteExperience(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.sql(`DELETE FROM agent_experience WHERE id = ?`), id)
	return err
}

func (s *Store) SaveTodos(ctx context.Context, sessionID string, itemsJSON string) error {
	if itemsJSON == "" {
		itemsJSON = "[]"
	}
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO agent_todo (sid, items, updated_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(sid) DO UPDATE SET
			items = excluded.items,
			updated_at = excluded.updated_at
	`), sessionID, itemsJSON)
	return err
}

func (s *Store) LoadTodos(ctx context.Context, sessionID string) (string, error) {
	var items dbJSON
	err := s.db.GetContext(ctx, &items, s.sql(`
		SELECT items FROM agent_todo WHERE sid = ?
	`), sessionID)
	if err != nil {
		return "[]", nil
	}
	out := items.String()
	if out == "" {
		return "[]", nil
	}
	return out, nil
}
