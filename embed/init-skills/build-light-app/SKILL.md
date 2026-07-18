---
slug: build-light-app
name: Build Light App
description: Build and launch a self-contained light app in a Wails child window, isolated from the workspace
---

# Build App

Build a small, standalone app and open it in its own window alongside the workspace. Apps live under `data/light-apps/` and are fully isolated — they cannot read workspace files.

## Tools

| Tool | Purpose |
|------|---------|
| `light_app_create` | Register the app in the database and create its directory. **Always call this first.** Returns `id`. |
| `light_app_write` | Write a file into the app directory (`data/light-apps/<id>/`). |
| `light_app_read` | Read a file from the app directory. |
| `light_app_ls` | List files in the app directory. |
| `light_app_launch` | Start the app process and return its local `url`. |
| `light_app_open` | Open a URL in the desktop browser. Call this after `light_app_launch` with the returned `url`. |

## Runtimes

### `static`
Plain HTML + CSS + JS. No server needed. Use for dashboards, calculators, visualisations, and forms that don't need a backend.

Entry point: `index.html`

### `python`
Flask or FastAPI. Use for apps that need server-side logic, data processing, or calling external APIs.

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
- File-based persistence: read/write files in the app directory
- Calling external HTTP APIs from Python
- Displaying data passed from the workspace by the agent (write a JSON data file, then reference it)

## Constraints

- No Node/npm — `static` apps must use CDN scripts or inline code only
- No compiled languages (Go, Rust, etc.)
- No access to workspace files from within the running app
- Port is OS-assigned and ephemeral; always use the `PORT` env var
- One process per app ID — re-launching stops the previous instance
- No sudo, no system-level packages; only pip dependencies
- App process has no network egress restrictions, but should not exfiltrate user data

## Workflow

1. Clarify the goal: what does the app do, what data does it need, which runtime fits.
2. Call `light_app_create` with `name`, `runtime`, and optionally `description` and `entry_point`. Save the returned `id`.
3. Write all source files with `light_app_write` (`path` is relative to the app root, e.g. `index.html` or `app.py`).
4. For Python apps, write `requirements.txt` if any packages are needed.
5. Call `light_app_launch` with the `id`. It returns `{ "url": "...", "port": ... }`.
6. Call `light_app_open` with the `url` to open the app in the desktop browser.
7. If the user requests changes, use `light_app_write` to update files, then `light_app_launch` again to restart.

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
    // your logic here
  </script>
</body>
</html>
```

### Python — minimal Flask
```python
import os
from flask import Flask, jsonify

app = Flask(__name__)

@app.route("/")
def index():
    return app.send_static_file("index.html")

@app.route("/api/data")
def data():
    return jsonify({"status": "ok"})

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
