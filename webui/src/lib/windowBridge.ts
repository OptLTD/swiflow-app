import { useLayoutStore } from '../stores/layout'
import { api } from '../api'
import type { ChatEvent } from '../types'

/** Handle a ui_request SSE event. Returns true if consumed (not a chat message). */
export function handleUiRequest(ev: ChatEvent): boolean {
  if (ev.type !== 'ui_request' || !ev.id || !ev.name) return false
  void fulfillUiRequest(ev)
  return true
}

async function fulfillUiRequest(ev: ChatEvent): Promise<void> {
  const id = ev.id!
  const layout = useLayoutStore()
  try {
    let payload: unknown
    switch (ev.name) {
      case 'window_opened': {
        const files = layout.openFiles.map((t) => ({ path: t.path, title: t.title }))
        payload = { files, count: files.length }
        break
      }
      case 'window_active': {
        const path = layout.activeFilePath
        if (!path) {
          payload = { path: null, reason: 'active tab is not a file' }
        } else {
          payload = { path, title: layout.activeTab.title }
        }
        break
      }
      case 'window_open': {
        const path = typeof ev.arguments?.path === 'string' ? ev.arguments.path : ''
        if (!path) throw new Error('path required')
        layout.openFile(path)
        payload = { opened: true, path }
        break
      }
      default:
        throw new Error(`unknown ui op: ${ev.name}`)
    }
    await api.replyWindow(id, JSON.stringify(payload))
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : 'ui request failed'
    try {
      await api.replyWindow(id, undefined, msg)
    } catch {
      /* ignore reply failures */
    }
  }
}
