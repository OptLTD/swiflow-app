---
slug: reflection-loop
name: Reflection Loop
description: Review recent experiences, promote recurring workflows to skills, and refine the agent's Ways of working charter. Optionally schedule a weekly cron.
---

## When to use

When the user says "set up reflection", "enable self-learning loop", "analyse past experiences", or "update ways of working".

## Steps

1. Call `experience_list` with `limit: 20` to retrieve recent experiences.
2. Group by tags; identify patterns that appear in 3 or more experiences.
3. For each recurring **workflow** pattern, call `skill_draft` with a new SKILL.md that captures it as a reusable skill.
4. For recurring **directional** principles (how to choose at forks — encoding defaults, "conclusion first", etc.), summarize them clearly for the user and ask whether to fold them into the agent charter (Ways of working). Do **not** wait for a multi-step Accept queue — if the user agrees in chat, they can paste/edit charter in Agent settings, or you may note the principle so a follow-up correction message can append it.
5. Report a brief summary of patterns found, drafts created, and charter suggestions.
6. Ask the user if they want a weekly reflection cron. If yes, call `schedule_create` with:
   - `schedule`: `"0 9 * * 1"` (Monday 9 am)
   - `message`: `"Run the reflection-loop skill: review recent experiences, promote patterns to skills, and suggest Ways of working updates."`
