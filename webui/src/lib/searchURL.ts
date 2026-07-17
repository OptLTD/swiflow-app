/** Build a human-facing search-results page URL for the configured provider. */

export function searchProviderPageURL(provider: string, baseURL: string, query: string): string {
  const q = query.trim()
  if (!q) return ''
  const enc = encodeURIComponent(q)
  switch ((provider || '').toLowerCase().trim()) {
    case 'bing':
      return `https://cn.bing.com/search?q=${enc}`
    case 'google':
      return `https://www.google.com/search?q=${enc}`
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
      return `https://cn.bing.com/search?q=${enc}`
  }
}
