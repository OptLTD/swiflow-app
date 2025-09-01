<script setup lang="ts">
import mermaid from 'mermaid'
import { request } from '@/support'
import { getActHtml } from '@/logics/chat'
import { useAppStore } from '@/stores/app'
import { useTaskStore } from '@/stores/task'
import { useViewStore } from '@/stores/view'
import { useClipboard } from '@vueuse/core'
import { onMounted, onUpdated, computed, ref } from 'vue'

const app = useAppStore()
const task = useTaskStore()
const view = useViewStore()

// 使用 useClipboard 组合式函数
const { isSupported } = useClipboard()
const { copy, copied } = useClipboard()

// 本地状态跟踪HTML复制
const htmlCopied = ref(false)

// 计算最终的复制状态
const isCopied = computed(() => {
  return copied.value || htmlCopied.value
})

onMounted(async() => {
  mermaid.initialize({
    startOnLoad: true,
  });
  await mermaid.run()
})
onUpdated(async() => {
  await mermaid.run()
})

const close = () => {
  app.toggleContent()
}

const replay = async () => {
  const data = view.getAction
  if (!data) {
    return false
  }
  const uuid = data.msgid
  const chatid = task.getActive
  const url = `/replay?uuid=${uuid}&task=${chatid}`
  const resp = await request.post(url) as any
  console.log('replay', uuid, resp)
}

// 复制内容到剪贴板
const copyContent = async () => {
  if (!view.getAction) return

  // 根据 app.setup.useCopy 配置决定复制格式
  const useCopy = app.getSetup?.useCopy || 'display'

  if (useCopy === 'source') {
    const data = view.getAction
    if (data.result) {
      await copy(data.result)
    }
    return
  }

  // 复制HTML格式（带样式，可直接粘贴到Word等富文本应用）
  const htmlContent = getActHtml(view.getAction)
  if (htmlContent) {
    try {
      // 使用 Clipboard API 复制HTML格式
      const clipboardItem = new ClipboardItem({
        'text/html': new Blob([htmlContent], { type: 'text/html' }),
        'text/plain': new Blob([htmlContent.replace(/<[^>]*>/g, '')], { type: 'text/plain' })
      })
      await navigator.clipboard.write([clipboardItem])
      htmlCopied.value = true
      setTimeout(() => {
        htmlCopied.value = false
      }, 1500)
    } catch (error) {
      const tempDiv = document.createElement('div')
      tempDiv.innerHTML = htmlContent
      const textContent = tempDiv.textContent || tempDiv.innerText || ''
      await copy(textContent)
    }
  }
}

const showReplay = computed(() => {
  if (!view.getAction || 1) {
    return false
  }
  const data = view.getAction
  switch (data?.type) {
    case "use-mcp-tool":
    case "use-self-tool":
    case "execute-command": {
      return true
    }
    case 'file-text-replace':
    case 'file-get-content':
    case 'file-put-content': {
      return true
    }
  }
  return false
})

const result = computed(() => {
  if (!view.getAction) {
    return ''
  }
  return getActHtml(view.getAction)
}) 
</script>

<template>
  <div v-if="view.getAction" id="main-header">
    <button class="btn-back" @click="close">
      <svg class="icon" viewBox="0 0 24 24">
        <path d="M15.41 7.41L14 6l-6 6 6 6 1.41-1.41L10.83 12z"/>
      </svg>
    </button>
    <h3 class="main-title">{{ $t('common.result') }}</h3>
  </div>
  <!-- content -->
  <div v-if="view.getAction" id="main-content">
    <button class="btn-replay" v-if="showReplay" @click="replay">
      REPLAY
    </button>
    <!-- 复制按钮 -->
    <button v-if="isSupported"
      @click="copyContent" class="btn-copy"
      :title="isCopied ? 'copied' : 'copy'"
    >
    {{ isCopied ? 'copied' : 'copy' }}
    </button>
    <div v-html="result" class="rich-text"/>
  </div>
</template>

<style scoped>
#main-content{
  min-width: 0;
  padding: 0px 15px;
  overflow-y: scroll;
  margin-bottom: 15px;
  height: calc(100vh - var(--nav-height));
}
/* #main-header{
  top: -40px;
  position: absolute;
}
#main-header .title{
  font-size: 1.2rem;
  margin: 10px 15px;
} */

#main-content iframe{
  border: 0;
  width: 100%;
  height: 100%;
}

#main-content .btn-replay{
  top: 12px;
  right: 12px;
  padding: 8px 12px;
  display: none;
  position: absolute;
  vertical-align: middle;
  align-items: center;
  flex-direction: row;
}

#main-content .btn-copy{
  top: 1rem;
  right: 1rem;
  padding: 8px 12px;
  display: none;
  position: absolute;
  vertical-align: middle;
  align-items: center;
  flex-direction: row;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: background-color 0.2s;
}
#main-content .btn-copy:hover{
  background-color: var(--bg-menu);
}

#main-content:hover .btn-replay,
#main-content:hover .btn-copy{
  display: inline-flex;
}
#main-content .btn-replay::before{
  content: '';
  width: 15px;
  height: 15px;
  margin-right: 5px;
  display: inline-block;
  background-size: 100% 100%;
  background-repeat: no-repeat;
  background-color: transparent;
  background-position: center center;
  background-image: url("/assets/restart.svg");
}
</style>
