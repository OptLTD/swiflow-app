import { defineStore } from 'pinia'

export interface Tab {
  id: string
  type: 'welcome' | 'file' | 'explore' | 'settings'
  title: string
  path?: string
  closable: boolean
}

const WELCOME_TAB: Tab = { id: 'welcome', type: 'welcome', title: 'Home', closable: false }

export const useLayoutStore = defineStore('layout', {
  state: () => ({
    chatPanelOpen: true,
    chatPanelWidth: 380,
    tabs: [WELCOME_TAB] as Tab[],
    activeTabId: 'welcome',
    explorePath: '.',
  }),
  getters: {
    activeTab: (s) => s.tabs.find((t) => t.id === s.activeTabId) || WELCOME_TAB,
  },
  actions: {
    toggleChatPanel() {
      this.chatPanelOpen = !this.chatPanelOpen
    },
    setChatPanelWidth(w: number) {
      this.chatPanelWidth = Math.max(280, Math.min(w, 800))
    },

    activateTab(id: string) {
      if (this.tabs.some((t) => t.id === id)) {
        this.activeTabId = id
      }
    },

    openTab(tab: Tab) {
      const existing = this.tabs.find((t) => t.id === tab.id)
      if (existing) {
        this.activeTabId = tab.id
        return
      }
      this.tabs.push(tab)
      this.activeTabId = tab.id
    },

    closeTab(id: string) {
      const tab = this.tabs.find((t) => t.id === id)
      if (!tab || !tab.closable) return
      const idx = this.tabs.indexOf(tab)
      this.tabs.splice(idx, 1)
      if (this.activeTabId === id) {
        // Activate the nearest tab
        const next = this.tabs[Math.min(idx, this.tabs.length - 1)]
        this.activeTabId = next ? next.id : 'welcome'
      }
    },

    openFile(path: string) {
      const name = path.split('/').pop() || path
      this.openTab({
        id: 'file:' + path,
        type: 'file',
        title: name,
        path,
        closable: true,
      })
    },

    openExplore(path = '.') {
      const existing = this.tabs.find((t) => t.id === 'explore')
      if (existing) {
        existing.path = path
        this.activeTabId = 'explore'
        return
      }
      this.openTab({
        id: 'explore',
        type: 'explore',
        title: 'Explore',
        path,
        closable: true,
      })
    },

    openSettings() {
      this.openTab({
        id: 'settings',
        type: 'settings',
        title: 'Settings',
        closable: true,
      })
    },

    setExplorePath(path: string) {
      this.explorePath = path || '.'
    },
  },
})
