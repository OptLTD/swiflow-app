<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useProvidersStore } from '../stores/providers'
import { useAgentsStore } from '../stores/agents'
import { api } from '../api'
import {
  DEFAULT_AGENT_KEY,
  DEFAULT_PROVIDER_NAME,
  DEFAULT_VISION_PROVIDER_NAME,
} from '../constants/defaults'
import { PROVIDER_PRESETS, DEFAULT_PROVIDER_PRESET_ID, defaultProviderPreset, guessPresetId } from '../constants/providerPresets'

const props = defineProps<{
  open: boolean
  /** Which tab to open when the dialog becomes visible. */
  initialKind?: 'text' | 'vision'
}>()
const emit = defineEmits<{ close: []; saved: [model: string] }>()

type Kind = 'text' | 'vision'

const providersStore = useProvidersStore()
const agentsStore = useAgentsStore()
const error = ref('')
const loading = ref(false)
const saving = ref(false)
const kind = ref<Kind>('text')

const defaultPreset = defaultProviderPreset()

const textForm = ref({
  api_base: defaultPreset.api_base,
  api_key: '',
  model: defaultPreset.model,
})
const visionForm = ref({
  api_base: defaultPreset.api_base,
  api_key: '',
  model: defaultPreset.vision_model || defaultPreset.model,
})
const textPreset = ref(DEFAULT_PROVIDER_PRESET_ID)
const visionPreset = ref(DEFAULT_PROVIDER_PRESET_ID)
const textProviderId = ref('')
const visionProviderId = ref('')

const form = computed({
  get: () => (kind.value === 'text' ? textForm.value : visionForm.value),
  set: (v) => {
    if (kind.value === 'text') textForm.value = v
    else visionForm.value = v
  },
})
const activePreset = computed({
  get: () => (kind.value === 'text' ? textPreset.value : visionPreset.value),
  set: (v: string) => {
    if (kind.value === 'text') textPreset.value = v
    else visionPreset.value = v
  },
})
const providerId = computed(() =>
  kind.value === 'text' ? textProviderId.value : visionProviderId.value,
)
const providerName = computed(() =>
  kind.value === 'text' ? DEFAULT_PROVIDER_NAME : DEFAULT_VISION_PROVIDER_NAME,
)

const findProvider = (name: string) =>
  providersStore.providers.find((p) => p.name === name) || null

const defaultAgent = () =>
  agentsStore.agents.find((a) => a.key === DEFAULT_AGENT_KEY) || null

watch(
  () => props.open,
  async (open) => {
    if (!open) {
      error.value = ''
      return
    }
    kind.value = props.initialKind === 'vision' ? 'vision' : 'text'
    loading.value = true
    error.value = ''
    try {
      await Promise.all([providersStore.load(), agentsStore.load()])
      syncForms()
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)

function emptyTextForm() {
  const p = defaultProviderPreset()
  return {
    api_base: p.api_base,
    api_key: '',
    model: p.model,
  }
}

function syncForms() {
  const text = findProvider(DEFAULT_PROVIDER_NAME)
  if (!text) {
    textProviderId.value = ''
    textForm.value = emptyTextForm()
    textPreset.value = DEFAULT_PROVIDER_PRESET_ID
  } else {
    textProviderId.value = text.id
    textForm.value = {
      api_base: text.api_base,
      api_key: '',
      model: text.model || defaultProviderPreset().model,
    }
    textPreset.value = guessPresetId(text.api_base)
  }

  const vision = findProvider(DEFAULT_VISION_PROVIDER_NAME)
  if (!vision) {
    visionProviderId.value = ''
    visionForm.value = {
      api_base: textForm.value.api_base,
      api_key: '',
      model: PROVIDER_PRESETS.find((p) => p.id === textPreset.value)?.vision_model
        || defaultProviderPreset().vision_model
        || defaultProviderPreset().model,
    }
    visionPreset.value = textPreset.value
  } else {
    visionProviderId.value = vision.id
    visionForm.value = {
      api_base: vision.api_base,
      api_key: '',
      model: vision.model || defaultProviderPreset().vision_model || defaultProviderPreset().model,
    }
    visionPreset.value = guessPresetId(vision.api_base)
  }
}

function applyPreset(id: string) {
  activePreset.value = id
  const preset = PROVIDER_PRESETS.find((p) => p.id === id)
  if (!preset || id === 'other') return
  form.value = {
    ...form.value,
    api_base: preset.api_base,
    model: kind.value === 'text' ? preset.model : (preset.vision_model || preset.model),
  }
}

async function ensureAgentBound() {
  const a = defaultAgent()
  const fields: Record<string, string> = {}
  if (kind.value === 'text') {
    fields.txt_model = DEFAULT_PROVIDER_NAME
  } else {
    fields.img_model = DEFAULT_VISION_PROVIDER_NAME
  }
  if (a) {
    const needTxt = kind.value === 'text' && a.txt_model !== DEFAULT_PROVIDER_NAME
    const needImg = kind.value === 'vision' && a.img_model !== DEFAULT_VISION_PROVIDER_NAME
    if (needTxt || needImg) {
      await api.updateAgent(a.key, fields)
    }
    return
  }
  await api.createAgent({
    key: DEFAULT_AGENT_KEY,
    display: 'Default Agent',
    txt_model: DEFAULT_PROVIDER_NAME,
    ...(kind.value === 'vision' ? { img_model: DEFAULT_VISION_PROVIDER_NAME } : {}),
  })
}

async function save() {
  saving.value = true
  error.value = ''
  try {
    const current = form.value
    const name = providerName.value
    const existing = findProvider(name)
    if (existing) {
      const body: Record<string, unknown> = {
        api_base: current.api_base,
        model: current.model,
        enabled: true,
      }
      if (current.api_key) body.api_key = current.api_key
      await api.updateProvider(existing.id, body)
    } else {
      if (!current.api_key) {
        error.value = 'API Key 必填'
        return
      }
      await api.createProvider({
        name,
        display: kind.value === 'text' ? 'Text' : 'Vision',
        api_base: current.api_base,
        api_key: current.api_key,
        model: current.model,
        enabled: true,
      })
    }
    await providersStore.load()
    await ensureAgentBound()
    await agentsStore.load()
    syncForms()
    emit('saved', current.model)
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
      <div class="px-4 pt-3 border-b border-neutral-200 shrink-0 flex items-end justify-between gap-2">
        <div class="flex gap-4 text-sm">
          <button
            type="button"
            class="pb-2 border-b-2 transition-colors"
            :class="kind === 'text'
              ? 'border-neutral-900 text-neutral-900 font-medium'
              : 'border-transparent text-neutral-500 hover:text-neutral-800'"
            @click="kind = 'text'"
          >推理模型</button>
          <button
            type="button"
            class="pb-2 border-b-2 transition-colors"
            :class="kind === 'vision'
              ? 'border-neutral-900 text-neutral-900 font-medium'
              : 'border-transparent text-neutral-500 hover:text-neutral-800'"
            @click="kind = 'vision'"
          >视觉模型</button>
        </div>
        <button
          class="text-neutral-500 hover:text-neutral-800 text-xl leading-none px-2 pb-1.5"
          type="button"
          @click="emit('close')"
        >×</button>
      </div>

      <form class="p-4 space-y-3" @submit.prevent="save">
        <div v-if="loading" class="text-sm text-neutral-500">Loading…</div>
        <div v-if="error" class="text-sm text-red-600">{{ error }}</div>

        <p class="text-xs text-neutral-500">
          <template v-if="kind === 'text'">配置推理 / 对话所用的模型（绑定到 Agent 的 txt_model）。</template>
          <template v-else>配置视觉 / 多模态所用的模型（绑定到 Agent 的 img_model）。</template>
        </p>

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
          <input
            v-model="form.model"
            class="w-full border rounded px-2 py-1.5 text-sm font-mono"
            :placeholder="kind === 'text' ? defaultPreset.model : (defaultPreset.vision_model || defaultPreset.model)"
          />
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
