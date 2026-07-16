<script setup lang="ts">
import { computed } from 'vue'
import { fromAtPath, parseAtPaths } from '../lib/workspacePath'

const props = defineProps<{ content: string }>()
const emit = defineEmits<{ open: [atPath: string] }>()

const refs = computed(() => parseAtPaths(props.content))
const shown = computed(() => refs.value.slice(0, 2))
const more = computed(() => Math.max(0, refs.value.length - 2))

function shortLabel(at: string): string {
  const name = fromAtPath(at).split('/').pop() || at
  return name.length > 22 ? name.slice(0, 10) + '…' + name.slice(-8) : name
}
</script>

<template>
  <div
    v-if="refs.length"
    class="w-full max-w-full flex items-center justify-end gap-1.5 min-w-0 overflow-hidden"
  >
    <button
      v-for="at in shown"
      :key="at"
      type="button"
      class="shrink min-w-0 max-w-[42%] inline-flex items-center px-2 py-0.5 rounded-md text-xs bg-neutral-100 text-neutral-700 hover:bg-neutral-200 font-mono truncate"
      :title="at"
      @click="emit('open', at)"
    >{{ shortLabel(at) }}</button>
    <span
      v-if="more > 0"
      class="shrink-0 text-xs text-neutral-500 tabular-nums"
      :title="refs.join('\n')"
    >+{{ more }} · {{ refs.length }} files</span>
    <span
      v-else-if="refs.length > 1"
      class="shrink-0 text-xs text-neutral-500 tabular-nums"
    >{{ refs.length }} files</span>
  </div>
</template>
