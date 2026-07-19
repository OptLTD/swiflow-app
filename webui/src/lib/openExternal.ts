import { api } from '../api'
import { isDesktop } from './desktop'

/** Open http(s) URL outside the app shell. */
export async function openExternalURL(raw: string): Promise<void> {
  const url = raw.trim()
  if (!/^https?:\/\//i.test(url)) return

  // Desktop (Wails): open OS default browser via backend.
  // Docker / plain web: never call /api/system/act open-url — that would run
  // browser.OpenURL on the server host. Use the client tab instead.
  if (!isDesktop()) {
    window.open(url, '_blank', 'noopener,noreferrer')
    return
  }
  try {
    await api.openURL(url)
  } catch {
    window.open(url, '_blank', 'noopener,noreferrer')
  }
}

/**
 * Capture-phase click handler for markdown / prose content.
 * Desktop: intercept so the WebView never navigates; open system browser.
 * Web/Docker: leave default (markdown already sets target=_blank).
 */
export function onProseLinkClick(e: MouseEvent): void {
  if (!isDesktop()) return
  if (e.defaultPrevented || e.button !== 0) return
  if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return
  const el = e.target
  if (!(el instanceof Element)) return
  const prose = el.closest('.prose-swiflow')
  if (!prose) return
  const a = el.closest('a[href]')
  if (!(a instanceof HTMLAnchorElement) || !prose.contains(a)) return
  const href = (a.getAttribute('href') || '').trim()
  if (!/^https?:\/\//i.test(href)) return
  e.preventDefault()
  e.stopPropagation()
  void openExternalURL(href)
}

/** Install once at app boot. */
export function installProseExternalLinks(): void {
  document.addEventListener('click', onProseLinkClick, true)
}
