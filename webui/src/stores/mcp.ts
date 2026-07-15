import { defineStore } from 'pinia'
import { api } from '../api'
import type { MCPServer } from '../types'

export const useMCPStore = defineStore('mcp', {
  state: () => ({
    servers: [] as MCPServer[],
    loaded: false,
  }),
  actions: {
    async load() {
      const r = await api.listMCPServers()
      this.servers = r.servers ?? []
      this.loaded = true
    },
  },
})
