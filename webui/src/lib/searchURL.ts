/** Build a human-facing search-results page URL for the configured provider. */

export function searchProviderPageURL(provider: string, baseURL: string, query: string): string {
  const q = query.trim()
  if (!q) return ''
  const enc = encodeURIComponent(q)
  switch ((provider || '').toLowerCase().trim()) {
    case 'brave':
      return `https://search.brave.com/search?q=${enc}`
    case 'searxng':
    case 'searx': {
      const base = (baseURL || '').trim().replace(/\/+$/, '')
      return base ? `${base}/search?q=${enc}` : ''
    }
    case 'duckduckgo':
    case 'ddg':
      return `https://duckduckgo.com/?q=${enc}`
    default:
      // Unknown / disabled → DuckDuckGo (matches serve default for web_search).
      return `https://duckduckgo.com/?q=${enc}`
  }
}
