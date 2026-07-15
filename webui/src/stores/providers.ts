import { defineStore } from 'pinia'
import { api } from '../api'
import type { Provider } from '../types'

export const useProvidersStore = defineStore('providers', {
  state: () => ({
    providers: [] as Provider[],
    loaded: false,
  }),
  actions: {
    async load() {
      const r = await api.listProviders()
      this.providers = r.providers ?? []
      this.loaded = true
    },
  },
})
