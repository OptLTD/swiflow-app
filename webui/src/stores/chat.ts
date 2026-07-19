import { defineStore } from 'pinia'
import { t } from '../i18n'
import { toAtPath } from '../lib/workspacePath'

const STORAGE_KEY = 'swiflow_chat_session'

export interface PendingAttachment {
  name: string
  path: string
  atPath: string
}

function readStoredKey(): string {
  try {
    return localStorage.getItem(STORAGE_KEY) || ''
  } catch {
    return ''
  }
}

function writeStoredKey(key: string) {
  try {
    if (key) localStorage.setItem(STORAGE_KEY, key)
    else localStorage.removeItem(STORAGE_KEY)
  } catch {
    /* ignore quota / private mode */
  }
}

/** Shared chat session selection; persisted across panel toggle and app restart. */
export const useChatStore = defineStore('chat', {
  state: () => ({
    currentKey: readStoredKey(),
    currentTitle: '',
    /** sessionKey → files pending attach on next send */
    pendingBySession: {} as Record<string, PendingAttachment[]>,
    /** One-shot composer text set by Welcome before opening a chat tab. */
    pendingPrompt: '',
  }),
  getters: {
    label: (s) => s.currentTitle || s.currentKey || t('layout.newChat'),
    pendingFor: (s) => (sessionKey: string) => s.pendingBySession[sessionKey] || [],
  },
  actions: {
    setSession(key: string, title = '') {
      this.currentKey = key
      this.currentTitle = title
      writeStoredKey(key)
    },
    setPendingPrompt(text: string) {
      this.pendingPrompt = text.trim()
    },
    consumePendingPrompt(): string {
      const text = this.pendingPrompt
      this.pendingPrompt = ''
      return text
    },
    clearSession() {
      this.currentKey = ''
      this.currentTitle = ''
      writeStoredKey('')
    },
    addPending(sessionKey: string, files: { name: string; path: string }[]) {
      if (!sessionKey || !files.length) return
      const cur = [...(this.pendingBySession[sessionKey] || [])]
      const seen = new Set(cur.map((f) => f.atPath))
      for (const f of files) {
        const atPath = toAtPath(f.path)
        if (seen.has(atPath)) continue
        seen.add(atPath)
        cur.push({ name: f.name, path: f.path, atPath })
      }
      this.pendingBySession = { ...this.pendingBySession, [sessionKey]: cur }
    },
    removePending(sessionKey: string, atPath: string) {
      const cur = this.pendingBySession[sessionKey]
      if (!cur) return
      const next = cur.filter((f) => f.atPath !== atPath)
      if (!next.length) {
        const copy = { ...this.pendingBySession }
        delete copy[sessionKey]
        this.pendingBySession = copy
        return
      }
      this.pendingBySession = { ...this.pendingBySession, [sessionKey]: next }
    },
    clearPending(sessionKey: string) {
      if (!this.pendingBySession[sessionKey]) return
      const copy = { ...this.pendingBySession }
      delete copy[sessionKey]
      this.pendingBySession = copy
    },
  },
})
