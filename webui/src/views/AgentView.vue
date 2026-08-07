<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAgentsStore } from '../stores/agents'
import { useProvidersStore } from '../stores/providers'
import { api } from '../api'
import LocalSvgIcon from '../components/LocalSvgIcon.vue'
import ProviderDialog from '../components/ProviderDialog.vue'
import { DEFAULT_AGENT_KEY, DEFAULT_PROVIDER_NAME, DEFAULT_VISION_PROVIDER_NAME } from '../constants/defaults'
import { PROMPT_STYLE_PRESETS, guessPromptStyleId } from '../constants/promptStyles'

const agentsStore = useAgentsStore()
const providersStore = useProvidersStore()
const { t } = useI18n()
const error = ref('')
const saving = ref(false)
const providerOpen = ref(false)
const providerKind = ref<'text' | 'vision'>('text')
const activePromptStyle = ref('none')
const form = ref({
  display: '',
  prompt: '',
  charter: '',
})

const defaultAgent = computed(() =>
  agentsStore.agents.find((a) => a.key === DEFAULT_AGENT_KEY) || agentsStore.agents[0] || null,
)

const defaultProvider = computed(() =>
  providersStore.providers.find((p) => p.name === DEFAULT_PROVIDER_NAME) || null,
)

const visionProvider = computed(() =>
  providersStore.providers.find((p) => p.name === DEFAULT_VISION_PROVIDER_NAME) || null,
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

function openProvider(kind: 'text' | 'vision') {
  providerKind.value = kind
  providerOpen.value = true
}

function syncForm() {
  const a = defaultAgent.value
  if (!a) return
  form.value = {
    display: a.display || '',
    prompt: a.prompt || '',
    charter: a.charter || '',
  }
  activePromptStyle.value = guessPromptStyleId(form.value.prompt)
}

function promptStyleLabel(id: string) {
  const keys: Record<string, string> = {
    concise: 'promptStyles.concise',
    engineer: 'promptStyles.engineer',
    researcher: 'promptStyles.research',
    teacher: 'promptStyles.mentor',
  }
  return keys[id] ? t(keys[id]) : id
}

function applyPromptStyle(id: string) {
  activePromptStyle.value = id
  const preset = PROMPT_STYLE_PRESETS.find((p) => p.id === id)
  if (!preset) return
  form.value.prompt = preset.prompt
}

function onPromptInput() {
  activePromptStyle.value = guessPromptStyleId(form.value.prompt)
}

async function onProviderSaved(model: string) {
  await Promise.all([providersStore.load(), agentsStore.load()])
  if (!defaultAgent.value && defaultProvider.value) {
    try {
      await api.createAgent({
        key: DEFAULT_AGENT_KEY,
        display: 'Default Agent',
        txt_model: DEFAULT_PROVIDER_NAME,
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
      display: form.value.display,
      txt_model: DEFAULT_PROVIDER_NAME,
      prompt: form.value.prompt,
      charter: form.value.charter,
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
  <div class="p-6  mx-auto">
    <div class="flex items-start justify-between gap-4 mb-4">
      <div>
        <h1 class="text-xl font-bold mb-1">Agent</h1>
      </div>
      <div class="shrink-0 flex items-center gap-2">
        <button
          type="button"
          :title="t('agent.textModelTitle')"
          class="h-8 px-3 flex items-center gap-1.5 rounded border border-neutral-200 bg-white hover:bg-neutral-50 text-sm text-neutral-700"
          @click="openProvider('text')"
        >
          <LocalSvgIcon name="provider" :size="15" />
          {{ t('agent.textModel') }}
        </button>
        <button
          type="button"
          :title="t('agent.imgModelTitle')"
          class="h-8 px-3 flex items-center gap-1.5 rounded border border-neutral-200 bg-white hover:bg-neutral-50 text-sm text-neutral-700"
          @click="openProvider('vision')"
        >
          <LocalSvgIcon name="provider" :size="15" />
          {{ t('agent.imgModel') }}
        </button>
        <button
          @click="save" type="button"
          class="h-8 px-3 bg-neutral-800 text-white rounded text-sm disabled:opacity-50"
          :disabled="saving || !defaultAgent"
        >{{ saving ? t('common.saving') : t('common.save') }}</button>
      </div>
    </div>

    <div v-if="error" class="text-red-600 mb-2 text-sm">{{ error }}</div>

    <div v-if="!defaultProvider" class="text-sm text-neutral-500 border border-neutral-200 rounded p-4 bg-neutral-50 mb-4">
      {{ t('agent.noTextModel') }}
    </div>

    <div v-else-if="defaultAgent" class="border border-neutral-200 rounded p-4 bg-white space-y-3">
      <div class="text-xs text-neutral-400 font-mono space-y-0.5 pb-1 border-b border-neutral-100">
        <div class="truncate">
          <span class="text-neutral-500">{{ t('agent.text') }}</span>
          · {{ defaultProvider.api_base }}
          <span v-if="defaultProvider.model"> · {{ defaultProvider.model }}</span>
        </div>
        <div class="truncate">
          <span class="text-neutral-500">{{ t('agent.vision') }}</span>
          <template v-if="visionProvider">
            · {{ visionProvider.api_base }}
            <span v-if="visionProvider.model"> · {{ visionProvider.model }}</span>
          </template>
          <template v-else> · {{ t('agent.notConfigured') }}</template>
        </div>
      </div>

      <div>
        <label class="block text-xs text-neutral-500 mb-1">Title</label>
        <input v-model="form.display" class="w-full border rounded px-2 py-1.5 text-sm" placeholder="Default Agent" />
      </div>

      <div>
        <label class="block text-xs text-neutral-500 mb-1">Prompt</label>
        <textarea
          v-model="form.prompt"
          class="w-full border rounded px-2 py-1.5 text-sm"
          rows="4"
          placeholder="Optional additional system instructions"
          @input="onPromptInput"
        />
        <div class="flex flex-wrap gap-1.5 mt-2">
          <button
            v-for="preset in PROMPT_STYLE_PRESETS"
            :key="preset.id"
            type="button"
            class="px-2.5 py-1 rounded text-xs border transition-colors"
            :class="activePromptStyle === preset.id
              ? 'bg-neutral-800 text-white border-neutral-800'
              : 'bg-white text-neutral-600 border-neutral-200 hover:bg-neutral-50'"
            @click="applyPromptStyle(preset.id)"
          >{{ promptStyleLabel(preset.id) }}</button>
        </div>
      </div>

      <div>
        <label class="block text-xs text-neutral-500 mb-1">{{ t('agent.charter') }}</label>
        <textarea
          v-model="form.charter"
          class="w-full border rounded px-2 py-1.5 text-sm font-mono"
          rows="5"
          :placeholder="t('agent.charterPlaceholder')"
        />
        <p class="text-xs text-neutral-400 mt-1">{{ t('agent.charterHint') }}</p>
      </div>
    </div>

    <ProviderDialog
      :open="providerOpen"
      :initial-kind="providerKind"
      @close="providerOpen = false"
      @saved="onProviderSaved"
    />
  </div>
</template>
