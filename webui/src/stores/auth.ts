import { defineStore } from 'pinia'

const TOKEN_KEY = 'swiflow_token'

type AuthMode = {
  local_mode: boolean
  auth: boolean
}

type AuthUser = {
  tid: string
  name: string
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    ready: false,
    localMode: true,
    authRequired: false,
    token: (typeof localStorage !== 'undefined' ? localStorage.getItem(TOKEN_KEY) : null) as string | null,
    user: null as AuthUser | null,
    error: '' as string,
  }),
  getters: {
    isAuthed(state): boolean {
      if (!state.ready) return false
      if (state.localMode || !state.authRequired) return true
      return !!state.token && !!state.user
    },
    needsLogin(state): boolean {
      if (!state.ready) return false
      return state.authRequired && !state.localMode && !(state.token && state.user)
    },
  },
  actions: {
    setToken(token: string | null) {
      this.token = token
      if (token) localStorage.setItem(TOKEN_KEY, token)
      else localStorage.removeItem(TOKEN_KEY)
    },
    async probe() {
      this.error = ''
      try {
        const mode = await fetch('/api/auth/mode', { signal: AbortSignal.timeout(10_000) }).then(async (r) => {
          if (!r.ok) throw new Error(`HTTP ${r.status}`)
          return (await r.json()) as AuthMode
        })
        this.localMode = !!mode.local_mode
        this.authRequired = !!mode.auth
        if (this.localMode || !this.authRequired) {
          this.user = { tid: 'default', name: 'local' }
          this.ready = true
          return
        }
        if (!this.token) {
          this.user = null
          this.ready = true
          return
        }
        const me = await fetch('/api/auth/me', {
          headers: { Authorization: `Bearer ${this.token}` },
          signal: AbortSignal.timeout(10_000),
        })
        if (!me.ok) {
          this.setToken(null)
          this.user = null
          this.ready = true
          return
        }
        this.user = (await me.json()) as AuthUser
      } catch (e) {
        this.error = e instanceof Error ? e.message : String(e)
        // Fail open only for local/desktop; otherwise require login.
        if (this.localMode) {
          this.user = { tid: 'default', name: 'local' }
        }
      } finally {
        this.ready = true
      }
    },
    async login(name: string, password: string) {
      this.error = ''
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, password }),
        signal: AbortSignal.timeout(30_000),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) {
        throw new Error((data as { error?: string }).error || `HTTP ${res.status}`)
      }
      const body = data as { token: string; tid: string; name: string }
      this.setToken(body.token)
      this.user = { tid: body.tid, name: body.name }
    },
    async register(name: string, password: string) {
      this.error = ''
      const res = await fetch('/api/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, password }),
        signal: AbortSignal.timeout(30_000),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) {
        throw new Error((data as { error?: string }).error || `HTTP ${res.status}`)
      }
      const body = data as { token: string; tid: string; name: string }
      this.setToken(body.token)
      this.user = { tid: body.tid, name: body.name }
    },
    async logout() {
      try {
        if (this.token) {
          await fetch('/api/auth/logout', {
            method: 'POST',
            headers: { Authorization: `Bearer ${this.token}` },
          })
        }
      } catch {
        /* ignore */
      }
      this.setToken(null)
      this.user = null
    },
  },
})
