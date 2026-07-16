export function isDesktop(): boolean {
  if (typeof window === 'undefined') return false
  // macOS: wails://localhost  |  Windows: http://wails.localhost
  return (
    window.location.protocol === 'wails:' ||
    window.location.hostname === 'wails.localhost' ||
    !!window._wails
  )
}

export function isWindowsDesktop(): boolean {
  if (!isDesktop() || typeof navigator === 'undefined') return false
  return /Win/i.test(navigator.platform) || /Windows/i.test(navigator.userAgent)
}

export function isMacDesktop(): boolean {
  if (!isDesktop() || typeof navigator === 'undefined') return false
  return /Mac/i.test(navigator.platform) || /Mac OS/i.test(navigator.userAgent)
}

interface WailsWindow {
  ToggleMaximise?: () => Promise<void>
  Minimise?: () => Promise<void>
  Close?: () => Promise<void>
}

interface WailsRuntime {
  Window?: WailsWindow
}

declare global {
  interface Window {
    wails?: WailsRuntime
    _wails?: Record<string, unknown>
  }
}

export function toggleMaximize(): void {
  void window.wails?.Window?.ToggleMaximise?.()
}

export function minimiseWindow(): void {
  void window.wails?.Window?.Minimise?.()
}

export function closeWindow(): void {
  void window.wails?.Window?.Close?.()
}
