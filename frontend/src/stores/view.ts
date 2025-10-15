import { defineStore } from 'pinia'

export const useViewStore = defineStore('view', {
  state: () => ({
    action: null as MsgAct | null,
    change: null as ChangeMsg | null, // 用于存储变更信息
  }),
  actions: {
    setAction(action: MsgAct | null) {
      this.action = action
    },
    setChange(change: any) {
      this.change = change
    },
    clearChange() {
      this.change = null
    }
  },
  getters: {
    getAction: (state) => {
      return state.action
    },
    getChange: (state) => {
      return state.change
    }
  }
}) 