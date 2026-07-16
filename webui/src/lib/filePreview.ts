import type { Extension } from '@codemirror/state'

export type PreviewKind = 'text' | 'markdown' | 'excel' | 'pdf' | 'doc' | 'image' | 'unsupported'

const BINARY_KINDS: Record<string, PreviewKind> = {
  xlsx: 'excel',
  xls: 'excel',
  xlsm: 'excel',
  pdf: 'pdf',
  docx: 'doc',
  doc: 'doc',
  png: 'image',
  jpg: 'image',
  jpeg: 'image',
  gif: 'image',
  webp: 'image',
  bmp: 'image',
  ico: 'image',
  avif: 'image',
  svg: 'image',
}

const MARKDOWN_EXTENSIONS = new Set(['md', 'markdown'])

const TEXT_EXTENSIONS = new Set([
  'txt', 'md', 'markdown', 'json', 'yaml', 'yml', 'xml', 'html', 'htm', 'css',
  'js', 'jsx', 'ts', 'tsx', 'vue', 'py', 'go', 'rs', 'java', 'c', 'cpp', 'h',
  'hpp', 'cs', 'rb', 'php', 'swift', 'kt', 'scala', 'sql', 'sh', 'bash', 'zsh',
  'fish', 'env', 'ini', 'toml', 'cfg', 'conf', 'log', 'csv', 'gitignore',
  'dockerfile', 'makefile', 'gradle', 'properties',
])

export function fileExtension(path: string): string {
  const base = path.split('/').pop() || path
  const dot = base.lastIndexOf('.')
  if (dot <= 0) return ''
  return base.slice(dot + 1).toLowerCase()
}

export function previewKind(path: string): PreviewKind {
  const ext = fileExtension(path)
  if (BINARY_KINDS[ext]) return BINARY_KINDS[ext]
  if (MARKDOWN_EXTENSIONS.has(ext)) return 'markdown'
  if (TEXT_EXTENSIONS.has(ext) || !ext) return 'text'
  return 'unsupported'
}

export interface ExcelSheet {
  name: string
  data: string[][]
}

const IMAGE_MIME: Record<string, string> = {
  png: 'image/png',
  jpg: 'image/jpeg',
  jpeg: 'image/jpeg',
  gif: 'image/gif',
  webp: 'image/webp',
  bmp: 'image/bmp',
  ico: 'image/x-icon',
  avif: 'image/avif',
  svg: 'image/svg+xml',
}

export function imageMimeType(path: string): string {
  const ext = fileExtension(path)
  return IMAGE_MIME[ext] || 'application/octet-stream'
}

/** Lazy-loaded so opening text files does not inflate the initial desktop bundle. */
export async function codemirrorLanguage(path: string): Promise<Extension | null> {
  const ext = fileExtension(path)
  const [
    { javascript },
    { json },
    { markdown },
    { python },
    { css },
    { html },
    { yaml },
  ] = await Promise.all([
    import('@codemirror/lang-javascript'),
    import('@codemirror/lang-json'),
    import('@codemirror/lang-markdown'),
    import('@codemirror/lang-python'),
    import('@codemirror/lang-css'),
    import('@codemirror/lang-html'),
    import('@codemirror/lang-yaml'),
  ])
  const LANG_MAP: Record<string, () => Extension> = {
    js: javascript,
    jsx: () => javascript({ jsx: true }),
    ts: () => javascript({ typescript: true }),
    tsx: () => javascript({ jsx: true, typescript: true }),
    json,
    py: python,
    md: markdown,
    markdown,
    yaml,
    yml: yaml,
    html,
    htm: html,
    xml: html,
    css,
    vue: html,
    sh: () => javascript(),
    bash: () => javascript(),
    zsh: () => javascript(),
    go: () => javascript(),
  }
  const factory = LANG_MAP[ext]
  return factory ? factory() : null
}
