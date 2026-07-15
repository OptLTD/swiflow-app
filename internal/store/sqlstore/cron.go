package sqlstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/internal/store"
)

type cronJobRow struct {
	ID        string     `db:"id"`
	Tid       string     `db:"tid"`
	Name      string     `db:"name"`
	Agent     string     `db:"agent"`
	Message   string     `db:"message"`
	Schedule  string     `db:"schedule"`
	Enabled   dbBool     `db:"enabled"`
	LastRunAt dbNullTime `db:"last_run_at"`
	CreatedAt dbTime     `db:"created_at"`
	UpdatedAt dbTime     `db:"updated_at"`
}

func (r cronJobRow) toCronJob() store.CronJob {
	j := store.CronJob{
		ID: r.ID, Name: r.Name, Agent: r.Agent, Message: r.Message,
		Schedule: r.Schedule, Enabled: r.Enabled.b,
		CreatedAt: r.CreatedAt.String(), UpdatedAt: r.UpdatedAt.String(),
	}
	if r.LastRunAt.valid {
		j.LastRunAt = r.LastRunAt.s
	}
	return j
}

func (s *Store) CreateCronJob(ctx context.Context, j *store.CronJob) error {
	_, err := s.db.ExecContext(ctx, s.sql(`
		INSERT INTO agent_sched (id, name, agent, message, schedule, enabled)
		VALUES (?, ?, ?, ?, ?, ?)
	`), j.ID, j.Name, j.Agent, j.Message, j.Schedule, s.boolArg(j.Enabled))
	return err
}

func (s *Store) ListCronJobs(ctx context.Context) ([]store.CronJob, error) {
	var rows []cronJobRow
	if err := s.db.SelectContext(ctx, &rows, s.sql(`SELECT * FROM agent_sched ORDER BY created_at`)); err != nil {
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
	if err := s.db.GetContext(ctx, &r, s.sql(`SELECT * FROM agent_sched WHERE id = ?`), id); err != nil {
		return nil, err
	}
	j := r.toCronJob()
	return &j, nil
}

func (s *Store) UpdateCronJob(ctx context.Context, id string, fields map[string]any) error {
	allowed := map[string]bool{
		"agent": true, "message": true, "schedule": true, "enabled": true,
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
	sets = append(sets, "updated_at = "+nowToken)
	args = append(args, id)
	q := fmt.Sprintf("UPDATE agent_sched SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, s.sql(q), args...)
	return err
}

func (s *Store) DeleteCronJob(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.sql(`DELETE FROM agent_sched WHERE id = ?`), id)
	return err
}

func (s *Store) SetCronJobLastRun(ctx context.Context, id string, at string) error {
	var arg any = at
	if s.driver == DialectPostgres {
		ts, err := time.Parse(time.RFC3339, at)
		if err != nil {
			ts, err = time.Parse(time.RFC3339Nano, at)
			if err != nil {
				return fmt.Errorf("last_run_at: %w", err)
			}
		}
		arg = ts.UTC()
	}
	_, err := s.db.ExecContext(ctx, s.sql(`
		UPDATE agent_sched SET last_run_at = ?, updated_at = datetime('now') WHERE id = ?
	`), arg, id)
	return err
}
