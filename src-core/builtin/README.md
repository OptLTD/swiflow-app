# Builtin Tools Module

The builtin tools module provides a small set of ready-to-use capabilities exposed directly from the backend. These are designed to be reliable primitives for common tasks and to complement MCP-based tools.

## Overview

- Entry point: `builtin/manager.go`
- Interface: any builtin implements `Prompt() string` and `Handle(args string) (string, error)`
- Manager initialization reads `cfg-data` from storage to configure certain tools (e.g., LLM clients)
- Saved user tools are wrapped as command aliases and exposed alongside builtins

## Builtin Tools

- `command` — execute a system shell command within the app’s working home
- `python3` — run Python 3 code snippets (requires Python in PATH)
- `image-ocr` — perform OCR on image content using configured LLM provider
- `chat2llm` — send text to an LLM and return the response
- `cmd-alias` — auto-generated entries that wrap saved tools (based on `llm_tool` storage records)

Each tool exposes:

- `Prompt()` — human-readable usage guide string for inclusion in model instructions
- `Handle(args)` — execution path that processes the input and returns output or error

## Manager Lifecycle

The manager is accessed through `builtin.GetManager()` and initialized via `Init(store)`:

1. Clears the internal tool list to avoid duplicates on re-init
2. Loads `cfg-data` records to build clients for tools like `image-ocr` and `chat2llm`
3. Appends builtins: `CommandTool`, `Python3Tool`, and conditionally `ImageOCRTool`, `Chat2LLMTool`
4. Loads saved tools from `llm_tool` and appends them as `CmdAliasTool`

Example initialization path:

```go
manager := builtin.GetManager().Init(store)
```

## Listing Tools

The manager provides a read-only listing method for external callers:

```go
type BuiltinInfo struct {
    Name   string // readable name or UUID for alias
    Kind   string // "builtin" or "alias"
    Prompt string // usage prompt
}

infos := builtin.GetManager().List()
```

This supports UI layers to render available tools and display usage guidance.

## Tool Selection

At runtime, tools are resolved by name through `Query(name)`:

- `chat2llm`, `image-ocr`, `command`, `python3` map to their respective builtin implementations
- Saved aliases are resolved by their `UUID` (the UUID generated for the saved tool record)

## Storage Model

Saved tools (aliases) are recorded in the `llm_tool` table:

- `uuid` — unique identifier, 12–16 characters (auto-generated if missing)
- `type` — typically `alias` for saved tools
- `name` — display name
- `desc` — description
- `code` — underlying base command or script
- `deps` — dependency string (optional, implementation-specific)

These are loaded during manager initialization and wrapped as `CmdAliasTool` for execution.

## HTTP API

A unified tool endpoint is exposed for listing builtins and managing saved aliases:

- `GET /api/tool?act=get-tools` — returns `[]BuiltinInfo` with names, kinds, and prompts
- `POST /api/tool?act=set-tool` — saves or updates a tool alias

Example payload for `set-self`:

```json
{
  "uuid": "",         // optional; auto-generated if empty
  "type": "alias",     // optional; defaults to "alias"
  "name": "my command",
  "desc": "quick helper",
  "code": "ls -la",
  "deps": ""          // optional; implementation-specific
}
```

On successful save, the builtin manager re-initializes to include the new alias in the listing.

## Adding a New Builtin

1. Implement the interface:

```go
type MyTool struct {}
func (t *MyTool) Prompt() string { /* describe usage */ }
func (t *MyTool) Handle(args string) (string, error) { /* do work */ }
```

2. Append it during manager initialization:

```go
func (m *BuiltinManager) Init(store storage.MyStore) *BuiltinManager {
    // ... existing setup
    m.Append(&MyTool{})
    return m
}
```

3. Extend `Query(name)` with a case mapping the external name to your type.

## Notes

- Prompts are intended for model instruction scaffolding; keep them concise and actionable
- Always use English comments in code to clarify intent for future maintainers
- Avoid side effects in `Prompt()`; all execution must be in `Handle()`