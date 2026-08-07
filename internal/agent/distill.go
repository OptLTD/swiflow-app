package agent

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/OptLTD/swiflow/internal/observe"
	"github.com/OptLTD/swiflow/internal/store"
)

const maxCharterBytes = 4096

// looksLikeCorrection detects follow-up preference / correction phrasing.
func looksLikeCorrection(msg string) bool {
	s := strings.ToLower(strings.TrimSpace(msg))
	if s == "" {
		return false
	}
	markers := []string{
		"以后", "下次", "不要再", "从现在起", "以后都", "记得",
		"always ", "always use", "don't ", "do not ", "instead ",
		"from now on", "next time", "prefer ", "never ",
	}
	for _, m := range markers {
		if strings.Contains(s, m) {
			return true
		}
	}
	return false
}

// appendCharterPrinciple appends a short user correction into the agent charter.
func appendCharterPrinciple(ctx context.Context, st store.Store, sessionID string, ag *store.Agent, principle string, source string) {
	if st == nil || ag == nil {
		return
	}
	principle = strings.TrimSpace(principle)
	if principle == "" {
		return
	}
	// Keep appends short.
	if utf8.RuneCountInString(principle) > 240 {
		runes := []rune(principle)
		principle = string(runes[:240]) + "…"
	}
	line := "- " + principle
	cur := strings.TrimSpace(ag.Charter)
	if cur == "" {
		cur = strings.TrimSpace(defaultCharterSeed)
	}
	if strings.Contains(cur, principle) || strings.Contains(cur, line) {
		return
	}
	next := cur + "\n" + line
	if len(next) > maxCharterBytes {
		// Drop oldest lines until under budget (keep seed-ish tail).
		lines := strings.Split(next, "\n")
		for len(lines) > 1 && len(strings.Join(lines, "\n")) > maxCharterBytes {
			lines = lines[1:]
		}
		next = strings.Join(lines, "\n")
	}
	if err := st.UpdateAgent(ctx, ag.ID, map[string]any{"charter": next}); err != nil {
		return
	}
	ag.Charter = next
	observe.CharterUpdated(sessionID, ag.Key, source)
}
