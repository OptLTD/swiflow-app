<script setup lang="ts">
import { defineAsyncComponent, ref } from 'vue'

const CronView = defineAsyncComponent(() => import('./CronView.vue'))
const ToolsView = defineAsyncComponent(() => import('./ToolsView.vue'))
const AgentView = defineAsyncComponent(() => import('./AgentView.vue'))
const SkillsView = defineAsyncComponent(() => import('./SkillsView.vue'))
const SettingView = defineAsyncComponent(() => import('./SettingView.vue'))
const AboutUsView = defineAsyncComponent(() => import('./AboutView.vue'))
const MCPServersView = defineAsyncComponent(() => import('./MCPServersView.vue'))

const tabs = [
  { key: 'agents', label: 'Agent' },
  { key: 'skills', label: 'Skills' },
  { key: 'tools', label: 'Tools' },
  { key: 'mcp', label: 'MCP' },
  // { key: 'cron', label: 'Cron' },
  { key: 'setting', label: 'Setting' },
  { key: 'aboutus', label: 'About us' },
] as const

type SubTab = (typeof tabs)[number]['key']
const activeSubTab = ref<SubTab>('agents')
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
        <!-- <CronView v-else-if="activeSubTab === 'cron'" /> -->
        <SettingView v-else-if="activeSubTab === 'setting'" />
        <AboutUsView v-else-if="activeSubTab === 'aboutus'" />
      </div>
    </div>
  </div>
</template>
