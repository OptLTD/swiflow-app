import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js/lib/core'
import jsonLang from 'highlight.js/lib/languages/json'
import python from 'highlight.js/lib/languages/python'
import bash from 'highlight.js/lib/languages/bash'
import javascript from 'highlight.js/lib/languages/javascript'
import typescript from 'highlight.js/lib/languages/typescript'
import go from 'highlight.js/lib/languages/go'
import xml from 'highlight.js/lib/languages/xml'
import css from 'highlight.js/lib/languages/css'
import yaml from 'highlight.js/lib/languages/yaml'
import markdown from 'highlight.js/lib/languages/markdown'

hljs.registerLanguage('md', markdown)
hljs.registerLanguage('markdown', markdown)
hljs.registerLanguage('python', python)
hljs.registerLanguage('json', jsonLang)
hljs.registerLanguage('shell', bash)
hljs.registerLanguage('bash', bash)
hljs.registerLanguage('sh', bash)
hljs.registerLanguage('go', go)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('html', xml)
hljs.registerLanguage('css', css)
hljs.registerLanguage('yml', yaml)
hljs.registerLanguage('yaml', yaml)
hljs.registerLanguage('js', javascript)
hljs.registerLanguage('ts', typescript)
hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('typescript', typescript)

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  typographer: true,
})

/** Render markdown to HTML with syntax-highlighted code and wrapped tables. */
export function renderMarkdown(content: string): string {
  const html = md.render(content || '')
  const el = document.createElement('div')
  el.innerHTML = html
  el.querySelectorAll('pre code').forEach((node) => {
    hljs.highlightElement(node as HTMLElement)
  })
  el.querySelectorAll('table').forEach((table) => {
    if (table.parentElement?.classList.contains('prose-table-wrap')) return
    const wrap = document.createElement('div')
    wrap.className = 'prose-table-wrap'
    table.parentNode?.insertBefore(wrap, table)
    wrap.appendChild(table)
  })
  el.querySelectorAll('a[href]').forEach((node) => {
    const a = node as HTMLAnchorElement
    a.setAttribute('target', '_blank')
    a.setAttribute('rel', 'noopener noreferrer')
  })
  return el.innerHTML
}
