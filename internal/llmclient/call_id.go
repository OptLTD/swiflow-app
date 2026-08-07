package llmclient

import "fmt"

// dedupeCallIDs ensures tool-call ids are unique across the message list.
//
// Responses API providers (DeepSeek, OpenAI-compatible /v1/responses) reject
// requests with Duplicate 'call_id'. Duplicates show up when:
//   - a single assistant turn lists the same tool call id twice
//   - a later turn reuses an earlier call_id
//   - history contains a repeated tool-result for the same id
//
// Within a turn, extra tool_calls with a repeated id are dropped. Across turns,
// colliding ids are rewritten and matching tool results are remapped. Extra
// tool results for an already-closed call_id are dropped.
func dedupeCallIDs(msgs []Message) []Message {
	if len(msgs) == 0 {
		return msgs
	}
	out := make([]Message, len(msgs))
	copy(out, msgs)
	for i := range out {
		if len(out[i].ToolCalls) > 0 {
			out[i].ToolCalls = append([]ToolCall(nil), out[i].ToolCalls...)
		}
	}

	used := map[string]struct{}{}
	for i := range out {
		if out[i].Role != "assistant" || len(out[i].ToolCalls) == 0 {
			continue
		}
		remap := map[string]string{}
		inTurn := map[string]struct{}{}
		fresh := make([]ToolCall, 0, len(out[i].ToolCalls))
		for _, tc := range out[i].ToolCalls {
			id := tc.ID
			if id == "" {
				id = fmt.Sprintf("call_gen_%d", len(used)+len(fresh)+1)
				tc.ID = id
			}
			if _, dup := inTurn[id]; dup {
				continue
			}
			inTurn[id] = struct{}{}
			final := id
			if _, taken := used[id]; taken {
				final = fmt.Sprintf("%s_d%d", id, len(used)+1)
				tc.ID = final
				remap[id] = final
			}
			used[final] = struct{}{}
			fresh = append(fresh, tc)
		}
		out[i].ToolCalls = fresh
		for j := i + 1; j < len(out) && out[j].Role == "tool"; j++ {
			if nid, ok := remap[out[j].ToolCallID]; ok {
				out[j].ToolCallID = nid
			}
		}
	}

	seenOut := map[string]struct{}{}
	final := make([]Message, 0, len(out))
	for _, m := range out {
		if m.Role == "tool" && m.ToolCallID != "" {
			if _, ok := seenOut[m.ToolCallID]; ok {
				continue
			}
			seenOut[m.ToolCallID] = struct{}{}
		}
		final = append(final, m)
	}
	return final
}
