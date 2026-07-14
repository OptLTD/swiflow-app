<script setup lang="ts">
import { useToastStore } from '../stores/toast'

const toast = useToastStore()
</script>

<template>
  <div class="fixed bottom-4 right-4 z-[100] flex flex-col gap-2 pointer-events-none max-w-sm">
    <TransitionGroup name="toast">
      <div
        v-for="item in toast.items"
        :key="item.id"
        class="px-4 py-2.5 rounded-lg shadow-lg text-sm leading-snug"
        :class="item.type === 'success' ? 'bg-neutral-900 text-white' : 'bg-red-600 text-white'"
      >
        {{ item.message }}
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.2s ease;
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
