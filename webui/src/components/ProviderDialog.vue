<script setup lang="ts">
import { ref, watch } from 'vue'
import { useProvidersStore } from '../stores/providers'
import { useAgentsStore } from '../stores/agents'
import { api } from '../api'
import { DEFAULT_AGENT_KEY, DEFAULT_PROVIDER_NAME } from '../constants/defaults'
import { PROVIDER_PRESETS, guessPresetId } from '../constants/providerPresets'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: []; saved: [model: string] }>()

const providersStore = useProvidersStore()
const agentsStore = useAgentsStore()
const error = ref('')
const loading = ref(false)
const saving = ref(false)
const providerId = ref('')
const activePreset = ref('openai')
const form = ref({
  api_base: 'https://api.openai.com/v1',
  api_key: '',
  model: 'gpt-4o-mini',
})

const defaultProvider = () =>
  providersStore.providers.find((p) => p.name === DEFAULT_PROVIDER_NAME) || null

const defaultAgent = () =>
  agentsStore.agents.find((a) => a.key === DEFAULT_AGENT_KEY) || null

watch(
  () => props.open,
  async (open) => {
    if (!open) {
      error.value = ''
      return
    }
    loading.value = true
    error.value = ''
    try {
      await Promise.all([providersStore.load(), agentsStore.load()])
      syncForm()
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)

function syncForm() {
  const p = defaultProvider()
  if (!p) {
    providerId.value = ''
    form.value = {
      api_base: 'https://api.openai.com/v1',
      api_key: '',
      model: 'gpt-4o-mini',
    }
    activePreset.value = guessPresetId(form.value.api_base)
    return
  }
  providerId.value = p.id
  form.value = {
    api_base: p.api_base,
    api_key: '',
    model: p.model || 'gpt-4o-mini',
  }
  activePreset.value = guessPresetId(p.api_base)
}

function applyPreset(id: string) {
  activePreset.value = id
  const preset = PROVIDER_PRESETS.find((p) => p.id === id)
  if (!preset || id === 'other') return
  form.value.api_base = preset.api_base
  form.value.model = preset.model
}

async function ensureAgentBound() {
  const a = defaultAgent()
  if (a) {
    if (a.txt_model !== DEFAULT_PROVIDER_NAME) {
      await api.updateAgent(a.key, { txt_model: DEFAULT_PROVIDER_NAME })
    }
    return
  }
  await api.createAgent({
    key: DEFAULT_AGENT_KEY,
    display: 'Default Agent',
    txt_model: DEFAULT_PROVIDER_NAME,
  })
}

async function save() {
  saving.value = true
  error.value = ''
  try {
    const p = defaultProvider()
    if (p) {
      const body: Record<string, unknown> = {
        api_base: form.value.api_base,
        model: form.value.model,
        enabled: true,
      }
      if (form.value.api_key) body.api_key = form.value.api_key
      await api.updateProvider(p.id, body)
    } else {
      if (!form.value.api_key) {
        error.value = 'API Key 必填'
        return
      }
      await api.createProvider({
        name: DEFAULT_PROVIDER_NAME,
        api_base: form.value.api_base,
        api_key: form.value.api_key,
        model: form.value.model,
        enabled: true,
      })
    }
    await providersStore.load()
    await ensureAgentBound()
    await agentsStore.load()
    syncForm()
    emit('saved', form.value.model)
    emit('close')
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

function onBackdrop(e: MouseEvent) {
  if (e.target === e.currentTarget) emit('close')
}
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
    @click="onBackdrop"
  >
    <div
      class="bg-white rounded-lg shadow-xl w-full max-w-md flex flex-col"
      @click.stop
    >
      <div class="px-4 py-3 border-b border-neutral-200 flex justify-between items-center shrink-0">
        <div>
          <h2 class="font-semibold">Provider</h2>
          <!-- <p class="text-xs text-neutral-500 font-mono">{{ DEFAULT_PROVIDER_NAME }}</p> -->
        </div>
        <button
          class="text-neutral-500 hover:text-neutral-800 text-xl leading-none px-2"
          @click="emit('close')"
        >×</button>
      </div>

      <form class="p-4 space-y-3" @submit.prevent="save">
        <div v-if="loading" class="text-sm text-neutral-500">Loading…</div>
        <div v-if="error" class="text-sm text-red-600">{{ error }}</div>

        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="preset in PROVIDER_PRESETS"
            :key="preset.id"
            type="button"
            class="px-2.5 py-1 rounded text-xs border transition-colors"
            :class="activePreset === preset.id
              ? 'bg-neutral-800 text-white border-neutral-800'
              : 'bg-white text-neutral-600 border-neutral-200 hover:bg-neutral-50'"
            @click="applyPreset(preset.id)"
          >{{ preset.label }}</button>
        </div>

        <div>
          <label class="block text-xs text-neutral-500 mb-1">API Base</label>
          <input v-model="form.api_base" class="w-full border rounded px-2 py-1.5 text-sm font-mono" />
        </div>
        <div>
          <label class="block text-xs text-neutral-500 mb-1">Model</label>
          <input v-model="form.model" class="w-full border rounded px-2 py-1.5 text-sm font-mono" placeholder="gpt-4o-mini" />
        </div>
        <div>
          <label class="block text-xs text-neutral-500 mb-1">API Key</label>
          <input
            v-model="form.api_key"
            type="password"
            class="w-full border rounded px-2 py-1.5 text-sm"
            :placeholder="providerId ? '留空则不修改' : 'sk-…'"
          />
        </div>

        <div class="flex justify-end gap-2 pt-1">
          <button type="button" class="px-3 py-1.5 border rounded text-sm" @click="emit('close')">Cancel</button>
          <button
            type="submit"
            class="px-3 py-1.5 bg-neutral-800 text-white rounded text-sm disabled:opacity-50"
            :disabled="saving"
          >{{ saving ? 'Saving…' : 'Save' }}</button>
        </div>
      </form>
    </div>
  </div>
</template>
