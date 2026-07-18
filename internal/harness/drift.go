package harness

import (
	"strings"
	"time"
)

const (
	noProgressAfter     = 45 * time.Second
	todoStaleMinRounds  = 2
	goalMismatchMinTool = 4 // after this many tool calls still only list/experience
)

// evalDrift returns newly fired signals for the current run state.
// It is deterministic and does not call an LLM.
func evalDrift(s *runState, now time.Time) []DriftSignal {
	if s == nil {
		return nil
	}
	var out []DriftSignal

	if s.repeatToolStreak >= 3 {
		out = append(out, DriftSignal{
			Code: DriftStallRepeatTools, Severity: "error",
			Message: "same tool calls repeated ≥3 times", At: now,
		})
	}
	if s.consecErrors >= 3 {
		out = append(out, DriftSignal{
			Code: DriftStallToolErrors, Severity: "error",
			Message: "consecutive tool errors ≥3", At: now,
		})
	}

	if s.snap.ParentID != "" && s.snap.MaxRounds > 0 && s.snap.Round >= s.snap.MaxRounds/2 {
		if !s.hadProgressSinceHalf {
			out = append(out, DriftSignal{
				Code: DriftBudgetPressure, Severity: "warn",
				Message: "child past half budget with no todo/artifact progress", At: now,
			})
		}
	}

	if s.roundsWithTools >= todoStaleMinRounds && s.todosFingerprint != "" &&
		s.todosFingerprint == s.todosFingerprintAtRoundStart && s.snap.Status == StatusRunning {
		out = append(out, DriftSignal{
			Code: DriftTodoStale, Severity: "warn",
			Message: "tools ran for ≥2 rounds but checklist did not change", At: now,
		})
	}

	if (s.snap.Status == StatusDone || s.snap.Status == StatusBudget || s.snap.Status == StatusStall) &&
		hasOpenTodos(s.snap.Todos) {
		out = append(out, DriftSignal{
			Code: DriftDoneOpenTodos, Severity: "warn",
			Message: "run finished while checklist still has open items", At: now,
		})
	}

	if s.snap.Status == StatusRunning && !s.lastProgress.IsZero() &&
		now.Sub(s.lastProgress) >= noProgressAfter {
		out = append(out, DriftSignal{
			Code: DriftNoProgress, Severity: "warn",
			Message: "no tool/delta progress for ≥45s", At: now,
		})
	}

	if looksLikeBatchGoal(s.snap.Goal) && s.snap.Metrics.ToolCalls >= goalMismatchMinTool &&
		s.onlyExploreTools {
		out = append(out, DriftSignal{
			Code: DriftGoalToolMismatch, Severity: "warn",
			Message: "goal looks like batch extract/summary but tools are only list/experience", At: now,
		})
	}

	return out
}

func hasOpenTodos(items []TodoItem) bool {
	for _, t := range items {
		if !t.Done {
			return true
		}
	}
	return false
}

func looksLikeBatchGoal(goal string) bool {
	g := strings.ToLower(goal)
	if strings.Contains(goal, "@/") {
		return true
	}
	keys := []string{"汇总", "表格", "excel", "csv", "xlsx", "ocr", "提取", "content_extract", "batch", "多张", "多份"}
	for _, k := range keys {
		if strings.Contains(g, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

func isExploreOnlyTool(name string) bool {
	switch name {
	case "fs_list", "experience_list", "experience_search", "todo_read", "skill_search":
		return true
	default:
		return false
	}
}

func isProgressTool(name string) bool {
	switch name {
	case "content_extract", "fs_write", "fs_edit", "exec", "python_run", "node_run":
		return true
	default:
		return false
	}
}

func todosFingerprint(items []TodoItem) string {
	var b strings.Builder
	for _, t := range items {
		b.WriteString(t.ID)
		b.WriteByte('|')
		b.WriteString(t.Text)
		if t.Done {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
		b.WriteByte(';')
	}
	return b.String()
}
