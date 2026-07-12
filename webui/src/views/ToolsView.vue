<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useToolsStore } from '../stores/tools'
import { api } from '../api'
import type { ToolInfo } from '../types'

const toolsStore = useToolsStore()
const error = ref('')

const runtimeTools = new Set(['exec'])

const groupLabels: Record<string, string> = {
  fs: 'Filesystem',
  web: 'Web',
  browser: 'Browser',
  runtime: 'Runtime',
  skill: 'Skills',
  schedule: 'Scheduling',
  other: 'Other',
}

const groupOrder = ['fs', 'web', 'browser', 'runtime', 'skill', 'schedule', 'other']

function toolGroup(name: string): string {
  if (runtimeTools.has(name)) return 'runtime'
  if (name === 'browser') return 'browser'
  const idx = name.indexOf('_')
  return idx > 0 ? name.slice(0, idx) : 'other'
}

function toolLocked(name: string): boolean {
  if (name === 'browser') return !toolsStore.browserEnabled
  if (runtimeTools.has(name)) return !toolsStore.execEnabled
  return false
}

const groupedTools = computed(() => {
  const groups = new Map<string, ToolInfo[]>()
  for (const t of toolsStore.tools) {
    const g = toolGroup(t.name)
    const list = groups.get(g) ?? []
    list.push(t)
    groups.set(g, list)
  }
  return groupOrder
    .filter((g) => groups.has(g))
    .map((g) => ({
      key: g,
      label: groupLabels[g] ?? g,
      tools: groups.get(g)!,
    }))
})

onMounted(load)
async function load() {
  try { await toolsStore.load() }
  catch (e: any) { error.value = e.message }
}
async function toggle(name: string, enabled: boolean) {
  try {
    await api.setTool(name, enabled)
    await load()
  } catch (e: any) { error.value = e.message }
}
</script>

<template>
  <div class="p-6 max-w-[960px] mx-auto">
    <h1 class="text-xl font-bold mb-2">Tools</h1>
    <p class="text-sm text-neutral-500 mb-4">Built-in agent tools. MCP tools are managed on the MCP page.</p>
    <div
      v-if="!toolsStore.browserEnabled"
      class="text-sm text-amber-800 bg-amber-50 border border-amber-200 rounded p-3 mb-4"
    >
      Browser tool (<code>browser</code>) is listed but locked.
      Set <code>tools.browser_enabled</code> to <code>true</code> in config (or <code>MIRA_BROWSER=true</code>) and restart the server.
      First run downloads Chromium automatically.
    </div>
    <div
      v-if="!toolsStore.execEnabled"
      class="text-sm text-amber-800 bg-amber-50 border border-amber-200 rounded p-3 mb-4"
    >
      Runtime tool (<code>exec</code>) is listed but locked.
      Set <code>tools.exec_enabled</code> to <code>true</code> in config (or <code>MIRA_EXEC=true</code>) and restart the server to use it.
    </div>
    <div v-if="error" class="text-red-600 mb-2 text-sm">{{ error }}</div>
    <div class="space-y-6">
      <section v-for="group in groupedTools" :key="group.key">
        <h2 class="text-sm font-semibold text-neutral-700 mb-2">{{ group.label }}</h2>
        <div class="space-y-2">
          <div
            v-for="t in group.tools"
            :key="t.name"
            class="border border-neutral-200 rounded p-3 bg-white flex justify-between items-center"
          >
            <div>
              <div class="font-mono text-sm">{{ t.name }}</div>
              <div class="text-xs text-neutral-500">{{ t.description }}</div>
            </div>
            <label class="text-sm flex items-center gap-2">
              <input
                type="checkbox"
                :checked="t.enabled"
                :disabled="toolLocked(t.name)"
            @change="toggle(t.name, ($event.target as HTMLInputElement).checked)"
              />
              enabled
            </label>
          </div>
        </div>
      </section>
      <div v-if="!toolsStore.tools.length" class="text-neutral-500 text-sm">No built-in tools loaded.</div>
    </div>
  </div>
</template>
