import { defineStore } from 'pinia'
import { api } from '../api'
import type { Agent } from '../types'

export const useAgentsStore = defineStore('agents', {
  state: () => ({
    agents: [] as Agent[],
    loaded: false,
  }),
  actions: {
    async load() {
      const r = await api.listAgents()
      this.agents = r.agents ?? []
      this.loaded = true
    },
  },
})
