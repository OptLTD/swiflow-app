export function isDesktop(): boolean {
  if (typeof window === 'undefined') return false
  // macOS: wails://localhost  |  Windows: http://wails.localhost
  return (
    window.location.protocol === 'wails:' ||
    window.location.hostname === 'wails.localhost' ||
    !!window._wails
  )
}

interface WailsWindow {
  ToggleMaximise?: () => Promise<void>
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
