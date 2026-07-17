<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { useLayoutStore } from '../stores/layout'
import LocalSvgIcon from '../components/LocalSvgIcon.vue'

const LOG_REL_PATH = 'swiflow.log'
const layout = useLayoutStore()

const version = ref('…')
const loadError = ref('')

onMounted(() => {
  void loadVersion()
})

async function loadVersion() {
  loadError.value = ''
  try {
    const r = await api.health()
    version.value = r.version || '0.1.0'
  } catch (e: unknown) {
    version.value = '0.1.0'
    loadError.value = e instanceof Error ? e.message : 'failed to load version'
  }
}

function openLogs() {
  layout.openFile(LOG_REL_PATH)
}

function openWorkspace() {
  layout.openExplore('.')
}
</script>

<template>
  <div class="p-6 max-w-[960px] mx-auto space-y-4">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-xl font-bold mb-1">About us</h1>
        <p class="text-sm text-neutral-500">产品介绍、版本与工作区入口。</p>
      </div>
    </div>

    <div v-if="loadError" class="text-sm text-amber-700">{{ loadError }}</div>

    <!-- Product -->
    <section class="border border-neutral-200 rounded-lg p-4 bg-white space-y-3">
      <div class="flex items-start justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold text-neutral-900">Swiflow</h2>
          <p class="text-xs text-neutral-500 mt-0.5">自托管 AI Agent 运行时</p>
        </div>
        <span
          class="shrink-0 inline-flex items-center h-7 px-2.5 rounded bg-neutral-100 text-neutral-700 text-xs font-medium tabular-nums"
        >v{{ version }}</span>
      </div>
      <p class="text-sm text-neutral-700 leading-relaxed">
        Swiflow 是一款面向开发者与小团队的自托管 AI Agent 运行时。单进程即可托管可配置的
        LLM Agent，通过 HTTP + SSE 与客户端交互，并在工作区内安全地使用文件系统、网页访问、
        Shell 与 Skills 等工具，对话历史持久保存。
      </p>
      <ul class="text-sm text-neutral-600 space-y-1 list-disc pl-5">
        <li>本地优先：数据与密钥留在你自己的环境</li>
        <li>工具与 MCP：扩展 Agent 能力，对接外部服务</li>
        <li>桌面与 Web：同一套运行时，便于日常使用与调试</li>
      </ul>
    </section>

    <!-- Logs / workspace -->
    <section class="border border-neutral-200 rounded-lg p-4 bg-white space-y-3">
      <div>
        <h2 class="text-sm font-semibold text-neutral-900">Logs & workspace</h2>
        <p class="text-xs text-neutral-500 mt-0.5">
          应用日志写入工作区根目录
          <code class="bg-neutral-100 px-1 py-0.5 rounded">{{ LOG_REL_PATH }}</code>。
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <button
          type="button"
          class="inline-flex items-center gap-1.5 h-8 px-3 text-sm border border-neutral-200 rounded bg-white hover:bg-neutral-50"
          @click="openLogs"
        >
          <LocalSvgIcon name="file" :size="14" />
          查看日志
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 h-8 px-3 text-sm border border-neutral-200 rounded bg-white hover:bg-neutral-50"
          @click="openWorkspace"
        >
          <LocalSvgIcon name="folder" :size="14" />
          打开工作区
        </button>
      </div>
    </section>
  </div>
</template>
