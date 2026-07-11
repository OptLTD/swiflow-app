import { defineStore } from 'pinia'

// Shared chat-UI state so the top nav (☰ + session label) and the ChatView
// drawer stay in sync.
export const useChatStore = defineStore('chat', {
  state: () => ({
    drawerOpen: false,
    currentKey: '',
    currentTitle: '',
  }),
  getters: {
    label: (s) => s.currentTitle || s.currentKey || 'new chat',
  },
  actions: {
    toggleDrawer() {
      this.drawerOpen = !this.drawerOpen
    },
    closeDrawer() {
      this.drawerOpen = false
    },
    setSession(key: string, title: string) {
      this.currentKey = key
      this.currentTitle = title
    },
  },
})
