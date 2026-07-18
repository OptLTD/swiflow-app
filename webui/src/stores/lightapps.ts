import { defineStore } from 'pinia'
import { api } from '../api'
import type { LightApp } from '../types'

export const useLightAppsStore = defineStore('lightApps', {
  state: () => ({
    apps: [] as LightApp[],
    loaded: false,
  }),
  actions: {
    async load() {
      const r = await api.listLightApps()
      this.apps = r.apps
      this.loaded = true
    },
  },
})
