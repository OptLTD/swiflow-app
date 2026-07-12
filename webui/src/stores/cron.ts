import { defineStore } from 'pinia'
import { api } from '../api'
import type { CronJob } from '../types'

export const useCronStore = defineStore('cron', {
  state: () => ({
    jobs: [] as CronJob[],
    loaded: false,
  }),
  actions: {
    async load() {
      const r = await api.listCronJobs()
      this.jobs = r.jobs
      this.loaded = true
    },
  },
})
