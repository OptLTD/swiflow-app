package agent

import (
	"context"
	"encoding/json"

	"github.com/OptLTD/swiflow/internal/llmclient"
	"github.com/OptLTD/swiflow/internal/store"
)

const maxReflectPerRun = 2

const reflectNudge = "Before finishing, self-check against the user's goal for this turn: (1) Does your reply fully deliver what they asked? (2) What evidence do you have? (3) What gaps remain? If gaps are fixable with tools, continue working now — do not ask whether you may submit. Only call clarify if a critical fact is missing and no reasonable default exists. Otherwise give the final answer."

const defaultCharterSeed = `- Prefer delivering the user's stated goal over exhaustive exploration.
- Keep the workspace tidy: put new files under a topic slug folder, never dump into the workspace root.
- When stuck, change approach; do not repeat the same failing tool pattern.
- For batch file/table work, spawn subagents early rather than serial extract.
- Before claiming done, verify the deliverable matches the request; fix gaps yourself.
- Ask the user (clarify) only when a critical fact is missing and no reasonable default exists.`

var reflectToolAllow = map[string]bool{
	"todo_read":        true,
	"todo_write":       true,
	"fs_read":          true,
	"experience_write": true,
	"clarify":          true,
}

func reflectToolDefs(all []llmclient.ToolDef, deny map[string]bool) []llmclient.ToolDef {
	return filterTools(all, RunOpts{AllowTools: reflectToolAllow, DenyTools: deny})
}

func sessionHasOpenTodos(ctx context.Context, st store.Store, sessionID string) bool {
	if st == nil {
		return false
	}
	raw, err := st.LoadTodos(ctx, sessionID)
	if err != nil || raw == "" {
		return false
	}
	return parseOpenTodos(raw)
}

func parseOpenTodos(itemsJSON string) bool {
	if itemsJSON == "" || itemsJSON == "[]" {
		return false
	}
	var items []struct {
		Done bool `json:"done"`
	}
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return false
	}
	for _, it := range items {
		if !it.Done {
			return true
		}
	}
	return false
}

func isSignificantRun(toolsUsed bool, openTodos bool) bool {
	return toolsUsed || openTodos
}
