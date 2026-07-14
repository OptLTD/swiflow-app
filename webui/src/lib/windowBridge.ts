import { useLayoutStore } from '../stores/layout'
import { useClarifyStore } from '../stores/clarify'
import { api } from '../api'
import type { ChatEvent } from '../types'

/** Handle a ui_request SSE event. Returns true if consumed (not a chat message). */
export function handleUiRequest(ev: ChatEvent, sessionKey?: string): boolean {
  if (ev.type !== 'ui_request' || !ev.id || !ev.name) return false
  void fulfillUiRequest(ev, sessionKey)
  return true
}

async function fulfillUiRequest(ev: ChatEvent, sessionKey?: string): Promise<void> {
  const id = ev.id!
  const layout = useLayoutStore()
  try {
    if (ev.name === 'clarify') {
      const args = ev.arguments || {}
      const question = typeof args.question === 'string' ? args.question : ''
      if (!question) throw new Error('question required')
      const options = Array.isArray(args.options)
        ? args.options.filter((o): o is string => typeof o === 'string' && !!o)
        : []
      const allowFreeText = args.allow_free_text !== false
      const key = sessionKey || ''
      if (!key) throw new Error('session required for clarify')
      useClarifyStore().setPending({
        id,
        sessionKey: key,
        question,
        options,
        allowFreeText,
      })
      return // wait for ChatPanel to call api.replyWindow
    }

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

export async function submitClarifyAnswer(
  sessionKey: string,
  answer: string,
): Promise<void> {
  const store = useClarifyStore()
  const pending = store.bySession[sessionKey]
  if (!pending) return
  const text = answer.trim()
  if (!text) return
  store.clear(sessionKey)
  await api.replyWindow(pending.id, JSON.stringify({ answer: text }))
}

export async function cancelClarify(sessionKey: string, reason = 'user cancelled'): Promise<void> {
  const store = useClarifyStore()
  const pending = store.bySession[sessionKey]
  if (!pending) return
  store.clear(sessionKey)
  try {
    await api.replyWindow(pending.id, undefined, reason)
  } catch {
    /* ignore */
  }
}
