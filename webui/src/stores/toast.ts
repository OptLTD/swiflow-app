import { defineStore } from 'pinia'

export type ToastType = 'success' | 'error'

export interface ToastItem {
  id: number
  message: string
  type: ToastType
}

let nextId = 0

export const useToastStore = defineStore('toast', {
  state: () => ({
    items: [] as ToastItem[],
  }),
  actions: {
    show(message: string, type: ToastType = 'success', ms = 3200) {
      const item: ToastItem = { id: ++nextId, message, type }
      this.items.push(item)
      setTimeout(() => {
        this.items = this.items.filter((t) => t.id !== item.id)
      }, ms)
    },
    success(message: string) {
      this.show(message, 'success')
    },
    error(message: string) {
      this.show(message, 'error', 4500)
    },
  },
})
