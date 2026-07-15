import { defineStore } from 'pinia'
import { useAgentsStore } from './agents'
import { useProvidersStore } from './providers'
import { DEFAULT_AGENT_KEY } from '../constants/defaults'

export const useSetupStore = defineStore('setup', {
  state: () => ({
    checked: false,
    /** Sticky until complete(); avoids dismantling the wizard mid-flow after agent is created. */
    showWizard: false,
  }),
  getters: {
    needsSetup(): boolean {
      if (!this.checked) return false
      const providers = useProvidersStore().providers ?? []
      const agents = useAgentsStore().agents ?? []
      if (!providers.length) return true
      return !agents.some((a) => a.key === DEFAULT_AGENT_KEY)
    },
  },
  actions: {
    async check() {
      const providers = useProvidersStore()
      const agents = useAgentsStore()
      try {
        await Promise.all([providers.load(), agents.load()])
      } finally {
        this.checked = true
        if (this.needsSetup) this.showWizard = true
      }
    },
    /** Re-load stores and close the wizard only when setup is actually done. */
    async complete() {
      const providers = useProvidersStore()
      const agents = useAgentsStore()
      await Promise.all([providers.load(), agents.load()])
      this.checked = true
      this.showWizard = this.needsSetup
    },
  },
})
