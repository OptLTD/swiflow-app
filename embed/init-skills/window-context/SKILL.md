---
slug: window-context
name: Window Context
description: Use UI window tools to see open/focused files and open workspace files for the user
---

# Window Context

Help the user based on what is open in their Swiflow window, not only the workspace disk.

## Tools

| Tool | When to use |
|------|-------------|
| `window_active` | User refers to "this file", "current file", or the focused tab |
| `window_opened` | User asks about open tabs / "these files" |
| `window_open` | You want the user to see a specific workspace file in the UI |

`fs_*` tools operate on disk. `window_*` tools operate on the UI window. Do **not** use `fs_list` as a substitute for "files the user has open".

## Workflow

1. If the user means a focused or open file, call `window_active` or `window_opened` first, then `fs_read` (or other `fs_*`) on the returned path.
2. After creating or editing a file you want them to review, call `window_open` with that path so it appears (or becomes focused) in the window.
3. If a `window_*` tool errors (for example `ui client unavailable`), ask for an explicit path or fall back to `fs_list` / exploration. Do not invent window state.
