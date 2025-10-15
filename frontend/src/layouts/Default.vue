<script setup lang="ts">
import { ref, unref, watch, onMounted } from 'vue'
import { useDraggable } from '@vueuse/core'
import { useAppStore } from '@/stores/app'
import { Window } from "@wailsio/runtime";

const app = useAppStore()

watch(() => app.getContent, (val) => {
  var clz = 'has-content'
  var list1 = unref(menuPanel)?.classList
  var list2 = unref(chatPanel)?.classList
  val ? list1!.add(clz) : list1!.remove(clz)
  val ? list2!.add(clz) : list2!.remove(clz)
})

onMounted(() => {
  // Events.On('time', (timeValue: { data: string }) => {
  //   console.log('time', timeValue.data)
  // });
  const header = unref(topHeader)
  if (header && Window.Get('Swiflow')) {
    header.addEventListener('dblclick', async () => {
      if (await Window.IsMaximised()) {
        await Window.UnMaximise()
      } else {
        await Window.Maximise()
      }
    })
  }
})

const mouse = {
  name: '', open: 0,
  init: 0, move: 0,
  width: 0, final: 0
}
const handler = ref<HTMLElement>()
const menuPanel = ref<HTMLElement>()
const chatPanel = ref<HTMLElement>()
const topHeader = ref<HTMLElement>()
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
  <div id="top-header" ref="topHeader"/>
  <div id="nav-panel" ref="navPanel">
    <slot name="nav" />
  </div>
  <div v-show="app.getMenuBar" id="menu-panel" ref="menuPanel">
    <slot name="menu" />
  </div>
  <div v-show="app.getChatBar" id="chat-panel" ref="chatPanel">
    <div id="chat-handler" ref="handler" />
    <slot name="left" />
  </div>
  <main v-if="app.getContent" id="main-panel">
    <slot name="main" />
  </main>
</template>

<style scoped>
#top-header {
  top: 0;
  left: 0;
  width: 100%;
  cursor: auto;
  position: fixed;
  /* z-index: 100; */
  --wails-draggable:drag;
  box-sizing: border-box;
  height: var(--nav-height);
  background: var(--bg-light);
  border-bottom: 1px solid var(--bg-menu);

  user-select: none;
  -webkit-user-select: none;
  -moz-user-select: none;
  -ms-user-select: none;
}
</style>
