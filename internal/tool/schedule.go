// Scheduling tools: one-shot delayed runs and cron jobs. Phase 2.
package tool

import (
	"context"
	"fmt"
	"time"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/library/support"
)

type jobScheduler interface {
	ScheduleRun(sessionID, agentKey, message string, after time.Duration)
	AddJob(ctx context.Context, job *store.CronJob) error
}

type agentLookup interface {
	GetAgentByKey(ctx context.Context, key string) (*store.Agent, error)
}

type scheduleTools struct {
	st    agentLookup
	sched jobScheduler
}

type scheduleRunTool struct{ base *scheduleTools }

func (t *scheduleRunTool) Name() string { return "schedule_run" }
func (t *scheduleRunTool) Description() string {
	return "Schedule a one-shot task that re-invokes the agent in the current session after a delay. The message becomes a new user turn."
}
func (t *scheduleRunTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"delay_seconds": map[string]any{"type": "integer", "description": "Seconds to wait before running (1–86400)."},
			"message":       map[string]any{"type": "string", "description": "User message sent to the agent when the task fires."},
		},
		"required": []string{"delay_seconds", "message"},
	}
}
func (t *scheduleRunTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	rc, ok := RunContextFrom(ctx)
	if !ok || rc.SessionID == "" {
		return "", fmt.Errorf("no active session for scheduled task")
	}
	message, _ := args["message"].(string)
	if message == "" {
		return "", fmt.Errorf("message is required")
	}
	agentKey := "default"
	if rc.Agent != "" {
		agentKey = rc.Agent
	}
	if _, err := t.base.st.GetAgentByKey(ctx, agentKey); err != nil {
		return "", fmt.Errorf("unknown agent: %s", agentKey)
	}
	delay := clampDelay(args)
	after := time.Duration(delay) * time.Second
	t.base.sched.ScheduleRun(rc.SessionID, agentKey, message, after)
	return fmt.Sprintf("scheduled agent run in %ds for session %s", delay, rc.SessionID), nil
}

type scheduleCreateTool struct{ base *scheduleTools }

func (t *scheduleCreateTool) Name() string { return "schedule_create" }
func (t *scheduleCreateTool) Description() string {
	return "Create a recurring cron job that runs the agent with a message. For one-shot delayed tasks in the current chat use schedule_run."
}
func (t *scheduleCreateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":     map[string]any{"type": "string"},
			"message":  map[string]any{"type": "string", "description": "User message sent to the agent when the job runs."},
			"schedule": map[string]any{"type": "string", "description": "Cron expression, e.g. @hourly, 0 9 * * *, @every 5m."},
		},
		"required": []string{"name", "message", "schedule"},
	}
}
func (t *scheduleCreateTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	name, _ := args["name"].(string)
	message, _ := args["message"].(string)
	schedExpr, _ := args["schedule"].(string)
	if name == "" || message == "" || schedExpr == "" {
		return "", fmt.Errorf("name, message, and schedule are required")
	}
	agentKey := "default"
	if rc, ok := RunContextFrom(ctx); ok && rc.Agent != "" {
		agentKey = rc.Agent
	}
	if _, err := t.base.st.GetAgentByKey(ctx, agentKey); err != nil {
		return "", fmt.Errorf("unknown agent: %s", agentKey)
	}
	job := &store.CronJob{
		ID: support.NewID(), Name: name, Agent: agentKey,
		Message: message, Schedule: schedExpr, Enabled: true,
	}
	if err := t.base.sched.AddJob(ctx, job); err != nil {
		return "", err
	}
	return fmt.Sprintf("cron job %q created (id %s, schedule %s)", name, job.ID, schedExpr), nil
}

func clampDelay(args map[string]any) int {
	delay := 30
	if d, ok := args["delay_seconds"].(float64); ok && d > 0 {
		delay = int(d)
	}
	if delay < 1 {
		delay = 1
	}
	if delay > 86400 {
		delay = 86400
	}
	return delay
}

// RegisterSchedule registers delayed-run and cron scheduling tools.
func RegisterSchedule(r *Registry, st agentLookup, sched jobScheduler) {
	base := &scheduleTools{st: st, sched: sched}
	r.Register(&scheduleRunTool{base: base})
	r.Register(&scheduleCreateTool{base: base})
}
