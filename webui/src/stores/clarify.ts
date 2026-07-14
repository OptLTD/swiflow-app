import { defineStore } from 'pinia'

export interface ClarifyPrompt {
  id: string
  sessionKey: string
  question: string
  options: string[]
  allowFreeText: boolean
}

export const useClarifyStore = defineStore('clarify', {
  state: () => ({
    /** sessionKey → pending clarify prompt */
    bySession: {} as Record<string, ClarifyPrompt>,
  }),
  getters: {
    forSession: (s) => (sessionKey: string) => s.bySession[sessionKey] || null,
  },
  actions: {
    setPending(p: ClarifyPrompt) {
      this.bySession = { ...this.bySession, [p.sessionKey]: p }
    },
    clear(sessionKey: string) {
      if (!this.bySession[sessionKey]) return
      const next = { ...this.bySession }
      delete next[sessionKey]
      this.bySession = next
    },
  },
})
