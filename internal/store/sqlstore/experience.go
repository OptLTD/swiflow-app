package sqlstore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/library/support"
)

const maxExperienceWeight = 100

func (s *Store) CreateExperience(ctx context.Context, e *store.Experience) error {
	if e.ID == "" {
		e.ID = support.NewID()
	}
	if e.Weight <= 0 {
		e.Weight = 1
	}
	tags, _ := json.Marshal(e.Tags)
	if len(tags) == 0 {
		tags = []byte("[]")
	}
	t := tid(ctx)
	e.Tid = t
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO agent_experience (id, tid, sid, agent, summary, outcome, tags, weight)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`), e.ID, t, e.Sid, e.Agent, e.Summary, e.Outcome, string(tags), e.Weight)
	return err
}

func (s *Store) ListExperiences(ctx context.Context, agentKey string, limit int) ([]store.Experience, error) {
	if limit <= 0 {
		limit = 10
	}
	var rows []struct {
		ID        string `db:"id"`
		Tid       string `db:"tid"`
		Sid       string `db:"sid"`
		Agent     string `db:"agent"`
		Summary   string `db:"summary"`
		Outcome   string `db:"outcome"`
		Tags      dbJSON `db:"tags"`
		Weight    int    `db:"weight"`
		CreatedAt dbTime `db:"created_at"`
	}
	if err := s.db.SelectContext(ctx, &rows, s.sql(`
		SELECT id, tid, sid, agent, summary, outcome, tags, weight, created_at
		FROM agent_experience
		WHERE agent = ? AND tid = ?
		ORDER BY weight DESC, created_at DESC
		LIMIT ?
	`), agentKey, tid(ctx), limit); err != nil {
		return nil, err
	}
	out := make([]store.Experience, 0, len(rows))
	for _, r := range rows {
		var tags []string
		_ = json.Unmarshal([]byte(r.Tags.String()), &tags)
		if tags == nil {
			tags = []string{}
		}
		w := r.Weight
		if w <= 0 {
			w = 1
		}
		out = append(out, store.Experience{
			ID:        r.ID,
			Tid:       r.Tid,
			Sid:       r.Sid,
			Agent:     r.Agent,
			Summary:   r.Summary,
			Outcome:   r.Outcome,
			Tags:      tags,
			Weight:    w,
			CreatedAt: r.CreatedAt.String(),
		})
	}
	return out, nil
}

func (s *Store) BumpExperienceWeight(ctx context.Context, id string, delta int) (*store.Experience, error) {
	if id == "" {
		return nil, fmt.Errorf("id required")
	}
	if delta <= 0 {
		delta = 1
	}
	t := tid(ctx)
	res, err := s.db.ExecContext(ctx, s.sql(`
		UPDATE agent_experience
		SET weight = CASE
			WHEN weight + ? > ? THEN ?
			WHEN weight + ? < 1 THEN 1
			ELSE weight + ?
		END
		WHERE id = ? AND tid = ?
	`), delta, maxExperienceWeight, maxExperienceWeight, delta, delta, id, t)
	if err != nil {
		return nil, err
	}
	if err := s.affectedOrNoRows(res, nil); err != nil {
		return nil, err
	}
	var row struct {
		ID        string `db:"id"`
		Tid       string `db:"tid"`
		Sid       string `db:"sid"`
		Agent     string `db:"agent"`
		Summary   string `db:"summary"`
		Outcome   string `db:"outcome"`
		Tags      dbJSON `db:"tags"`
		Weight    int    `db:"weight"`
		CreatedAt dbTime `db:"created_at"`
	}
	if err := s.db.GetContext(ctx, &row, s.sql(`
		SELECT id, tid, sid, agent, summary, outcome, tags, weight, created_at
		FROM agent_experience WHERE id = ? AND tid = ?
	`), id, t); err != nil {
		return nil, err
	}
	var tags []string
	_ = json.Unmarshal([]byte(row.Tags.String()), &tags)
	if tags == nil {
		tags = []string{}
	}
	return &store.Experience{
		ID: row.ID, Tid: row.Tid, Sid: row.Sid, Agent: row.Agent,
		Summary: row.Summary, Outcome: row.Outcome, Tags: tags,
		Weight: row.Weight, CreatedAt: row.CreatedAt.String(),
	}, nil
}

func (s *Store) DeleteExperience(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, s.sql(`
		DELETE FROM agent_experience WHERE id = ? AND tid = ?
	`), id, tid(ctx))
	return s.affectedOrNoRows(res, err)
}

func (s *Store) SaveTodos(ctx context.Context, sessionID string, itemsJSON string) error {
	if itemsJSON == "" {
		itemsJSON = "[]"
	}
	t := tid(ctx)
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO agent_todo (sid, tid, items, updated_at)
		VALUES (?, ?, ?, datetime('now'))
		ON CONFLICT(sid) DO UPDATE SET
			items = excluded.items,
			tid = excluded.tid,
			updated_at = excluded.updated_at
	`), sessionID, t, itemsJSON)
	return err
}

func (s *Store) LoadTodos(ctx context.Context, sessionID string) (string, error) {
	var items dbJSON
	err := s.db.GetContext(ctx, &items, s.sql(`
		SELECT items FROM agent_todo WHERE sid = ? AND tid = ?
	`), sessionID, tid(ctx))
	if err != nil {
		return "[]", nil
	}
	out := items.String()
	if out == "" {
		return "[]", nil
	}
	return out, nil
}
