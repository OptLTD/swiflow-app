---
slug: document-extract
name: Document Extract
description: OCR / extract fields from images and PDFs. Single file → document_extract; ≥3 files or table/Excel batch → delegate_task (do not OCR on the main agent).
---

# Document Extract

When the user asks to read, OCR, transcribe, or extract content from an **image**, **PDF**, or **scan**, do **not** say you cannot read images.

## Main agent vs sub-agent (required)

| Situation | What to do |
|-----------|------------|
| **One or two** files, quick read / OCR | Main may call `document_extract` (cost probe) |
| Probe shows tools are **slow** (soft-async still running, more work left) | Runtime full-handoff: main loses `document_extract`, **must** `delegate_task` for the rest |
| Many files / Excel-table intent is obvious up front | Prefer `delegate_task` immediately; **do not** ask which columns — use sensible defaults |

For batch / table work, the child should:

1. Run `document_extract` on each `@/` path (soft-async may overlap).
2. Write the spreadsheet or structured result under the workspace (e.g. `extract-result.xlsx`).
3. Return a **short summary** with the output path and row count — not full OCR JSON.

## Tool (single-file / small jobs)

| Tool | When to use |
|------|-------------|
| `document_extract` | One (or two) workspace image/PDF/text file needs content or fields |
| `delegate_task` | Batch OCR / many files / Excel or table deliverable |

Prefer `document_extract` over guessing from the filename. Prefer it over `fs_read` for binary images and scanned PDFs.

## Single-file workflow

1. Resolve the file path (user message, `window_active`, or `fs_list`).
2. Call `document_extract` with at least one of:
   - `prompt` — free-form instructions (e.g. "提取图片中的全部可见文字")
   - `fields` — flat field names to pull (e.g. `["title","amount","date"]`)
   - `schema` — JSON object describing the target structure
3. For plain OCR / “读一下这张图”, use `prompt` plus `include_raw_text: true` when helpful.
4. Summarize the returned JSON for the user; quote key text when they asked for transcription.

## Batch example (main agent)

One `delegate_task` only. Put **every** remaining `@/` path inside `goal` (not a `path` field — that param does not exist). Do **not** pass `tools`; the child picks `document_extract` / `fs_*` itself. Ask the child to write e.g. `@/weighbridge.xlsx` and return only that path + row count.

```json
{
  "goal": "OCR these files and write @/weighbridge.xlsx with columns ...:\n@/a.png\n@/b.png\n@/c.png",
  "context": "Vehicle weighbridge slips",
  "max_rounds": 16
}
```

## Single-file examples

```json
{"path":"receipt.png","prompt":"提取图片中的全部文字","include_raw_text":true}
```

```json
{"path":"invoice.pdf","fields":["vendor","total","date"]}
```

## Failures

- If the tool says document is disabled or provider not configured, tell the user to configure a **视觉模型** (Settings → Agent → Provider → 视觉模型) or enable document extraction.
- Do not invent file contents when the tool fails.
