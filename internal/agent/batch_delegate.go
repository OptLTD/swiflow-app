package agent

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/OptLTD/swiflow/internal/llmclient"
	"github.com/OptLTD/swiflow/internal/tool"
)

var (
	atPathLine = regexp.MustCompile(`(?m)^@/[^\s]+`)
	atPathAny  = regexp.MustCompile(`@/[^\s]+`)
)

// batchDelegateThreshold: attaching this many @/ files forces a full handoff to
// a child (deterministic routing — no runtime cost probing / EWMA).
const batchDelegateThreshold = 3

func listAttachedAtPaths(msg string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(m string) {
		if seen[m] {
			return
		}
		seen[m] = true
		out = append(out, m)
	}
	start := strings.Index(msg, "[UPLOAD FILES START]")
	end := strings.Index(msg, "[UPLOAD FILES END]")
	if start >= 0 && end > start {
		for _, m := range atPathLine.FindAllString(msg[start:end], -1) {
			add(m)
		}
		return out
	}
	for _, m := range atPathAny.FindAllString(msg, -1) {
		add(m)
	}
	return out
}

func countAttachedAtPaths(msg string) int {
	return len(listAttachedAtPaths(msg))
}

func pathFromToolArgs(args map[string]any) string {
	if args == nil {
		return ""
	}
	p, _ := args["path"].(string)
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "@/") {
		return p
	}
	return "@/" + strings.TrimPrefix(p, "/")
}

// extractAtPaths returns @/ paths mentioned anywhere in text (used to report a
// child's artifacts back to the parent).
func extractAtPaths(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, m := range atPathAny.FindAllString(text, -1) {
		m = strings.TrimRight(m, ".,;:)]}\"'）】")
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		out = append(out, m)
	}
	return out
}

// artifactPath returns the @/ file a tool call writes, if it is a file writer.
func artifactPath(tc llmclient.ToolCall) string {
	if !strings.Contains(tc.Name, "write") {
		return ""
	}
	return pathFromToolArgs(tc.Arguments)
}

func appendUnique(list []string, v string) []string {
	if v == "" {
		return list
	}
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

// shouldForceBatchDelegate is a pure function of the request: when the user
// attaches >= threshold files, the main agent must delegate the whole batch.
func shouldForceBatchDelegate(userMessage string, childRun bool) (paths []string, forced bool) {
	if childRun {
		return nil, false
	}
	paths = listAttachedAtPaths(userMessage)
	if len(paths) < batchDelegateThreshold {
		return nil, false
	}
	return paths, true
}

// batchDelegateNudge instructs the main agent to hand the batch to one child.
func batchDelegateNudge(paths []string) string {
	var b strings.Builder
	b.WriteString("[Routing] User attached ")
	b.WriteString(strconv.Itoa(len(paths)))
	b.WriteString(" @/ files for a batch (likely a table/Excel). content_extract is DISABLED on the MAIN agent. ")
	b.WriteString("Do NOT ask which columns — use sensible defaults (单据编号、车牌号、装货量、卸货量、日期…). ")
	b.WriteString("Call delegate_task ONCE now (synchronous full handoff). Put EVERY path below inside the goal text (not a separate path arg), ")
	b.WriteString("ask the child to write an xlsx under workspace, use max_rounds ≥ 16, and let the child choose tools. Do not one-file-per-delegate:\n")
	for _, p := range paths {
		b.WriteString(p)
		b.WriteByte('\n')
	}
	return b.String()
}

// isChildRun reports whether opts belong to a delegate_task child.
func isChildRun(opts RunOpts) bool {
	return opts.DenyTools["delegate_task"]
}

func denyContentExtract(opts *RunOpts) {
	if opts.DenyTools == nil {
		opts.DenyTools = map[string]bool{}
	}
	opts.DenyTools[tool.ToolContentExtract] = true
}
