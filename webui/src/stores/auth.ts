import { defineStore } from 'pinia'

/** Auth is currently open (no bearer). Kept as a thin store for setup gating. */
export const useAuthStore = defineStore('auth', {
  state: () => ({
    ready: true,
  }),
  getters: {
    isAuthed: () => true,
    needsLogin: () => false,
  },
  actions: {
    async probe() {
      this.ready = true
    },
  },
})
