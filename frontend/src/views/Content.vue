<script setup lang="ts">
import { getActHtml } from '@/logics/chat'
import { useAppStore } from '@/stores/app'
import { useViewStore } from '@/stores/view'
import { useClipboard } from '@vueuse/core'
import { onMounted, onUpdated, computed, ref } from 'vue'

const app = useAppStore()
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
  const m = await import('mermaid')
  const mermaid = (m as any).default || m
  mermaid.initialize({ startOnLoad: true })
  await mermaid.run()
})
onUpdated(async() => {
  const m = await import('mermaid')
  const mermaid = (m as any).default || m
  await mermaid.run()
})

const close = () => {
  app.toggleContent()
}

// 复制内容到剪贴板
const copyContent = async () => {
  if (!view.getAction) return

  // 根据 app.setup.useCopy 配置决定复制格式
  const useCopy = app.getSetup?.useCopyMode || 'display'
  if (useCopy === 'source') {
    const data = view.getAction
    if (data.result) {
      return await copy(data.result)
    }
    const act = data as Complete
    if (act.content) {
      return await copy(act.content)
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

#main-content:hover .btn-copy{
  display: inline-flex;
}

</style>
