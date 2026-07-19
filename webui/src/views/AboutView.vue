<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'
import { useLayoutStore } from '../stores/layout'
import { useToastStore } from '../stores/toast'
import LocalSvgIcon from '../components/LocalSvgIcon.vue'
import type { RuntimeBinary, RuntimeInfo } from '../types'

const { t } = useI18n()
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
    loadError.value = e instanceof Error ? e.message : t('common.failedToLoad')
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
    runtimeError.value = e instanceof Error ? e.message : t('about.detectFailed')
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
      ? t('about.installStartedPy')
      : t('about.installStartedNode'))
    startPolling()
    await loadRuntime()
  } catch (e: unknown) {
    installing.value[kind] = false
    toast.error(e instanceof Error ? e.message : t('about.installStartFailed'))
  }
}

function statusLabel(b?: RuntimeBinary | null) {
  if (!b) return '—'
  if (!b.found) return t('common.notDetected')
  return b.version || t('common.installed')
}

async function openLogs() {
  opening.value = true
  try {
    await api.openLogFile()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : t('about.openLogsFailed'))
  } finally {
    opening.value = false
  }
}

async function openStorageDir() {
  opening.value = true
  try {
    await api.openDataDir()
  } catch (e: unknown) {
    toast.error(e instanceof Error ? e.message : t('about.openStorageFailed'))
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
        <h1 class="text-xl font-bold mb-1">{{ t('about.title') }}</h1>
        <p class="text-sm text-neutral-500">{{ t('about.subtitle') }}</p>
      </div>
    </div>

    <div v-if="loadError" class="text-sm text-amber-700">{{ loadError }}</div>

    <!-- Product -->
    <section class="border border-neutral-200 rounded-lg p-4 bg-white space-y-3">
      <div class="flex items-start justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold text-neutral-900">Swiflow</h2>
          <p class="text-xs text-neutral-500 mt-0.5">{{ t('about.productTagline') }}</p>
        </div>
        <span
          class="shrink-0 inline-flex items-center h-7 px-2.5 rounded bg-neutral-100 text-neutral-700 text-xs font-medium tabular-nums"
        >v{{ version }}</span>
      </div>
      <p class="text-sm text-neutral-700 leading-relaxed">
        {{ t('about.productBody') }}
      </p>
      <ul class="text-sm text-neutral-600 space-y-1 list-disc pl-5">
        <li>{{ t('about.bulletLocal') }}</li>
        <li>{{ t('about.bulletTools') }}</li>
        <li>{{ t('about.bulletDesktop') }}</li>
      </ul>
    </section>

    <!-- Runtime env -->
    <section class="border border-neutral-200 rounded-lg p-4 bg-white space-y-3">
      <div class="flex items-start justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold text-neutral-900">{{ t('about.runtimeTitle') }}</h2>
          <p class="text-xs text-neutral-500 mt-0.5">
            {{ t('about.runtimeHint') }}
          </p>
        </div>
        <button
          type="button"
          class="shrink-0 inline-flex items-center gap-1.5 h-8 px-3 text-sm border border-neutral-200 rounded bg-white hover:bg-neutral-50 disabled:opacity-50"
          :disabled="runtimeLoading"
          @click="loadRuntime"
        >
          {{ t('common.detectAgain') }}
        </button>
      </div>

      <div class="flex items-center gap-2 text-xs">
        <span class="text-neutral-500">{{ t('about.mirror') }}</span>
        <button
          type="button"
          class="px-2 py-0.5 rounded border"
          :class="installMode === 'mainland'
            ? 'bg-neutral-800 text-white border-neutral-800'
            : 'border-neutral-200 text-neutral-600'"
          @click="installMode = 'mainland'"
        >{{ t('about.mainland') }}</button>
        <button
          type="button"
          class="px-2 py-0.5 rounded border"
          :class="installMode === 'standard'
            ? 'bg-neutral-800 text-white border-neutral-800'
            : 'border-neutral-200 text-neutral-600'"
          @click="installMode = 'standard'"
        >{{ t('about.official') }}</button>
      </div>

      <div v-if="runtimeLoading && !runtime" class="text-sm text-neutral-500">{{ t('common.detecting') }}</div>
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
            >{{ installing['uvx-py'] ? t('common.installing') : t('common.install') }}</button>
            <span v-else class="shrink-0 text-xs text-green-700 px-1">{{ t('common.ready') }}</span>
          </div>
          <p v-if="installing['uvx-py']" class="text-xs text-neutral-500">
            {{ t('about.installBackground') }}
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
            >{{ installing['js-npx'] ? t('common.installing') : t('common.install') }}</button>
            <span v-else class="shrink-0 text-xs text-green-700 px-1">{{ t('common.ready') }}</span>
          </div>
          <p v-if="installing['js-npx']" class="text-xs text-neutral-500">
            {{ t('about.installBackground') }}
          </p>
        </div>
      </div>
    </section>

    <!-- Logs / storage -->
    <section class="border border-neutral-200 rounded-lg p-4 bg-white space-y-3">
      <div>
        <h2 class="text-sm font-semibold text-neutral-900">{{ t('about.logsTitle') }}</h2>
        <p class="text-xs text-neutral-500 mt-0.5">
          {{ t('about.logsHint') }}
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
          {{ t('about.openStorage') }}
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 h-8 px-3 text-sm border border-neutral-200 rounded bg-white hover:bg-neutral-50 disabled:opacity-50"
          :disabled="opening"
          @click="openLogs"
        >
          <LocalSvgIcon name="file" :size="14" />
          {{ t('about.openLogs') }}
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 h-8 px-3 text-sm border border-neutral-200 rounded bg-white hover:bg-neutral-50"
          @click="openWorkspace"
        >
          <LocalSvgIcon name="folder" :size="14" />
          {{ t('about.openWorkspace') }}
        </button>
      </div>
    </section>
  </div>
</template>
