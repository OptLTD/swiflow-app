<script setup lang="ts">
import { ref, watch } from 'vue'
import { api } from '../api'
import type { MCPCapabilities, MCPServer } from '../types'

const props = defineProps<{
  server: MCPServer | null
  open: boolean
}>()
const emit = defineEmits<{ close: [] }>()

const loading = ref(false)
const error = ref('')
const caps = ref<MCPCapabilities | null>(null)
const tab = ref<'tools' | 'resources' | 'templates'>('tools')

watch(
  () => [props.open, props.server?.id] as const,
  async ([open, id]) => {
    if (!open || !id) {
      caps.value = null
      error.value = ''
      tab.value = 'tools'
      return
    }
    loading.value = true
    error.value = ''
    try {
      caps.value = await api.getMCPCapabilities(id)
    } catch (e: any) {
      error.value = e.message
      caps.value = null
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)

async function toggleTool(name: string, enabled: boolean) {
  try {
    await api.setTool(name, enabled)
    if (props.server?.id) {
      caps.value = await api.getMCPCapabilities(props.server.id)
    }
  } catch (e: any) {
    error.value = e.message
  }
}

function onBackdrop(e: MouseEvent) {
  if (e.target === e.currentTarget) emit('close')
}
</script>

<template>
  <div
    v-if="open && server"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
    @click="onBackdrop"
  >
    <div class="bg-white rounded-lg shadow-xl w-full max-w-2xl max-h-[85vh] flex flex-col" @click.stop>
      <div class="px-4 py-3 border-b border-neutral-200 flex justify-between items-center">
        <div>
          <h2 class="font-semibold">{{ server.display_name || server.name }}</h2>
          <p class="text-xs text-neutral-500 font-mono">{{ server.name }}</p>
        </div>
        <button class="text-neutral-500 hover:text-neutral-800 text-xl leading-none px-2" @click="emit('close')">×</button>
      </div>

      <div class="px-4 pt-3 flex gap-2 border-b border-neutral-100">
        <button
          class="px-3 py-1 text-sm rounded"
          :class="tab === 'tools' ? 'bg-neutral-800 text-white' : 'bg-neutral-100'"
          @click="tab = 'tools'"
        >Tools</button>
        <button
          class="px-3 py-1 text-sm rounded"
          :class="tab === 'resources' ? 'bg-neutral-800 text-white' : 'bg-neutral-100'"
          @click="tab = 'resources'"
        >Resources</button>
        <button
          class="px-3 py-1 text-sm rounded"
          :class="tab === 'templates' ? 'bg-neutral-800 text-white' : 'bg-neutral-100'"
          @click="tab = 'templates'"
        >Templates</button>
      </div>

      <div class="p-4 overflow-y-auto flex-1">
        <div v-if="loading" class="text-sm text-neutral-500">Loading…</div>
        <div v-else-if="error" class="text-sm text-red-600">{{ error }}</div>
        <div v-else-if="caps && !caps.connected" class="text-sm text-neutral-500">
          Server is not connected. Enable it and click Reload.
        </div>
        <template v-else-if="caps">
          <div v-if="tab === 'tools'" class="space-y-2">
            <div
              v-for="t in caps.tools"
              :key="t.name"
              class="border border-neutral-200 rounded p-3 flex justify-between items-start gap-3"
            >
              <div class="min-w-0">
                <div class="font-mono text-sm">{{ t.mcp_name }}</div>
                <div class="text-xs text-neutral-400 font-mono truncate">{{ t.name }}</div>
                <div class="text-xs text-neutral-500 mt-1">{{ t.description }}</div>
              </div>
              <label class="text-sm flex items-center gap-2 shrink-0">
                <input
                  type="checkbox"
                  :checked="t.enabled"
                  @change="toggleTool(t.name, ($event.target as HTMLInputElement).checked)"
                />
                on
              </label>
            </div>
            <div v-if="!caps.tools.length" class="text-sm text-neutral-500">No tools.</div>
          </div>

          <div v-else-if="tab === 'resources'" class="space-y-2">
            <div v-for="r in caps.resources" :key="r.uri" class="border border-neutral-200 rounded p-3">
              <div class="font-mono text-sm">{{ r.name || r.uri }}</div>
              <div v-if="r.title" class="text-sm text-neutral-700">{{ r.title }}</div>
              <div class="text-xs text-neutral-400 font-mono break-all">{{ r.uri }}</div>
              <div v-if="r.description" class="text-xs text-neutral-500 mt-1">{{ r.description }}</div>
              <div v-if="r.mime_type" class="text-xs text-neutral-400 mt-1">{{ r.mime_type }}</div>
            </div>
            <div v-if="!caps.resources.length" class="text-sm text-neutral-500">No resources.</div>
          </div>

          <div v-else class="space-y-2">
            <div v-for="t in caps.templates" :key="t.uri_template" class="border border-neutral-200 rounded p-3">
              <div class="font-mono text-sm">{{ t.name || t.uri_template }}</div>
              <div v-if="t.title" class="text-sm text-neutral-700">{{ t.title }}</div>
              <div class="text-xs text-neutral-400 font-mono break-all">{{ t.uri_template }}</div>
              <div v-if="t.description" class="text-xs text-neutral-500 mt-1">{{ t.description }}</div>
              <div v-if="t.mime_type" class="text-xs text-neutral-400 mt-1">{{ t.mime_type }}</div>
            </div>
            <div v-if="!caps.templates.length" class="text-sm text-neutral-500">No resource templates.</div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>
