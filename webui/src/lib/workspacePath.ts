/** Workspace path helpers: @/ relative to workspace root (like Vue @/). */

export const UPLOAD_FILES_START = '[UPLOAD FILES START]'
export const UPLOAD_FILES_END = '[UPLOAD FILES END]'

const UPLOAD_BLOCK_RE =
  /(?:^|\n)\[UPLOAD FILES START\]\n([\s\S]*?)\n\[UPLOAD FILES END\](?:\n|$)/

export function toAtPath(rel: string): string {
  let p = (rel || '').trim().replace(/\\/g, '/')
  if (p.startsWith('@/')) return p
  while (p.startsWith('./')) p = p.slice(2)
  if (p === '.' || p === '') return '@/'
  if (p.startsWith('/')) p = p.replace(/^\/+/, '')
  return '@/' + p
}

/** Strip leading @/ → workspace-relative path (no leading slash). */
export function fromAtPath(at: string): string {
  let p = (at || '').trim().replace(/\\/g, '/')
  if (p.startsWith('@/')) p = p.slice(2)
  while (p.startsWith('./')) p = p.slice(2)
  return p.replace(/^\/+/, '')
}

const AT_PATH_RE = /@\/[^\s\n]+/g

function uniqueAtPaths(text: string): string[] {
  if (!text) return []
  const seen = new Set<string>()
  const out: string[] = []
  for (const m of text.matchAll(AT_PATH_RE)) {
    const at = m[0]
    if (seen.has(at)) continue
    seen.add(at)
    out.push(at)
  }
  return out
}

/**
 * Extract unique @/… references.
 * Prefer the [UPLOAD FILES START/END] block when present; otherwise scan whole text
 * (legacy messages that appended bare @/ lines).
 */
export function parseAtPaths(text: string): string[] {
  if (!text) return []
  const block = text.match(UPLOAD_BLOCK_RE)
  if (block) return uniqueAtPaths(block[1])
  return uniqueAtPaths(text)
}

/** User-visible body without the upload-files block (and trim). */
export function displayMessageBody(text: string): string {
  if (!text) return ''
  return text.replace(UPLOAD_BLOCK_RE, '\n').replace(/\n{3,}/g, '\n\n').trim()
}

export function composeMessageWithAttachments(body: string, atPaths: string[]): string {
  const text = body.trimEnd()
  const refs = atPaths.map(toAtPath).filter((p) => p !== '@/')
  if (!refs.length) return text.trim()
  const block = `${UPLOAD_FILES_START}\n${refs.join('\n')}\n${UPLOAD_FILES_END}`
  if (!text.trim()) return block
  return text.trimEnd() + '\n\n' + block
}
