import { defineStore } from 'pinia'

const STORAGE_KEY = 'swiflow_chat_session'

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
  }),
  getters: {
    label: (s) => s.currentTitle || s.currentKey || 'New Chat',
  },
  actions: {
    setSession(key: string, title = '') {
      this.currentKey = key
      this.currentTitle = title
      writeStoredKey(key)
    },
    clearSession() {
      this.currentKey = ''
      this.currentTitle = ''
      writeStoredKey('')
    },
  },
})
