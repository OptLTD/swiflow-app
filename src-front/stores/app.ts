import { defineStore } from 'pinia'
import { usePreferredDark } from '@vueuse/core';

export const useAppStore = defineStore('app', {
  state: () => ({
    theme: 'auto',
    loaded: false,
    release: null,
    version: '1.0.0',
    action: 'default',
    layout: 'default',
    inDocker: 'no',
    chatbar: true,
    menubar: false,
    history: false,
    content: false,
    refresh: false,
    model: '' as string,
    login: {} as LoginMeta,
    setup: {} as SetupMeta,
    mcpEnv: {} as McpEnvMeta,
    active: {} as BotEntity,
    botList: [] as BotEntity[],
    mcpList: [] as McpServer[],
    display: {
      complete: 'inline',
      epigraphText: '',
      showEpigraph: false
    },
    uploads: [] as string[],
    gateway: '' as string
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
    setLoaded(loaded: boolean) {
      this.loaded = loaded
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
      this.useTheme(data.useTheme)
      if (data.authGateway) {
        this.setAuthGate(data.authGateway)
      }
    },
    setRelease(data: any) {
      this.release = data
    },
    setAuthGate(text: string) {
      this.gateway = text
    },
    setInDocker(val: string) {
      this.inDocker = val
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
    setUploads(uploads: string[]) {
      this.uploads = uploads
    },
    setMcpEnv(env: McpEnvMeta) {
      this.mcpEnv = env
    },
  },
  getters: {
    getLogin: (state) => state.login,
    getSetup: (state) => state.setup,
    getMcpEnv: (state) => state.mcpEnv,
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
    getUseModel: (state) => state.model,
    getVersion: (state) => state.version,
    getRelease: (state) => state.release,
    getInDocker: (state) => state.inDocker,
    authGateway: (state) => state.gateway,
    useIsolated: (state) => state.setup.useIsolated,
    useSubAgent: (state) => state.setup.useSubAgent,
    useDebugMode: (state) => state.setup.useDebugMode,
    getTheme: (state) => {
      if (state.theme && state.theme != 'auto') {
        return state.theme
      }
      return usePreferredDark().value ? 'dark' : 'light'
    },
    getLoaded: (state) => state.loaded,
    getUploads: (state) => state.uploads,
  }
})
