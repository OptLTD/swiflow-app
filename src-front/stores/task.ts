import { defineStore } from 'pinia'

export const useTaskStore = defineStore('task', {
  state: () => ({
    active: '' as string,
    history: [] as TaskEntity[],
  }),
  actions: {
    setActive(active: string) {
      this.active = active
    },
    setHistory(data: TaskEntity[] = []) {
      this.history = data
    },
  },
  getters: {
    getActive: (state) => {
      return state.active
    },
    getHistory: (state) => {
      return state.history
    },
    getCurrent: (state) => {
      if (!state.active) {
        return null
      }
      return state.history.find((x: TaskEntity) => {
        return x.uuid === state.active
      })
    },
    getRunning: (state) => {
      return (bot: string) => {
        const runningStates = ['running', 'process', 'failed']
        return state.history.find((task: TaskEntity) => {
          const taskBot = task.bots?.[0] || task.bots
          return taskBot === bot && runningStates.includes(task.state)
        }) || null
      }
    },
  }
})
