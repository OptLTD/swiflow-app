<script setup lang="ts">
import { defineAsyncComponent, onMounted, ref, watch } from 'vue'
import { api } from '../api'
import { useLayoutStore } from '../stores/layout'
import LocalSvgIcon from '../components/LocalSvgIcon.vue'

const CronView = defineAsyncComponent(() => import('./CronView.vue'))
const ToolsView = defineAsyncComponent(() => import('./ToolsView.vue'))
const AgentView = defineAsyncComponent(() => import('./AgentView.vue'))
const SkillsView = defineAsyncComponent(() => import('./SkillsView.vue'))
const MCPServersView = defineAsyncComponent(() => import('./MCPServersView.vue'))

const LOG_REL_PATH = 'swiflow.log'

const tabs = [
  { key: 'agents', label: 'Agent' },
  { key: 'skills', label: 'Skills' },
  { key: 'tools', label: 'Tools' },
  { key: 'mcp', label: 'MCP' },
  { key: 'cron', label: 'Cron' },
  { key: 'system', label: 'System' },
] as const

type SubTab = (typeof tabs)[number]['key']
const activeSubTab = ref<SubTab>('agents')
const layout = useLayoutStore()

const searchProvider = ref('duckduckgo')
const searchBaseURL = ref('')
const searchAPIKey = ref('')
const searchAPIKeySet = ref(false)
const searchSaving = ref(false)
const searchError = ref('')
const searchSaved = ref(false)

const searchProviders = [
  { value: '', label: 'Disabled' },
  { value: 'duckduckgo', label: 'DuckDuckGo' },
  { value: 'brave', label: 'Brave' },
  { value: 'searxng', label: 'SearXNG' },
] as const

async function loadSearchSettings() {
  searchError.value = ''
  try {
    const r = await api.getSearchSettings()
    searchProvider.value = r.provider ?? ''
    searchBaseURL.value = r.base_url ?? ''
    searchAPIKeySet.value = !!r.api_key_set
    searchAPIKey.value = ''
  } catch (e: unknown) {
    searchError.value = e instanceof Error ? e.message : 'failed to load'
  }
}

async function saveSearchSettings() {
  searchSaving.value = true
  searchError.value = ''
  searchSaved.value = false
  try {
    const body: { provider?: string; api_key?: string; base_url?: string } = {
      provider: searchProvider.value,
      base_url: searchBaseURL.value,
    }
    if (searchAPIKey.value.trim()) {
      body.api_key = searchAPIKey.value.trim()
    }
    const r = await api.putSearchSettings(body)
    searchProvider.value = r.provider ?? searchProvider.value
    searchBaseURL.value = r.base_url ?? searchBaseURL.value
    searchAPIKeySet.value = !!r.api_key_set
    searchAPIKey.value = ''
    searchSaved.value = true
  } catch (e: unknown) {
    searchError.value = e instanceof Error ? e.message : 'save failed'
  } finally {
    searchSaving.value = false
  }
}

watch(activeSubTab, (tab) => {
  if (tab === 'system') void loadSearchSettings()
})

onMounted(() => {
  if (activeSubTab.value === 'system') void loadSearchSettings()
})

function openLogs() {
  layout.openFile(LOG_REL_PATH)
}

function openWorkspace() {
  layout.openExplore('.')
}
</script>

<template>
  <div class="h-full flex flex-col min-w-0 bg-white">
    <div class="shrink-0 border-b border-neutral-200">
      <div class="max-w-[960px] mx-auto w-full px-4 py-2 flex items-center gap-1 text-sm overflow-x-auto">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="px-3 py-1 rounded text-neutral-500 hover:bg-neutral-100 transition-colors shrink-0"
          :class="activeSubTab === tab.key ? 'bg-neutral-100 text-neutral-900 font-medium' : ''"
          @click="activeSubTab = tab.key"
        >{{ tab.label }}</button>
      </div>
    </div>

    <div class="flex-1 min-h-0 overflow-y-auto overscroll-contain">
      <div class="max-w-[960px] mx-auto w-full">
        <AgentView v-if="activeSubTab === 'agents'" />
        <ToolsView v-else-if="activeSubTab === 'tools'" />
        <SkillsView v-else-if="activeSubTab === 'skills'" />
        <MCPServersView v-else-if="activeSubTab === 'mcp'" />
        <CronView v-else-if="activeSubTab === 'cron'" />
        <div v-else-if="activeSubTab === 'system'" class="p-6 space-y-8">
          <section class="space-y-3">
            <div>
              <h2 class="text-sm font-semibold text-neutral-900">Web search</h2>
              <p class="text-sm text-neutral-500 mt-1">
                Provider used by the <code class="text-xs bg-neutral-100 px-1 py-0.5 rounded">web_search</code> tool.
              </p>
            </div>
            <label class="block text-sm">
              <span class="text-neutral-600">Provider</span>
              <select
                v-model="searchProvider"
                class="mt-1 w-full border border-neutral-200 rounded px-2 py-1.5 bg-white"
              >
                <option v-for="p in searchProviders" :key="p.value || 'off'" :value="p.value">
                  {{ p.label }}
                </option>
              </select>
            </label>
            <label v-if="searchProvider === 'brave'" class="block text-sm">
              <span class="text-neutral-600">
                Brave API key
                <span v-if="searchAPIKeySet" class="text-neutral-400">(saved; leave blank to keep)</span>
              </span>
              <input
                v-model="searchAPIKey"
                type="password"
                autocomplete="off"
                placeholder="BSA..."
                class="mt-1 w-full border border-neutral-200 rounded px-2 py-1.5"
              />
            </label>
            <label v-if="searchProvider === 'searxng'" class="block text-sm">
              <span class="text-neutral-600">SearXNG base URL</span>
              <input
                v-model="searchBaseURL"
                type="url"
                placeholder="https://searx.example"
                class="mt-1 w-full border border-neutral-200 rounded px-2 py-1.5"
              />
            </label>
            <div class="flex items-center gap-3">
              <button
                type="button"
                class="px-3 py-1.5 text-sm rounded bg-neutral-900 text-white hover:bg-neutral-800 disabled:opacity-50"
                :disabled="searchSaving"
                @click="saveSearchSettings"
              >
                {{ searchSaving ? 'Saving…' : 'Save' }}
              </button>
              <span v-if="searchSaved" class="text-sm text-green-700">Saved</span>
              <span v-if="searchError" class="text-sm text-red-600">{{ searchError }}</span>
            </div>
          </section>

          <section class="space-y-3">
            <div>
              <h2 class="text-sm font-semibold text-neutral-900">Logs</h2>
              <p class="text-sm text-neutral-500 mt-1">
                Application logs are written to
                <code class="text-xs bg-neutral-100 px-1 py-0.5 rounded">{{ LOG_REL_PATH }}</code>
                in the workspace root. You can also open it from Explore.
              </p>
            </div>
            <div class="flex flex-wrap gap-2">
              <button
                type="button"
                class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm border border-neutral-200 rounded hover:bg-neutral-50"
                @click="openLogs"
              >
                <LocalSvgIcon name="file" :size="14" />
                View log file
              </button>
              <button
                type="button"
                class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm border border-neutral-200 rounded hover:bg-neutral-50"
                @click="openWorkspace"
              >
                <LocalSvgIcon name="folder" :size="14" />
                Open workspace
              </button>
            </div>
          </section>
        </div>
      </div>
    </div>
  </div>
</template>
