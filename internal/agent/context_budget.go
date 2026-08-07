package agent

import (
	"encoding/json"

	"github.com/OptLTD/swiflow/internal/llmclient"
)

const (
	defaultMaxContextChars = 120_000
	defaultKeepTailMsgs    = 16
	softBudgetRatio        = 0.85
	toolCompactSoft        = 200
	toolCompactAggressive  = 80
	maxContextCompacts     = 2
)

// contextFitOpts controls how aggressively fitMessagesToBudget shrinks msgs.
type contextFitOpts struct {
	// Aggressive uses a shorter tool-result cap (overflow retry path).
	Aggressive bool
	// KeepTail is how many trailing messages to protect from drop (not from tool compact).
	KeepTail int
}

func softContextBudget(budget int) int {
	if budget <= 0 {
		return 0
	}
	return int(float64(budget) * softBudgetRatio)
}

func estimateChars(msgs []llmclient.Message) int {
	n := 0
	for _, m := range msgs {
		n += len(m.Role) + len(m.Content) + len(m.Thinking) + len(m.ToolCallID) + len(m.ToolName)
		if len(m.ToolCalls) > 0 {
			b, err := json.Marshal(m.ToolCalls)
			if err == nil {
				n += len(b)
			} else {
				n += 64 * len(m.ToolCalls)
			}
		}
		n += 8 // role / framing overhead
	}
	return n
}

func compactToolContent(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "\n...[compacted]"
}

func cloneMessages(msgs []llmclient.Message) []llmclient.Message {
	out := make([]llmclient.Message, len(msgs))
	copy(out, msgs)
	for i := range out {
		if len(out[i].ToolCalls) > 0 {
			out[i].ToolCalls = append([]llmclient.ToolCall(nil), out[i].ToolCalls...)
		}
	}
	return out
}

// fitMessagesToBudget shrinks in-memory LLM messages to fit under budget chars.
// It never rewrites DB history. budget <= 0 disables fitting.
func fitMessagesToBudget(msgs []llmclient.Message, budget int, opts contextFitOpts) []llmclient.Message {
	if budget <= 0 || len(msgs) == 0 {
		return msgs
	}
	target := budget
	if !opts.Aggressive {
		target = softContextBudget(budget)
		if target <= 0 {
			target = budget
		}
	}
	if estimateChars(msgs) <= target {
		return msgs
	}

	out := cloneMessages(msgs)
	keepTail := opts.KeepTail
	if keepTail <= 0 {
		keepTail = defaultKeepTailMsgs
	}
	toolMax := toolCompactSoft
	if opts.Aggressive {
		toolMax = toolCompactAggressive
	}

	// 1) Compact older tool results (preserve the protected tail verbatim).
	protectFrom := len(out) - keepTail
	if protectFrom < 0 {
		protectFrom = 0
	}
	for i := 0; i < protectFrom; i++ {
		if out[i].Role == "tool" && out[i].Content != "" {
			out[i].Content = compactToolContent(out[i].Content, toolMax)
		}
	}
	if estimateChars(out) <= target {
		return sanitizeLLMMessages(out)
	}

	// 2) Also compact tool results inside the tail if still over budget.
	for i := protectFrom; i < len(out); i++ {
		if out[i].Role == "tool" && out[i].Content != "" {
			out[i].Content = compactToolContent(out[i].Content, toolMax)
		}
	}
	if estimateChars(out) <= target {
		return sanitizeLLMMessages(out)
	}

	// 3) Drop middle turns while keeping system + last message (+ tail).
	out = dropMiddleTurns(out, target, keepTail)
	return sanitizeLLMMessages(out)
}

// dropMiddleTurns removes older non-system messages until under target, always
// keeping the first system message(s) and the final message.
func dropMiddleTurns(msgs []llmclient.Message, target, keepTail int) []llmclient.Message {
	if len(msgs) <= 2 {
		return msgs
	}
	sysEnd := 0
	for sysEnd < len(msgs) && msgs[sysEnd].Role == "system" {
		sysEnd++
	}
	if sysEnd >= len(msgs) {
		return msgs
	}

	// Keep at least the last message and ideally keepTail messages.
	keep := keepTail
	if keep < 1 {
		keep = 1
	}
	for estimateChars(msgs) > target && len(msgs)-sysEnd > keep {
		// Drop the oldest non-system message; skip leading orphan tools after cut.
		dropAt := sysEnd
		msgs = append(msgs[:dropAt:dropAt], msgs[dropAt+1:]...)
		for len(msgs) > sysEnd && msgs[sysEnd].Role == "tool" {
			msgs = append(msgs[:sysEnd:sysEnd], msgs[sysEnd+1:]...)
		}
		// If we would drop below keep, stop.
		if len(msgs)-sysEnd <= keep {
			break
		}
		// Gradually allow dropping more of the "tail" if still oversized.
		if keep > 1 && estimateChars(msgs) > target {
			keep--
		}
	}
	return msgs
}

// sanitizeLLMMessages mirrors sanitizeToolHistory for llmclient.Message.
func sanitizeLLMMessages(msgs []llmclient.Message) []llmclient.Message {
	if len(msgs) == 0 {
		return msgs
	}
	out := make([]llmclient.Message, 0, len(msgs))
	for i := 0; i < len(msgs); {
		m := msgs[i]
		if m.Role == "tool" {
			i++
			continue
		}
		if m.Role != "assistant" || len(m.ToolCalls) == 0 {
			out = append(out, m)
			i++
			continue
		}
		needed := make(map[string]llmclient.ToolCall, len(m.ToolCalls))
		uniqCalls := make([]llmclient.ToolCall, 0, len(m.ToolCalls))
		for _, tc := range m.ToolCalls {
			if tc.ID == "" {
				continue
			}
			if _, ok := needed[tc.ID]; ok {
				continue
			}
			needed[tc.ID] = tc
			uniqCalls = append(uniqCalls, tc)
		}
		m.ToolCalls = uniqCalls
		found := make(map[string]llmclient.Message, len(needed))
		j := i + 1
		for j < len(msgs) && msgs[j].Role == "tool" {
			id := msgs[j].ToolCallID
			if _, ok := needed[id]; ok {
				if _, have := found[id]; !have {
					found[id] = msgs[j]
				}
			}
			j++
		}
		out = append(out, m)
		for _, tc := range uniqCalls {
			if tm, ok := found[tc.ID]; ok {
				out = append(out, tm)
				continue
			}
			out = append(out, llmclient.Message{
				Role:       "tool",
				Content:    "error: tool call interrupted or result missing",
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
			})
		}
		i = j
	}
	return out
}
