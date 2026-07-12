import { defineStore } from 'pinia'
import { api } from '../api'
import type { SkillInfo } from '../types'

export const useSkillsStore = defineStore('skills', {
  state: () => ({
    skills: [] as SkillInfo[],
    loaded: false,
  }),
  actions: {
    async load() {
      const r = await api.listSkills()
      this.skills = r.skills
      this.loaded = true
    },
  },
})
