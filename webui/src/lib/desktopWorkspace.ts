import { isDesktop } from './desktop'

export interface WorkspaceBinaryPayload {
  path: string
  encoding: string
  content: string
  size: number
}

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

export async function desktopDownloadWorkspaceFile(path: string): Promise<WorkspaceBinaryPayload | null> {
  const call = await wailsCallNS()
  if (!call) return null
  try {
    const method = 'main.WorkspaceService.DownloadFile'
    let result: unknown
    if (typeof call.ByName === 'function') {
      result = await call.ByName(method, path)
    } else if (typeof call.Call === 'function') {
      result = await call.Call({ methodName: method, args: [path] })
    } else {
      return null
    }
    const payload = result as WorkspaceBinaryPayload | null
    if (!payload) return null
    return payload
  } catch {
    return null
  }
}
