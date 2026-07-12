package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"mira/internal/store"
)

type cronJobRow struct {
	ID        string `db:"id"`
	Name      string `db:"name"`
	AgentKey  string `db:"agent_key"`
	Message   string `db:"message"`
	Schedule  string `db:"schedule"`
	Enabled   int    `db:"enabled"`
	LastRunAt sql.NullString `db:"last_run_at"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}

func (r cronJobRow) toCronJob() store.CronJob {
	j := store.CronJob{
		ID: r.ID, Name: r.Name, AgentKey: r.AgentKey, Message: r.Message,
		Schedule: r.Schedule, Enabled: r.Enabled == 1,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.LastRunAt.Valid {
		j.LastRunAt = r.LastRunAt.String
	}
	return j
}

func (s *Store) CreateCronJob(ctx context.Context, j *store.CronJob) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cron_jobs (id, name, agent_key, message, schedule, enabled)
		VALUES (?, ?, ?, ?, ?, ?)
	`, j.ID, j.Name, j.AgentKey, j.Message, j.Schedule, boolToInt(j.Enabled))
	return err
}

func (s *Store) ListCronJobs(ctx context.Context) ([]store.CronJob, error) {
	var rows []cronJobRow
	if err := s.db.SelectContext(ctx, &rows, `SELECT * FROM cron_jobs ORDER BY created_at`); err != nil {
		return nil, err
	}
	out := make([]store.CronJob, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toCronJob())
	}
	return out, nil
}

func (s *Store) GetCronJobByID(ctx context.Context, id string) (*store.CronJob, error) {
	var r cronJobRow
	if err := s.db.GetContext(ctx, &r, `SELECT * FROM cron_jobs WHERE id = ?`, id); err != nil {
		return nil, err
	}
	j := r.toCronJob()
	return &j, nil
}

func (s *Store) UpdateCronJob(ctx context.Context, id string, fields map[string]any) error {
	allowed := map[string]bool{
		"agent_key": true, "message": true, "schedule": true, "enabled": true,
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
		default:
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
	q := fmt.Sprintf("UPDATE cron_jobs SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, q, args...)
	return err
}

func (s *Store) DeleteCronJob(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM cron_jobs WHERE id = ?`, id)
	return err
}

func (s *Store) SetCronJobLastRun(ctx context.Context, id string, at string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE cron_jobs SET last_run_at = ?, updated_at = datetime('now') WHERE id = ?
	`, at, id)
	return err
}
