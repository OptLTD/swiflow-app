<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAgentsStore } from '../stores/agents'
import { useProvidersStore } from '../stores/providers'
import { api } from '../api'
import LocalSvgIcon from '../components/LocalSvgIcon.vue'
import ProviderDialog from '../components/ProviderDialog.vue'
import { DEFAULT_AGENT_KEY, DEFAULT_PROVIDER_NAME } from '../constants/defaults'

const agentsStore = useAgentsStore()
const providersStore = useProvidersStore()
const error = ref('')
const saving = ref(false)
const providerOpen = ref(false)
const form = ref({
  display_name: '',
  system_extra: '',
})

const defaultAgent = computed(() =>
  agentsStore.agents.find((a) => a.key === DEFAULT_AGENT_KEY) || agentsStore.agents[0] || null,
)

const defaultProvider = computed(() =>
  providersStore.providers.find((p) => p.name === DEFAULT_PROVIDER_NAME) || null,
)

onMounted(load)

async function load() {
  try {
    await Promise.all([agentsStore.load(), providersStore.load()])
    syncForm()
  } catch (e: any) {
    error.value = e.message
  }
}

function syncForm() {
  const a = defaultAgent.value
  if (!a) return
  form.value = {
    display_name: a.display_name || '',
    system_extra: a.system_extra || '',
  }
}

async function onProviderSaved(model: string) {
  await Promise.all([providersStore.load(), agentsStore.load()])
  if (!defaultAgent.value && defaultProvider.value) {
    try {
      await api.createAgent({
        key: DEFAULT_AGENT_KEY,
        display_name: 'Default Agent',
        provider: DEFAULT_PROVIDER_NAME,
        model,
      })
      await agentsStore.load()
    } catch (e: any) {
      error.value = e.message
    }
  }
  syncForm()
}

async function save() {
  const a = defaultAgent.value
  if (!a) return
  saving.value = true
  error.value = ''
  try {
    await api.updateAgent(a.key, {
      display_name: form.value.display_name,
      provider: DEFAULT_PROVIDER_NAME,
      system_extra: form.value.system_extra,
    })
    await agentsStore.load()
    syncForm()
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="p-6 max-w-[640px] mx-auto">
    <div class="flex items-start justify-between gap-4 mb-4">
      <div>
        <h1 class="text-xl font-bold mb-1">Agent</h1>
      </div>
      <div class="shrink-0 flex items-center gap-2">
        <button
          type="button"
          class="h-8 px-3 flex items-center gap-1.5 rounded border border-neutral-200 bg-white hover:bg-neutral-50 text-sm text-neutral-700"
          title="Provider 配置"
          @click="providerOpen = true"
        >
          <LocalSvgIcon name="provider" :size="15" />
          Provider
        </button>
        <button
          type="button"
          class="h-8 px-3 bg-neutral-800 text-white rounded text-sm disabled:opacity-50"
          :disabled="saving || !defaultAgent"
          @click="save"
        >{{ saving ? 'Saving…' : 'Save' }}</button>
      </div>
    </div>

    <div v-if="error" class="text-red-600 mb-2 text-sm">{{ error }}</div>

    <div v-if="!defaultProvider" class="text-sm text-neutral-500 border border-neutral-200 rounded p-4 bg-neutral-50 mb-4">
      尚未配置 Provider，请点击右上角 <strong>Provider</strong> 按钮添加 API 连接。
    </div>

    <div v-else-if="defaultAgent" class="border border-neutral-200 rounded p-4 bg-white space-y-3">
      <div v-if="defaultProvider" class="text-xs text-neutral-400 font-mono truncate pb-1 border-b border-neutral-100">
        {{ defaultProvider.api_base }}
        <span v-if="defaultAgent.model"> · {{ defaultAgent.model }}</span>
      </div>

      <div>
        <label class="block text-xs text-neutral-500 mb-1">Title</label>
        <input v-model="form.display_name" class="w-full border rounded px-2 py-1.5 text-sm" placeholder="Default Agent" />
      </div>

      <div>
        <label class="block text-xs text-neutral-500 mb-1">Prompt</label>
        <textarea
          v-model="form.system_extra"
          class="w-full border rounded px-2 py-1.5 text-sm"
          rows="4"
          placeholder="Optional additional system instructions"
        />
      </div>
    </div>

    <ProviderDialog :open="providerOpen" @close="providerOpen = false" @saved="onProviderSaved" />
  </div>
</template>
