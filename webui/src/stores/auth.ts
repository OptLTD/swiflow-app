import { defineStore } from 'pinia'
import { getToken, setToken, clearToken } from '../api'

export const useAuthStore = defineStore('auth', {
  state: () => ({ token: getToken() }),
  getters: { isAuthed: (s) => !!s.token },
  actions: {
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
