import { isDesktop } from './desktop'
import { openExternalURL } from './openExternal'

type WailsCallNS = {
  Call?: (options: { methodName: string; args: unknown[] }) => Promise<unknown>
  ByName?: (methodName: string, ...args: unknown[]) => Promise<unknown>
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

/** Open a light app: Wails child window on desktop, system/tab browser otherwise. */
export async function openLightApp(url: string, title = 'Light App'): Promise<void> {
  const trimmed = url.trim()
  if (!/^https?:\/\//i.test(trimmed)) return

  if (isDesktop()) {
    const call = await wailsCallNS()
    if (call) {
      // Wails v3 FQN is "<import path>.<Type>.<Method>" for cmd/desktop package main.
      const methods = [
        'github.com/OptLTD/swiflow/cmd/desktop.LightAppService.OpenWindow',
        'main.LightAppService.OpenWindow',
      ]
      for (const method of methods) {
        try {
          if (typeof call.ByName === 'function') {
            await call.ByName(method, trimmed, title)
            return
          }
          if (typeof call.Call === 'function') {
            await call.Call({ methodName: method, args: [trimmed, title] })
            return
          }
        } catch {
          /* try next name */
        }
      }
    }
  }

  await openExternalURL(trimmed)
}
