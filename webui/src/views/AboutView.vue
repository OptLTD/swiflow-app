<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { api } from '../api'
import { useLayoutStore } from '../stores/layout'
import { useToastStore } from '../stores/toast'
import LocalSvgIcon from '../components/LocalSvgIcon.vue'
import type { RuntimeBinary, RuntimeInfo } from '../types'

const layout = useLayoutStore()
const toast = useToastStore()

const version = ref('…')
const loadError = ref('')
const dataDir = ref('')
const opening = ref(false)

const runtime = ref<RuntimeInfo | null>(null)
const runtimeLoading = ref(false)
const runtimeError = ref('')
const installMode = ref<'mainland' | 'standard'>('mainland')
const installing = ref<Record<string, boolean>>({})
let pollTimer: ReturnType<typeof setInterval> | null = null

const pythonReady = computed(() =>
  !!(runtime.value?.python3?.found && runtime.value?.uvx?.found),
)
const nodeReady = computed(() =>
  !!(runtime.value?.node?.found && runtime.value?.npx?.found),
)

onMounted(() => {
  void loadVersion()
  void loadPaths()
  void loadRuntime()
  startPolling()
})

onUnmounted(() => {
  stopPolling()
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

async function loadPaths() {
  try {
    const r = await api.getPaths()
    dataDir.value = r.data_dir || ''
  } catch {
    dataDir.value = ''
  }
}

async function loadRuntime() {
  runtimeLoading.value = true
  runtimeError.value = ''
  try {
    runtime.value = await api.getRuntime()
    if (runtime.value.installing) {
      installing.value = { ...installing.value, ...runtime.value.installing }
    }
    if (pythonReady.value) installing.value['uvx-py'] = false
    if (nodeReady.value) installing.value['js-npx'] = false
  } catch (e: unknown) {
    runtimeError.value = e instanceof Error ? e.message : '检测失败'
    runtime.value = null
  } finally {
    runtimeLoading.value = false
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(() => {
    const busy = installing.value['uvx-py'] || installing.value['js-npx']
      || runtime.value?.installing?.['uvx-py'] || runtime.value?.installing?.['js-npx']
    if (busy) void loadRuntime()
  }, 3000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function onInstall(kind: 'uvx-py' | 'js-npx') {
  installing.value[kind] = true
  try {
    await api.installRuntime(kind, installMode.value)
    toast.success(kind === 'uvx-py'
      ? '已开始安装 Python / UV，请稍候…'
      : '已开始安装 Node.js / npx，请稍候…')
    startPolling()
    await loadRuntime()
  } catch (e: unknown) {
    installing.value[kind] = false
    toast.error(e instanceof Error ? e.message : '启动安装失败')
  }
}

function statusLabel(b?: RuntimeBinary | null) {
  if (!b) return '—'
  if (!b.found) return '未检测到'
  return b.version || '已安装'
}

async function openLogs() {
  opening.value = true
  try {
    await api.openLogFile()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '打开日志失败')
  } finally {
    opening.value = false
  }
}

async function openStorageDir() {
  opening.value = true
  try {
    await api.openDataDir()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : '打开存储目录失败')
  } finally {
    opening.value = false
  }
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

    <!-- Runtime env -->
    <section class="border border-neutral-200 rounded-lg p-4 bg-white space-y-3">
      <div class="flex items-start justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold text-neutral-900">Runtime environment</h2>
          <p class="text-xs text-neutral-500 mt-0.5">
            检测并安装 Python（含 UV）与 Node.js（含 npx）。MCP / 脚本工具常用。
          </p>
        </div>
        <button
          type="button"
          class="shrink-0 inline-flex items-center gap-1.5 h-8 px-3 text-sm border border-neutral-200 rounded bg-white hover:bg-neutral-50 disabled:opacity-50"
          :disabled="runtimeLoading"
          @click="loadRuntime"
        >
          重新检测
        </button>
      </div>

      <div class="flex items-center gap-2 text-xs">
        <span class="text-neutral-500">镜像：</span>
        <button
          type="button"
          class="px-2 py-0.5 rounded border"
          :class="installMode === 'mainland'
            ? 'bg-neutral-800 text-white border-neutral-800'
            : 'border-neutral-200 text-neutral-600'"
          @click="installMode = 'mainland'"
        >国内</button>
        <button
          type="button"
          class="px-2 py-0.5 rounded border"
          :class="installMode === 'standard'
            ? 'bg-neutral-800 text-white border-neutral-800'
            : 'border-neutral-200 text-neutral-600'"
          @click="installMode = 'standard'"
        >官方</button>
      </div>

      <div v-if="runtimeLoading && !runtime" class="text-sm text-neutral-500">检测中…</div>
      <div v-else-if="runtimeError" class="text-sm text-red-600">{{ runtimeError }}</div>
      <div v-else class="space-y-2">
        <div class="border border-neutral-200 rounded p-3 space-y-2">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="font-medium text-sm">Python + UV</div>
              <div class="text-xs mt-1 space-y-0.5">
                <div :class="runtime?.python3?.found ? 'text-green-700' : 'text-amber-700'">
                  Python: {{ statusLabel(runtime?.python3) }}
                </div>
                <div :class="runtime?.uvx?.found ? 'text-green-700' : 'text-amber-700'">
                  uvx: {{ statusLabel(runtime?.uvx) }}
                </div>
              </div>
              <div v-if="runtime?.python3?.path" class="text-xs text-neutral-400 font-mono truncate mt-0.5">
                {{ runtime.python3.path }}
              </div>
            </div>
            <button
              v-if="!pythonReady"
              type="button"
              class="shrink-0 px-2.5 py-1 border rounded text-xs text-neutral-600 hover:bg-neutral-50 disabled:opacity-50"
              :disabled="!!installing['uvx-py']"
              @click="onInstall('uvx-py')"
            >{{ installing['uvx-py'] ? '安装中…' : '安装' }}</button>
            <span v-else class="shrink-0 text-xs text-green-700 px-1">就绪</span>
          </div>
          <p v-if="installing['uvx-py']" class="text-xs text-neutral-500">
            正在后台安装，可能需要几分钟；完成后状态会自动更新。
          </p>
        </div>

        <div class="border border-neutral-200 rounded p-3 space-y-2">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="font-medium text-sm">Node.js + npx</div>
              <div class="text-xs mt-1 space-y-0.5">
                <div :class="runtime?.node?.found ? 'text-green-700' : 'text-amber-700'">
                  Node: {{ statusLabel(runtime?.node) }}
                </div>
                <div :class="runtime?.npx?.found ? 'text-green-700' : 'text-amber-700'">
                  npx: {{ statusLabel(runtime?.npx) }}
                </div>
              </div>
              <div v-if="runtime?.node?.path" class="text-xs text-neutral-400 font-mono truncate mt-0.5">
                {{ runtime.node.path }}
              </div>
            </div>
            <button
              v-if="!nodeReady"
              type="button"
              class="shrink-0 px-2.5 py-1 border rounded text-xs text-neutral-600 hover:bg-neutral-50 disabled:opacity-50"
              :disabled="!!installing['js-npx']"
              @click="onInstall('js-npx')"
            >{{ installing['js-npx'] ? '安装中…' : '安装' }}</button>
            <span v-else class="shrink-0 text-xs text-green-700 px-1">就绪</span>
          </div>
          <p v-if="installing['js-npx']" class="text-xs text-neutral-500">
            正在后台安装，可能需要几分钟；完成后状态会自动更新。
          </p>
        </div>
      </div>
    </section>

    <!-- Logs / storage -->
    <section class="border border-neutral-200 rounded-lg p-4 bg-white space-y-3">
      <div>
        <h2 class="text-sm font-semibold text-neutral-900">Logs & storage</h2>
        <p class="text-xs text-neutral-500 mt-0.5">
          数据库与应用日志保存在存储目录；日志文件为
          <code class="bg-neutral-100 px-1 py-0.5 rounded">swiflow.log</code>。
        </p>
        <p
          v-if="dataDir"
          class="text-xs text-neutral-500 mt-1 break-all font-mono"
        >{{ dataDir }}</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <button
          type="button"
          class="inline-flex items-center gap-1.5 h-8 px-3 text-sm border border-neutral-200 rounded bg-white hover:bg-neutral-50 disabled:opacity-50"
          :disabled="opening"
          @click="openStorageDir"
        >
          <LocalSvgIcon name="folder" :size="14" />
          打开存储目录
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 h-8 px-3 text-sm border border-neutral-200 rounded bg-white hover:bg-neutral-50 disabled:opacity-50"
          :disabled="opening"
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
