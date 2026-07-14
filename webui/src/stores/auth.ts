import { defineStore } from 'pinia'
import { getToken, setToken, clearToken } from '../api'
import { isDesktop } from '../lib/desktop'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: getToken(),
    skipAuth: isDesktop(),
    ready: isDesktop(),
  }),
  getters: {
    isAuthed: (s) => s.skipAuth || !!s.token,
    needsLogin: (s) => s.ready && !s.skipAuth && !s.token,
  },
  actions: {
    async probe() {
      if (this.ready) return
      try {
        const res = await fetch('/api/health')
        if (res.ok) {
          const data = (await res.json()) as { skip_auth?: boolean }
          if (data.skip_auth) this.skipAuth = true
        }
      } catch {
        // backend not ready yet
      } finally {
        this.ready = true
      }
    },
    login(t: string) {
      setToken(t)
      this.token = t
    },
    logout() {
      clearToken()
      this.token = ''
    },
  },
})
