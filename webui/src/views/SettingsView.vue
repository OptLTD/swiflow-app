<script setup lang="ts">
import { ref } from 'vue'
import AgentsView from './AgentsView.vue'
import ProvidersView from './ProvidersView.vue'
import SkillsView from './SkillsView.vue'
import ToolsView from './ToolsView.vue'
import MCPServersView from './MCPServersView.vue'
import CronView from './CronView.vue'

const tabs = [
  { key: 'agents', label: 'Agents' },
  { key: 'providers', label: 'Providers' },
  { key: 'skills', label: 'Skills' },
  { key: 'tools', label: 'Tools' },
  { key: 'mcp', label: 'MCP' },
  { key: 'cron', label: 'Cron' },
] as const

type SubTab = (typeof tabs)[number]['key']
const activeSubTab = ref<SubTab>('agents')
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
      <ProvidersView v-else-if="activeSubTab === 'providers'" />
      <SkillsView v-else-if="activeSubTab === 'skills'" />
      <ToolsView v-else-if="activeSubTab === 'tools'" />
      <MCPServersView v-else-if="activeSubTab === 'mcp'" />
      <CronView v-else-if="activeSubTab === 'cron'" />
    </div>
  </div>
</template>
