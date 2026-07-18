---
slug: build-light-app
name: Build Light App
description: Build and launch a self-contained light app in a Wails child window, isolated from the workspace
---

# Build App

Build a small, standalone app and open it in its own window alongside the workspace. Apps live under `data/light-apps/` and are fully isolated — they cannot read workspace files.

## Tools (use ONLY these for light apps)

| Tool | Purpose |
|------|---------|
| `clarify` | Confirm plan, **storage**, and **acceptance spec** before creating. **Required for new apps.** |
| `browser` | **Development:** self-test launch URL against `SPEC.md` (navigate / content / click / type / eval / screenshot). |
| `light_app_list` | List registered apps (`id`, `name`, `runtime`, `status`). **Start here** when editing an existing app. |
| `light_app_create` | Register a new app and create its directory. Returns `id`. |
| `light_app_write` | Write a file: path = `"<app_id>/<file>"` (e.g. `"abc/index.html"`). |
| `light_app_read` | Read a file: path = `"<app_id>/<file>"`. |
| `light_app_ls` | List files inside one app: path = `"<app_id>"` or `"<app_id>/subdir"`. |
| `light_app_launch` | Start/restart the process → `{ url, port, name }`. Needed for both self-test and later use. |
| `light_app_open` | **Usage only:** open the child window for the user after self-test has passed. |

### Do NOT use workspace tools on light apps

Light apps are **outside** the workspace sandbox. Never use:

- `fs_list` / `fs_read` / `fs_write` / `fs_delete`
- `exec` (`find`, `ls`, `cat`, `sed`, …) on `data/light-apps/`

Wrong path args like `"data/light-apps 内容"` will fail. Correct flow for edits:

```
light_app_list                          → find id by name
light_app_read({ path: "<id>/index.html" })
light_app_write({ path: "<id>/index.html", content: "..." })
light_app_launch({ id }) → light_app_open({ url, title })
```

## Plan before create (mandatory)

**Do not call `light_app_create` until the user confirms a short plan + acceptance spec.** Static HTML alone has no server DB — if you skip storage confirmation, apps silently fall back to `localStorage` / in-memory and users are surprised. If you skip acceptance criteria, “done” is undefined and you cannot self-test.

Use `clarify` (one or more rounds). Cover at least:

1. **What the app does** (one sentence)
2. **Runtime**: `static` vs `python`
3. **Data storage** (pick one — see table below)
4. **Secrets / env keys** needed (tokens, DSN, etc.)
5. **Acceptance spec** — 3–8 checkable “符合预期” items (see below)

### Data storage options

| Option | Runtime | When to use | How |
|--------|---------|-------------|-----|
| **None / ephemeral** | static or python | Demo, calculator, one-shot UI | In-memory only; refresh loses data |
| **Browser `localStorage`** | static | Small personal prefs / drafts, single device | `localStorage.setItem` — not shared, easy to clear |
| **JSON/files in app dir** | **python** | Structured app data without a DB | Read/write files under the app directory (e.g. `data.json`) via Flask/FastAPI |
| **External database** | **python** | Real persistence, multi-session, relational | User provides DSN/URL via Light Apps env (e.g. `DATABASE_URL`); never hard-code credentials |
| **Export/import only** | static | User owns the file | Download/upload JSON — no automatic persistence |

**Default guidance:**
- Need durable or shared data → prefer **python + files** or **python + DB** (ask for connection string / env key).
- Pure UI with tiny local state → static + `localStorage` is OK **if the user explicitly accepts it**.
- Do **not** silently choose static + localStorage when the user expects “存到服务器/数据库”.

Example `clarify` options for storage:

```
- 不持久化（刷新即丢）
- 浏览器 localStorage（仅本机）
- Python + 应用目录 JSON 文件
- Python + 外部数据库（我会提供 DATABASE_URL）
```

### Acceptance spec（符合预期）

During the same confirmation phase, propose a concrete checklist and get the user to confirm or edit it. Each item must be **observable in the browser** (UI text, map center, search results visible, button works, etc.).

Write the confirmed list into the app as `SPEC.md` via `light_app_write` right after create (path `"<id>/SPEC.md"`). Later self-test **only** against this file — not against vague memory.

Example `SPEC.md`:

```markdown
# Spec: 旅行地图

## Storage
- runtime: static
- data: localStorage (places list)
- env: TIANDITU_TOKEN

## Acceptance
1. 打开后地图中心在中国（约 lat 35–40, lng 100–120），不是澳大利亚
2. 搜索框输入城市名后，结果列表有条目可点击
3. 点击地图可添加地点，侧栏列表出现该地点
4. 刷新页面后地点仍在（localStorage）
5. 页面无 JS 报错（控制台 / eval 检查）
```

Only after the user confirms plan **and** acceptance items → proceed to create.

## Development: self-test with browser (mandatory)

Self-test is part of **development**, not handoff. Loop until SPEC passes **before** opening the app for the user.

```
light_app_write → light_app_launch → browser (test SPEC) → fail? fix & repeat
```

Do **not** call `light_app_open` in this loop.

After `light_app_launch`, use the `browser` tool on the launch `url` (`http://127.0.0.1:<port>`):

1. `navigate` → launch URL  
2. `content` and/or `screenshot` — page loads, no blank/error splash  
3. `eval` — key invariants (map center, no missing-env throw, etc.)  
4. `click` / `type` — primary flows from `SPEC.md`  
5. Record each SPEC item pass/fail  

Rules:

- Required for new apps and non-trivial UX/data edits.
- Fail → `light_app_write` → `light_app_launch` → re-test. Stay in the development loop.
- Test with **`browser`**, not by eyeballing the Wails window.
- If `browser` is disabled: say so, paste SPEC, ask the user to verify — do not claim “已符合预期”.

## Usage: open for the user

Only after self-test has passed, call `light_app_open` so the user gets the child window. This is **usage / handoff**, not part of debugging.

```
# development (above) finished with all SPEC items green
light_app_open({ url, title: name })
```

If the app is already running and SPEC was already verified, `light_app_open` alone is enough when the user just wants to use it again.

Do **not** only paste the URL unless `light_app_open` fails (CLI / no UI client).

## Environment variables

Users configure secrets in **Light Apps → Environment Variables** (e.g. `TIANDITU_TOKEN`, `DATABASE_URL`). Injected at every launch.

### Static — `window.swiflow.env(key)`

Auto-injected into HTML at launch (inline `<script>` at the start of `<head>`). **Required keys must use `.env()`** — missing or empty values throw, so users and tests notice immediately. Do **not** hard-code secrets or silently fall back.

```js
const TOKEN = window.swiflow.env('TIANDITU_TOKEN');
```

If this throws, tell the user to add the key under Light Apps → Environment Variables, then re-launch.

### Python — `os.environ["KEY"]`

```python
token = os.environ["TIANDITU_TOKEN"]  # KeyError if missing — preferred for required secrets
dsn = os.environ["DATABASE_URL"]      # when storage = external DB
```

Avoid `os.environ.get("KEY", "")` for required secrets (silent empty string hides misconfiguration).

**Rules:**
- Never hard-code API keys/tokens/DSNs in source.
- If the user mentions a token/key/DSN, tell them to set it in Light Apps env, then re-launch.
- Changing env requires a re-launch (`light_app_launch` restarts the process).

## Runtimes

### `static`
Plain HTML + CSS + JS. No server. Fine for UI-only or `localStorage` / export-import. **Cannot** write server-side files or talk to a DB unless the user only uses remote HTTP APIs from the browser (CORS permitting).

Entry point: `index.html`

### `python`
Flask or FastAPI. Use when you need server-side logic, file persistence in the app directory, or an external database.

Entry point: `app.py` — must bind to `0.0.0.0` on the port provided via the `PORT` environment variable:
```python
import os
port = int(os.environ.get("PORT", 8000))
app.run(host="0.0.0.0", port=port)
```
List pip dependencies in `requirements.txt` (one package per line, no version pins unless the user specifies).

## Capabilities

- Single-page UIs with vanilla JS or inline scripts (Chart.js, Alpine.js, etc. via CDN)
- REST endpoints with Flask/FastAPI
- File-based persistence in the app directory (**python**)
- External DB via env DSN (**python**, user-provided)
- Browser `localStorage` / export-import (**static**, only if confirmed)
- Secrets via Light Apps environment variables

## Constraints

- No Node/npm — `static` apps must use CDN scripts or inline code only
- No compiled languages (Go, Rust, etc.)
- No access to workspace files from within the running app
- Port is OS-assigned and ephemeral; always use the `PORT` env var
- One process per app ID — `light_app_launch` stops the previous instance then starts fresh
- No sudo, no system-level packages; only pip dependencies
- App process has no network egress restrictions, but should not exfiltrate user data
- **Never** `fs_*` / `exec` for light-app files — only `light_app_*`
- **Never** create before plan + acceptance spec are confirmed via `clarify`
- **Never** call `light_app_open` until browser self-test against `SPEC.md` has passed (unless browser disabled)
- **Never** claim done without that self-test

## Workflow

### New app

**Plan**
1. Understand the goal.
2. **`clarify`** — purpose, runtime, **data storage**, secrets, **acceptance spec**. Stop until answered.
3. Env keys needed → tell user to set them (or fail loudly later via `swiflow.env` / `os.environ`).

**Develop**
4. `light_app_create` → save `id`.
5. `light_app_write` `"<id>/SPEC.md"`.
6. `light_app_write` sources.
7. `light_app_launch` → **`browser` self-test** each SPEC item → fix / re-launch until all pass.  
   Do **not** `light_app_open` yet.

**Use (handoff)**
8. `light_app_open` → report SPEC pass results to the user.

### Edit existing app (e.g.「修改旅行地图」)

**Develop**
1. `light_app_list` → `id`.
2. Read `SPEC.md` (or entry file); update SPEC via `clarify` if behavior/storage changes.
3. `light_app_write` the fix.
4. `light_app_launch` → **`browser` self-test** until SPEC passes.

**Use**
5. `light_app_open`.

## Templates

### Static — minimal shell
```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>App</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 2rem; }
  </style>
</head>
<body>
  <h1>Hello</h1>
  <script>
    // Required secrets — throws if unset:
    // const token = window.swiflow.env('TIANDITU_TOKEN');
  </script>
</body>
</html>
```

### Python — file-backed JSON
```python
import json, os
from pathlib import Path
from flask import Flask, jsonify, request

app = Flask(__name__)
DATA = Path(__file__).with_name("data.json")

def load():
    if DATA.exists():
        return json.loads(DATA.read_text(encoding="utf-8"))
    return {"items": []}

def save(obj):
    DATA.write_text(json.dumps(obj, ensure_ascii=False, indent=2), encoding="utf-8")

@app.route("/")
def index():
    return app.send_static_file("index.html")

@app.route("/api/items", methods=["GET", "POST"])
def items():
    data = load()
    if request.method == "POST":
        data["items"].append(request.get_json(force=True))
        save(data)
    return jsonify(data)

if __name__ == "__main__":
    port = int(os.environ.get("PORT", 8000))
    app.run(host="0.0.0.0", port=port)
```

`requirements.txt`:
```
flask
```

## Error handling

- If `light_app_launch` fails with a missing-dependency error, check `requirements.txt` and re-launch.
- If the port is busy (rare), call `light_app_launch` again — a new port will be assigned.
- If `light_app_open` is unavailable (CLI mode), report the `url` to the user so they can open it manually.
- If `swiflow.env('KEY')` / `os.environ["KEY"]` throws, tell the user to set `KEY` in Light Apps env and re-launch — do not hard-code the value.
- If you do not know the app id, call `light_app_list` — do not `fs_list` / `exec` under `data/light-apps`.
- If the user wants persistence but you already built static-only, propose migrating to python (+ files or DB) via `clarify` before rewriting.
- If browser self-test fails a SPEC item, fix and re-test; do not report success.
- If `SPEC.md` is missing on an old app, draft one with `clarify` before claiming the edit is complete.
