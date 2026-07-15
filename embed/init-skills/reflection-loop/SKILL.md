---
slug: reflection-loop
name: Reflection Loop
description: Review recent experiences and synthesise recurring patterns into reusable skills. Set up a weekly cron to keep the loop running automatically.
---

## When to use

When the user says "set up reflection", "enable self-learning loop", or "analyse past experiences".

## Steps

1. Call `experience_list` with `limit: 20` to retrieve recent experiences.
2. Group by tags; identify patterns that appear in 3 or more experiences.
3. For each recurring pattern, call `skill_draft` with a new SKILL.md that captures the pattern as a reusable workflow.
4. Report a brief summary of patterns found and drafts created.
5. Ask the user if they want a weekly reflection cron. If yes, call `schedule_create` with:
   - `schedule`: `"0 9 * * 1"` (Monday 9 am)
   - `message`: `"Run the reflection-loop skill: review recent experiences and promote patterns to skills."`
