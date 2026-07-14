import { javascript } from '@codemirror/lang-javascript'
import { json } from '@codemirror/lang-json'
import { markdown } from '@codemirror/lang-markdown'
import { python } from '@codemirror/lang-python'
import { css } from '@codemirror/lang-css'
import { html } from '@codemirror/lang-html'
import { yaml } from '@codemirror/lang-yaml'
import type { Extension } from '@codemirror/state'

export type PreviewKind = 'text' | 'excel' | 'pdf' | 'doc' | 'unsupported'

const BINARY_KINDS: Record<string, PreviewKind> = {
  xlsx: 'excel',
  xls: 'excel',
  xlsm: 'excel',
  pdf: 'pdf',
  docx: 'doc',
  doc: 'doc',
}

const TEXT_EXTENSIONS = new Set([
  'txt', 'md', 'markdown', 'json', 'yaml', 'yml', 'xml', 'html', 'htm', 'css',
  'js', 'jsx', 'ts', 'tsx', 'vue', 'py', 'go', 'rs', 'java', 'c', 'cpp', 'h',
  'hpp', 'cs', 'rb', 'php', 'swift', 'kt', 'scala', 'sql', 'sh', 'bash', 'zsh',
  'fish', 'env', 'ini', 'toml', 'cfg', 'conf', 'log', 'csv', 'svg', 'gitignore',
  'dockerfile', 'makefile', 'gradle', 'properties',
])

const LANG_MAP: Record<string, () => Extension> = {
  js: javascript,
  jsx: () => javascript({ jsx: true }),
  ts: () => javascript({ typescript: true }),
  tsx: () => javascript({ jsx: true, typescript: true }),
  json: json,
  py: python,
  md: markdown,
  markdown: markdown,
  yaml: yaml,
  yml: yaml,
  html: html,
  htm: html,
  xml: html,
  css: css,
  vue: html,
  sh: () => javascript(),
  bash: () => javascript(),
  zsh: () => javascript(),
  go: () => javascript(),
}

export function fileExtension(path: string): string {
  const base = path.split('/').pop() || path
  const dot = base.lastIndexOf('.')
  if (dot <= 0) return ''
  return base.slice(dot + 1).toLowerCase()
}

export function previewKind(path: string): PreviewKind {
  const ext = fileExtension(path)
  if (BINARY_KINDS[ext]) return BINARY_KINDS[ext]
  if (TEXT_EXTENSIONS.has(ext) || !ext) return 'text'
  return 'unsupported'
}

export function codemirrorLanguage(path: string): Extension | null {
  const ext = fileExtension(path)
  const factory = LANG_MAP[ext]
  return factory ? factory() : null
}

export interface ExcelSheet {
  name: string
  data: string[][]
}
