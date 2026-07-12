import { defineStore } from 'pinia'
import { api } from '../api'
import type { ToolInfo } from '../types'

export const useToolsStore = defineStore('tools', {
  state: () => ({
    tools: [] as ToolInfo[],
    loaded: false,
  }),
  actions: {
    async load() {
      const r = await api.listTools()
      this.tools = r.tools
      this.loaded = true
    },
  },
})
