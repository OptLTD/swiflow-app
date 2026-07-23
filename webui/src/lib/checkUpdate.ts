import { isDesktop } from './desktop'

type WailsCallNS = {
  Call?: (options: { methodName: string; args: unknown[] }) => Promise<unknown>
  ByName?: (methodName: string, ...args: unknown[]) => Promise<unknown>
}

export type UpdateCheckResult = {
  available: boolean
  current: string
  latest?: string
  name?: string
  notes?: string
  error?: string
}

async function wailsCallNS(): Promise<WailsCallNS | null> {
  if (!isDesktop()) return null
  try {
    const loader = new Function('return import("/wails/runtime.js")') as () => Promise<{ Call?: WailsCallNS }>
    const mod = await loader()
    return mod.Call ?? null
  } catch {
    return null
  }
}

async function callMethod(method: string, args: unknown[] = []): Promise<unknown> {
  const call = await wailsCallNS()
  if (!call) return undefined
  const methods = [
    `github.com/OptLTD/swiflow/cmd/desktop.${method}`,
    `main.${method}`,
  ]
  for (const name of methods) {
    try {
      if (typeof call.ByName === 'function') {
        return await call.ByName(name, ...args)
      }
      if (typeof call.Call === 'function') {
        return await call.Call({ methodName: name, args })
      }
    } catch {
      /* try next */
    }
  }
  return undefined
}

type WailsEventsNS = {
  On?: (name: string, cb: (e: { data?: unknown } | unknown) => void) => (() => void) | void
}

/** Subscribe to a Wails runtime event; returns an unsubscribe fn. */
export async function onWailsEvent(
  name: string,
  cb: (data: unknown) => void,
): Promise<() => void> {
  if (!isDesktop()) return () => {}
  try {
    const loader = new Function('return import("/wails/runtime.js")') as () => Promise<{
      Events?: WailsEventsNS
    }>
    const mod = await loader()
    const on = mod.Events?.On
    if (typeof on !== 'function') return () => {}
    const off = on(name, (e) => {
      const payload = e && typeof e === 'object' && 'data' in e
        ? (e as { data?: unknown }).data
        : e
      cb(payload)
    })
    return typeof off === 'function' ? off : () => {}
  } catch {
    return () => {}
  }
}

/** Silent check — no update window. */
export async function detectDesktopUpdate(): Promise<UpdateCheckResult | null> {
  if (!isDesktop()) return null
  const raw = await callMethod('UpdateService.CheckLatest')
  if (!raw || typeof raw !== 'object') return null
  const r = raw as Record<string, unknown>
  return {
    available: !!r.available,
    current: String(r.current ?? ''),
    latest: r.latest != null ? String(r.latest) : undefined,
    name: r.name != null ? String(r.name) : undefined,
    notes: r.notes != null ? String(r.notes) : undefined,
    error: r.error != null ? String(r.error) : undefined,
  }
}

/** Open the native Wails update dialog / install flow. */
export async function openDesktopUpdateDialog(): Promise<boolean> {
  if (!isDesktop()) return false
  const call = await wailsCallNS()
  if (!call) return false

  const methods = [
    'github.com/OptLTD/swiflow/cmd/desktop.UpdateService.CheckForUpdates',
    'main.UpdateService.CheckForUpdates',
  ]
  for (const name of methods) {
    try {
      if (typeof call.ByName === 'function') {
        await call.ByName(name)
        return true
      }
      if (typeof call.Call === 'function') {
        await call.Call({ methodName: name, args: [] })
        return true
      }
    } catch {
      /* try next */
    }
  }
  return false
}

/** @deprecated use openDesktopUpdateDialog */
export async function checkForDesktopUpdates(): Promise<boolean> {
  return openDesktopUpdateDialog()
}

export const UPDATER_EVENTS = {
  updateAvailable: 'wails:updater:update-available',
  noUpdate: 'wails:updater:no-update',
  error: 'wails:updater:error',
} as const
