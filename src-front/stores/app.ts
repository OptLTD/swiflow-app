import { defineStore } from 'pinia'
import { usePreferredDark } from '@vueuse/core';

export const useAppStore = defineStore('app', {
  state: () => ({
    theme: 'auto',
    release: null,
    version: '1.0.0',
    action: 'default',
    layout: 'default',
    chatbar: true,
    menubar: false,
    history: false,
    content: false,
    refresh: false,
    model: '' as string,
    multi: false as boolean,
    authGate: '' as string,
    login: {} as LoginMeta,
    setup: {} as SetupMeta,
    launch: [] as string[],
    active: {} as BotEntity,
    botList: [] as BotEntity[],
    mcpList: [] as McpServer[],
    display: {
      complete: 'inline',
      epigraphText: '',
      showEpigraph: false
    },
    globalUploads: [] as string[]
  }),
  actions: {
    toggleMenuBar() {
      this.menubar = !this.menubar
    },
    toggleChatBar() {
      this.chatbar = !this.chatbar
    },
    toggleHistory() {
      this.history = !this.history
    },
    toggleContent() {
      this.content = !this.content
    },
    setContent(val: boolean) {
      this.content = val
    },
    setChatBar(val: boolean) {
      this.chatbar = val
    },
    setAction(val: string) {
      this.action = val
    },
    setRefresh(val: boolean) {
      this.refresh = val
    },
    setBotList(list: BotEntity[]) {
      this.botList = list
    },
    setMcpList(list: McpServer[]) {
      this.mcpList = list
    },
    setActive(bot: BotEntity) {
      this.active = bot
    },
    setLogin(val: LoginMeta) {
      this.login = val
      if (val && val.username) {
        localStorage.setItem('login', JSON.stringify(val))
      }
    },
    setSetup(data: SetupMeta) {
      this.setup = data
      this.useTheme(data.theme)
      this.useMulti(data.useMulti)
      if (data.version) {
        this.version = data.version
      }
      if (data.authGate) {
        this.authGate = data.authGate
      }
    },
    setLaunch(launch: string[]) {
      this.launch = launch
    },
    setRelease(data: any) {
      this.release = data
    },
    setAuthGate(text: string) {
      this.authGate = text
    },
    useMulti(ok: boolean) {
      this.multi = ok
    },
    useModel(name: string) {
      this.model = name
    },
    useTheme(theme: string) {
      const root = document.documentElement
      if (!theme || theme === 'auto') {
        root.removeAttribute('data-theme')
      } else {
        root.setAttribute('data-theme', theme)
      }
      this.theme = theme
    },
    setEpigraphText(text: string) {
      this.display.epigraphText = text
    },
    setShowEpigraph(show: boolean) {
      this.display.showEpigraph = show
    },
    setGlobalUploads(uploads: string[]) {
      this.globalUploads = uploads
    }
  },
  getters: {
    getLogin: (state) => state.login,
    getSetup: (state) => state.setup,
    getLaunch: (state) => state.launch,
    getActive: (state) => state.active,
    getAction: (state) => state.action,
    getLayout: (state) => state.layout,
    getRefresh: (state) => state.refresh,
    getMenuBar: (state) => state.menubar,
    getChatBar: (state) => state.chatbar,
    getHistory: (state) => state.history,
    getContent: (state) => state.content,
    getDisplay: (state) => state.display,
    getBotList: (state) => state.botList,
    getMcpList: (state) => state.mcpList,
    getIsMulti: (state) => state.multi,
    getUseModel: (state) => state.model,
    getVersion: (state) => state.version,
    getRelease: (state) => state.release,
    getAuthGate: (state) => state.authGate,
    getTheme: (state) => {
      if (state.theme && state.theme != 'auto') {
        return state.theme
      }
      return usePreferredDark().value ? 'dark' : 'light'
    },
    getGlobalUploads: (state) => state.globalUploads
  }
})
