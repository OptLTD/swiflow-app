import { defineStore } from 'pinia'

export interface Tab {
  id: string
  type: 'home' | 'file' | 'explore' | 'settings' | 'chat'
  title: string
  /** File path, explore path, or chat session key. */
  path?: string
  closable: boolean
}

const HOME_TAB: Tab = {
  id: 'home',
  type: 'home',
  title: 'Home',
  closable: false,
}

const EXPLORE_TAB: Tab = {
  id: 'explore',
  type: 'explore',
  title: 'Explore',
  path: '.',
  closable: false,
}

function chatTabId(sessionKey: string) {
  return 'chat:' + sessionKey
}

export const useLayoutStore = defineStore('layout', {
  state: () => ({
    tabs: [HOME_TAB, EXPLORE_TAB] as Tab[],
    explorePath: '.',
    activeTabId: 'home',
    chatPanelOpen: false,
    chatPanelWidth: 380,
  }),
  getters: {
    activeTab: (s) => s.tabs.find((t) => t.id === s.activeTabId) || HOME_TAB,
    isChatTabActive: (s) => {
      const t = s.tabs.find((tab) => tab.id === s.activeTabId)
      return t?.type === 'chat'
    },
    /** Sidebar chat is hidden while any Chat tab is focused. */
    showChatSidebar: (s) => {
      const t = s.tabs.find((tab) => tab.id === s.activeTabId)
      return s.chatPanelOpen && t?.type !== 'chat'
    },
    chatTabs: (s) => s.tabs.filter((t) => t.type === 'chat' && !!t.path),
    openFiles: (s) => s.tabs.filter((t) => t.type === 'file' && !!t.path),
    activeFilePath: (s) => {
      const t = s.tabs.find((tab) => tab.id === s.activeTabId)
      return t?.type === 'file' && t.path ? t.path : null
    },
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
        const next = this.tabs[Math.min(idx, this.tabs.length - 1)]
        this.activeTabId = next ? next.id : 'home'
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
        this.explorePath = path
        this.activeTabId = 'explore'
        return
      }
      // Fixed tab should already exist; recreate if somehow removed.
      this.tabs.splice(1, 0, { ...EXPLORE_TAB, path })
      this.explorePath = path
      this.activeTabId = 'explore'
    },

    openSettings() {
      this.openTab({
        id: 'settings',
        type: 'settings',
        title: 'Settings',
        closable: true,
      })
    },

    /** Open a chat session as a main tab. Same path (session key) reuses the existing tab. */
    openChatTab(sessionKey: string, title = '') {
      if (!sessionKey) return
      const id = chatTabId(sessionKey)
      const existing = this.tabs.find((t) => t.id === id)
      if (existing) {
        if (title) existing.title = title
        this.activeTabId = id
        return
      }
      this.openTab({
        id,
        type: 'chat',
        title: title || 'New Chat',
        path: sessionKey,
        closable: true,
      })
    },

    renameChatTab(sessionKey: string, title: string) {
      if (!sessionKey || !title) return
      const tab = this.tabs.find((t) => t.id === chatTabId(sessionKey))
      if (tab) tab.title = title
    },

    /** Close one maximized chat tab (default: active) and ensure sidebar can show. */
    exitChatTab(sessionKey?: string) {
      const tab = sessionKey
        ? this.tabs.find((t) => t.type === 'chat' && t.path === sessionKey)
        : this.tabs.find((t) => t.id === this.activeTabId && t.type === 'chat')
      if (tab) this.closeTab(tab.id)
      this.chatPanelOpen = true
    },

    setExplorePath(path: string) {
      this.explorePath = path || '.'
    },
  },
})
