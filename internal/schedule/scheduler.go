// Package schedule runs scheduled agent tasks. Phase 2.
package schedule

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/agentine/cadence"

	"github.com/OptLTD/swiflow/internal/agent"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/tenant"
	"github.com/OptLTD/swiflow/library/support"
)

// EventPublisher broadcasts agent events to session watchers (optional).
type EventPublisher interface {
	Publish(sessionID string, ev agent.Event)
}

// Scheduler executes cron jobs using the agent runner.
type Scheduler struct {
	st     store.Store
	runner *agent.Runner
	hub    EventPublisher
	cron   *cadence.Cron
	mu     sync.Mutex
	ids    map[string]cadence.EntryID // job ID -> cron entry
}

// New creates a scheduler. Call Start after construction. hub may be nil.
func New(st store.Store, runner *agent.Runner, hub EventPublisher) *Scheduler {
	parser := cadence.NewParser(cadence.Minute | cadence.Hour | cadence.Dom | cadence.Month | cadence.Dow | cadence.Descriptor)
	return &Scheduler{
		st:     st,
		runner: runner,
		hub:    hub,
		cron:   cadence.New(cadence.WithParser(parser)),
		ids:    map[string]cadence.EntryID{},
	}
}

// Start loads jobs from the store and begins the scheduler.
func (s *Scheduler) Start(ctx context.Context) error {
	if err := s.Reload(ctx); err != nil {
		return err
	}
	s.cron.Start()
	return nil
}

// Stop halts the scheduler.
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
}

// Reload refreshes scheduled jobs from the database (all tenants).
func (s *Scheduler) Reload(ctx context.Context) error {
	jobs, err := s.st.ListAllCronJobs(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for jobID, entryID := range s.ids {
		s.cron.Remove(entryID)
		delete(s.ids, jobID)
	}
	for _, job := range jobs {
		if !job.Enabled {
			continue
		}
		j := job
		entryID, err := s.cron.AddFunc(j.Schedule, func() { s.runJob(j) })
		if err != nil {
			slog.Error("cron invalid schedule", "job", j.Name, "schedule", j.Schedule, "error", err)
			continue
		}
		s.ids[j.ID] = entryID
		slog.Info("cron job scheduled", "job", j.Name, "tid", j.Tid, "schedule", j.Schedule)
	}
	return nil
}

func (s *Scheduler) runJob(job store.CronJob) {
	ctx := tenant.WithID(context.Background(), job.Tid)
	fresh, err := s.st.GetCronJobByID(ctx, job.ID)
	if err != nil || !fresh.Enabled {
		return
	}
	sessionID := support.NewID()
	slog.Info("cron job running", "job", fresh.Name, "tid", fresh.Tid, "session", sessionID)
	// Record the trigger time up front so last_run_at reflects when the job
	// fired, independent of how long the run (incl. retries/backoff) takes.
	_ = s.st.SetCronJobLastRun(ctx, fresh.ID, time.Now().UTC().Format(time.RFC3339))
	err = s.runner.Run(ctx, sessionID, fresh.Agent, fresh.Message, func(ev agent.Event) {
		if ev.Type == "error" {
			slog.Error("cron job error", "job", fresh.Name, "error", ev.Error)
		}
	})
	if err != nil {
		slog.Error("cron job failed", "job", fresh.Name, "error", err)
	}
}

// ScheduleRun starts a one-shot delayed agent run in sessionID after the given delay.
// The message is injected as a new user turn and the agent is invoked (not a static reply).
// tid scopes the run's tenant context (empty = default).
func (s *Scheduler) ScheduleRun(sessionID, agentKey, message string, after time.Duration, tid string) {
	if sessionID == "" || agentKey == "" || message == "" || after < 0 || s.runner == nil {
		return
	}
	time.AfterFunc(after, func() {
		ctx := tenant.WithID(context.Background(), tid)
		slog.Info("scheduled task running", "session", sessionID, "agent", agentKey, "tid", tid, "after", after)
		if s.hub != nil {
			s.hub.Publish(sessionID, agent.Event{Type: "user", Content: message})
		}
		err := s.runner.Run(ctx, sessionID, agentKey, message, func(ev agent.Event) {
			if s.hub != nil {
				s.hub.Publish(sessionID, ev)
			}
			if ev.Type == "error" {
				slog.Error("scheduled task error", "session", sessionID, "error", ev.Error)
			}
		})
		if err != nil {
			slog.Error("scheduled task failed", "session", sessionID, "error", err)
		}
	})
}

// AddJob persists a cron job and reloads the scheduler.
func (s *Scheduler) AddJob(ctx context.Context, job *store.CronJob) error {
	if err := s.st.CreateCronJob(ctx, job); err != nil {
		return err
	}
	return s.Reload(ctx)
}
