<script setup lang="ts">
import { ref, computed } from 'vue'

const props = defineProps<{ content: string }>()
const open = ref(false) // collapsed by default, like tool blocks

// First non-empty line of the thinking, truncated, shown as a preview in the
// header so you get the gist without expanding.
const firstLine = computed(() => {
  const line = (props.content || '')
    .split('\n')
    .map((l) => l.trim())
    .find((l) => l.length > 0)
  if (!line) return ''
  return line.length > 50 ? line.slice(0, 50) + '…' : line
})
</script>

<template>
  <div class="border border-neutral-200 rounded text-xs mb-0.5">
    <button
      class="w-full px-2 py-1 bg-neutral-50 flex justify-between items-center hover:bg-neutral-100 gap-2"
      @click="open = !open"
    >
      <span class="flex items-center gap-1 truncate">
        <span class="text-neutral-400">{{ open ? '▼' : '▶' }}</span>
        <span class="shrink-0">让我想想</span>
        <span v-if="firstLine" class="text-neutral-400 truncate">· {{ firstLine }}</span>
      </span>
      <span class="shrink-0">💡</span>
    </button>
    <pre
      v-show="open"
      class="p-2 whitespace-pre-wrap max-h-64 overflow-y-auto bg-neutral-50 text-neutral-600 italic"
    >{{ content }}</pre>
  </div>
</template>
