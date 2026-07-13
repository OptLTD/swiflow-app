<script setup lang="ts">
import { ref } from 'vue'
import { useLayoutStore } from '../stores/layout'

const layout = useLayoutStore()
const dragging = ref(false)

function onDown(e: MouseEvent) {
  e.preventDefault()
  dragging.value = true
  const startX = e.clientX
  const startWidth = layout.chatPanelWidth

  function onMove(ev: MouseEvent) {
    const delta = startX - ev.clientX
    layout.setChatPanelWidth(startWidth + delta)
  }
  function onUp() {
    dragging.value = false
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}
</script>

<template>
  <!-- Visual 1px border; hit area expands on hover -->
  <div
    class="resize-handle relative shrink-0 w-px group"
    :class="dragging ? 'bg-blue-400' : 'bg-neutral-200'"
    @mousedown="onDown"
  >
    <div
      class="absolute inset-y-0 -left-1 w-2 cursor-col-resize z-10"
      :class="dragging ? 'bg-blue-400/20' : 'hover:bg-blue-400/15'"
    />
  </div>
</template>
