<script setup lang="ts">
import { defineAsyncComponent, ref } from 'vue'
import { useLayoutStore } from '../stores/layout'
import LocalSvgIcon from '../components/LocalSvgIcon.vue'

const AgentsView = defineAsyncComponent(() => import('./AgentsView.vue'))
const SkillsView = defineAsyncComponent(() => import('./SkillsView.vue'))
const ToolsView = defineAsyncComponent(() => import('./ToolsView.vue'))
const MCPServersView = defineAsyncComponent(() => import('./MCPServersView.vue'))
const CronView = defineAsyncComponent(() => import('./CronView.vue'))

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

function openLogs() {
  layout.openFile(LOG_REL_PATH)
}

function openWorkspace() {
  layout.openExplore('.')
}
</script>

<template>
  <div class="h-full flex flex-col min-w-0 bg-white">
    <div class="shrink-0 border-b border-neutral-200 px-4 py-2 flex items-center gap-1 text-sm overflow-x-auto">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        class="px-3 py-1 rounded text-neutral-500 hover:bg-neutral-100 transition-colors shrink-0"
        :class="activeSubTab === tab.key ? 'bg-neutral-100 text-neutral-900 font-medium' : ''"
        @click="activeSubTab = tab.key"
      >{{ tab.label }}</button>
    </div>

    <div class="flex-1 min-h-0 overflow-y-auto overscroll-contain">
      <AgentsView v-if="activeSubTab === 'agents'" />
      <SkillsView v-else-if="activeSubTab === 'skills'" />
      <ToolsView v-else-if="activeSubTab === 'tools'" />
      <MCPServersView v-else-if="activeSubTab === 'mcp'" />
      <CronView v-else-if="activeSubTab === 'cron'" />
      <div v-else-if="activeSubTab === 'system'" class="p-6 max-w-xl space-y-4">
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
      </div>
    </div>
  </div>
</template>
