<script setup lang="ts">
import { ref, unref, watch } from 'vue'
import { useDraggable } from '@vueuse/core'
import { useAppStore } from '@/stores/app'

const store = useAppStore()

watch(() => store.getContent, (val) => {
  var clz = 'has-content'
  var list1 = unref(menuPanel)?.classList
  var list2 = unref(chatPanel)?.classList
  val ? list1!.add(clz) : list1!.remove(clz)
  val ? list2!.add(clz) : list2!.remove(clz)
})

const mouse = {
  name: '', open: 0,
  init: 0, move: 0,
  width: 0, final: 0
}
const handler = ref<HTMLElement>()
const menuPanel = ref<HTMLElement>()
const chatPanel = ref<HTMLElement>()
useDraggable(handler, {
  onEnd: () => {
    mouse.name = ''
    mouse.open = 0
    mouse.init = 0
    mouse.move = 0
    unref(handler)?.classList.remove('active')
    document.body.classList.remove('dragging')
  },
  onStart: () => {
    mouse.open = 1
    mouse.name = 'chatPanel'
    const chatHtml = unref(chatPanel)
    unref(handler)?.classList.add('active')
    if (chatHtml && mouse.name == 'chatPanel') {
      mouse.width = chatHtml?.offsetWidth
      mouse.final = chatHtml?.offsetWidth
    }
    document.body.classList.add('dragging')
  },
  onMove: (posi: { x: number }) => {
    const bodyWidth = document.body.offsetWidth
    if (mouse.open && mouse.name == 'chatPanel') {
      if (mouse.init == 0 && posi.x > 100) {
        mouse.init = parseFloat(posi.x.toFixed(1))
        return
      }
      mouse.move = mouse.init - parseFloat(posi.x.toFixed(1))
      mouse.final = mouse.width - mouse.move - 10
      if (mouse.final > bodyWidth / 2 || mouse.final < 320) {
        return
      }

      if (mouse.final < 500 && unref(menuPanel)) {
        unref(menuPanel)?.classList.add('mini')
      } else if (mouse.final > 500 && unref(menuPanel)) {
        unref(menuPanel)?.classList.remove('mini')
      }
      window.requestAnimationFrame(() => {
        const ele = unref(chatPanel)
        ele && (ele.style.width = `${mouse.final}px`)
      })
    }
  }
})
</script>

<template>
  <div id="headbar-border"></div>
  <div id="nav-panel" ref="navPanel">
    <slot name="nav" />
  </div>
  <div v-show="store.getMenuBar" id="menu-panel" ref="menuPanel">
    <slot name="menu" />
  </div>
  <div v-show="store.getChatBar" id="chat-panel" ref="chatPanel">
    <div id="chat-handler" ref="handler" />
    <slot name="left" />
  </div>
  <main v-if="store.getContent" id="main-panel">
    <slot name="main" />
  </main>
</template>
