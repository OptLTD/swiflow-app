<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'
import { useAgentsStore } from '../stores/agents'
import { useProvidersStore } from '../stores/providers'
import { useToastStore } from '../stores/toast'
import { DEFAULT_AGENT_KEY, DEFAULT_PROVIDER_NAME } from '../constants/defaults'
import { PROVIDER_PRESETS } from '../constants/providerPresets'
import { PROMPT_STYLE_PRESETS, guessPromptStyleId } from '../constants/promptStyles'
import type { RuntimeBinary, RuntimeInfo } from '../types'

const emit = defineEmits<{ done: [] }>()

const { t } = useI18n()
const providersStore = useProvidersStore()
const agentsStore = useAgentsStore()
const toast = useToastStore()

const step = ref(0) // 0 LLM, 1 Agent, 2 Env
const error = ref('')
const saving = ref(false)

const activePreset = ref('openai')
const llmForm = ref({
  api_base: 'https://api.openai.com/v1',
  api_key: '',
  model: 'gpt-4o-mini',
})

const agentForm = ref({
  display: 'Default Agent',
  prompt: '',
})
const activePromptStyle = ref('none')

const runtime = ref<RuntimeInfo | null>(null)
const runtimeLoading = ref(false)
const runtimeError = ref('')
const installMode = ref<'mainland' | 'standard'>('mainland')
const installing = ref<Record<string, boolean>>({})
let pollTimer: ReturnType<typeof setInterval> | null = null

const steps = [
  { key: 'llm', label: 'LLM' },
  { key: 'agent', label: 'Agent' },
  { key: 'env', label: 'Env' },
]

const defaultProvider = computed(() =>
  (providersStore.providers ?? []).find((p) => p.name === DEFAULT_PROVIDER_NAME) || null,
)

const defaultAgent = computed(() =>
  (agentsStore.agents ?? []).find((a) => a.key === DEFAULT_AGENT_KEY) || null,
)

const pythonReady = computed(() =>
  !!(runtime.value?.python3?.found && runtime.value?.uvx?.found),
)
const nodeReady = computed(() =>
  !!(runtime.value?.node?.found && runtime.value?.npx?.found),
)

watch(step, (s) => {
  error.value = ''
  if (s === 1 && defaultAgent.value) {
    agentForm.value = {
      display: defaultAgent.value.display || 'Default Agent',
      prompt: defaultAgent.value.prompt || '',
    }
    activePromptStyle.value = guessPromptStyleId(agentForm.value.prompt)
  }
  if (s === 2) {
    void loadRuntime()
    startPolling()
  } else {
    stopPolling()
  }
})

function providerPresetLabel(preset: { id: string; label: string }) {
  return preset.id === 'other' ? t('provider.other') : preset.label
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

function applyPreset(id: string) {
  activePreset.value = id
  const preset = PROVIDER_PRESETS.find((p) => p.id === id)
  if (!preset || id === 'other') return
  llmForm.value.api_base = preset.api_base
  llmForm.value.model = preset.model
}

function applyPromptStyle(id: string) {
  activePromptStyle.value = id
  const preset = PROMPT_STYLE_PRESETS.find((p) => p.id === id)
  if (!preset) return
  agentForm.value.prompt = preset.prompt
}

function onPromptInput() {
  activePromptStyle.value = guessPromptStyleId(agentForm.value.prompt)
}

function syncLLMFromProvider() {
  const p = defaultProvider.value
  if (!p) return
  llmForm.value = {
    api_base: p.api_base,
    api_key: '',
    model: p.model || 'gpt-4o-mini',
  }
  const preset = PROVIDER_PRESETS.find((x) => x.api_base === p.api_base)
  activePreset.value = preset?.id || 'other'
}

async function saveLLM() {
  saving.value = true
  error.value = ''
  try {
    const p = defaultProvider.value
    if (p) {
      const body: Record<string, unknown> = {
        api_base: llmForm.value.api_base,
        model: llmForm.value.model,
        enabled: true,
      }
      if (llmForm.value.api_key) body.api_key = llmForm.value.api_key
      await api.updateProvider(p.id, body)
    } else {
      if (!llmForm.value.api_key.trim()) {
        error.value = t('setup.apiKeyRequired')
        return
      }
      await api.createProvider({
        name: DEFAULT_PROVIDER_NAME,
        api_base: llmForm.value.api_base,
        api_key: llmForm.value.api_key,
        model: llmForm.value.model,
        enabled: true,
      })
    }
    await providersStore.load()
    if (!defaultProvider.value) {
      error.value = t('setup.providerSaveFailed')
      return
    }
    step.value = 1
  } catch (e: any) {
    error.value = e?.name === 'TimeoutError' || e?.name === 'AbortError'
      ? t('setup.timeout')
      : (e.message || t('setup.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function saveAgent() {
  if (!defaultProvider.value) {
    error.value = t('setup.finishLlmFirst')
    return
  }
  saving.value = true
  error.value = ''
  try {
    const a = defaultAgent.value
    if (a) {
      await api.updateAgent(a.key, {
        display: agentForm.value.display,
        txt_model: DEFAULT_PROVIDER_NAME,
        prompt: agentForm.value.prompt,
      })
    } else {
      await api.createAgent({
        key: DEFAULT_AGENT_KEY,
        display: agentForm.value.display || 'Default Agent',
        txt_model: DEFAULT_PROVIDER_NAME,
        prompt: agentForm.value.prompt,
      })
    }
    await agentsStore.load()
    if (!defaultAgent.value) {
      error.value = t('setup.agentSaveFailed')
      return
    }
    step.value = 2
  } catch (e: any) {
    error.value = e?.name === 'TimeoutError' || e?.name === 'AbortError'
      ? t('setup.timeout')
      : (e.message || t('setup.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function loadRuntime() {
  runtimeLoading.value = true
  runtimeError.value = ''
  try {
    runtime.value = await api.getRuntime()
    // Sync installing flags from server (survives page refresh mid-install).
    if (runtime.value.installing) {
      installing.value = { ...installing.value, ...runtime.value.installing }
    }
    // Clear local installing when binaries appear.
    if (pythonReady.value) installing.value['uvx-py'] = false
    if (nodeReady.value) installing.value['js-npx'] = false
  } catch (e: any) {
    runtimeError.value = e.message
    runtime.value = null
  } finally {
    runtimeLoading.value = false
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(() => {
    const busy = installing.value['uvx-py'] || installing.value['js-npx']
      || runtime.value?.installing?.['uvx-py'] || runtime.value?.installing?.['js-npx']
    if (busy || step.value === 2) void loadRuntime()
  }, 3000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function onInstall(kind: 'uvx-py' | 'js-npx') {
  installing.value[kind] = true
  error.value = ''
  try {
    await api.installRuntime(kind, installMode.value)
    toast.success(kind === 'uvx-py'
      ? t('setup.installStartedPy')
      : t('setup.installStartedNode'))
    startPolling()
    await loadRuntime()
  } catch (e: any) {
    installing.value[kind] = false
    const msg = e?.message || t('setup.installStartFailed')
    error.value = msg
    toast.error(msg)
  }
}

function finish() {
  stopPolling()
  emit('done')
}

function statusLabel(b?: RuntimeBinary | null) {
  if (!b) return '—'
  if (!b.found) return t('common.notDetected')
  return b.version || t('common.installed')
}

onMounted(() => {
  syncLLMFromProvider()
})

onUnmounted(() => {
  stopPolling()
})
</script>

<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-[60] bg-black/40 flex items-center justify-center p-4">
      <div
        class="bg-white rounded-lg shadow-xl w-full max-w-md overflow-hidden"
        role="dialog"
        aria-modal="true"
        aria-labelledby="setup-wizard-title"
      >
        <div class="px-4 py-3 border-b border-neutral-100">
          <h2 id="setup-wizard-title" class="font-semibold text-neutral-900">{{ t('setup.title') }}</h2>
          <p class="text-xs text-neutral-500 mt-0.5">{{ t('setup.subtitle') }}</p>
        </div>

        <div class="px-4 pt-3 flex gap-2">
          <div
            v-for="(s, i) in steps"
            :key="s.key"
            class="flex-1 text-center text-xs py-1.5 rounded border"
            :class="i === step
              ? 'bg-neutral-800 text-white border-neutral-800'
              : i < step
                ? 'bg-neutral-100 text-neutral-700 border-neutral-200'
                : 'bg-white text-neutral-400 border-neutral-100'"
          >
            {{ i + 1 }}. {{ s.label }}
          </div>
        </div>

        <div class="p-4 space-y-3">
          <div v-if="error" class="text-sm text-red-600">{{ error }}</div>

          <!-- Step 1: LLM -->
          <form v-if="step === 0" class="space-y-3" @submit.prevent="saveLLM">
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
              >{{ providerPresetLabel(preset) }}</button>
            </div>
            <div>
              <label class="block text-xs text-neutral-500 mb-1">API Base</label>
              <input v-model="llmForm.api_base" class="w-full border rounded px-2 py-1.5 text-sm font-mono" />
            </div>
            <div>
              <label class="block text-xs text-neutral-500 mb-1">Model</label>
              <input v-model="llmForm.model" class="w-full border rounded px-2 py-1.5 text-sm font-mono" placeholder="gpt-4o-mini" />
            </div>
            <div>
              <label class="block text-xs text-neutral-500 mb-1">API Key</label>
              <input
                v-model="llmForm.api_key"
                type="password"
                class="w-full border rounded px-2 py-1.5 text-sm"
                :placeholder="defaultProvider ? t('setup.leaveBlank') : 'sk-…'"
              />
            </div>
            <div class="flex justify-end pt-1">
              <button
                type="submit"
                class="px-3 py-1.5 bg-neutral-800 text-white rounded text-sm disabled:opacity-50"
                :disabled="saving"
              >{{ saving ? t('common.saving') : t('common.next') }}</button>
            </div>
          </form>

          <!-- Step 2: Agent -->
          <form v-else-if="step === 1" class="space-y-3" @submit.prevent="saveAgent">
            <div>
              <label class="block text-xs text-neutral-500 mb-1">Title</label>
              <input v-model="agentForm.display" class="w-full border rounded px-2 py-1.5 text-sm" placeholder="Default Agent" />
            </div>
            <div>
              <label class="block text-xs text-neutral-500 mb-1">Prompt</label>
              <textarea
                v-model="agentForm.prompt"
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
            <div class="flex justify-between pt-1">
              <button type="button" class="px-3 py-1.5 border rounded text-sm" @click="step = 0">{{ t('common.back') }}</button>
              <button
                type="submit"
                class="px-3 py-1.5 bg-neutral-800 text-white rounded text-sm disabled:opacity-50"
                :disabled="saving"
              >{{ saving ? t('common.saving') : t('common.next') }}</button>
            </div>
          </form>

          <!-- Step 3: Env -->
          <div v-else class="space-y-3">
            <p class="text-sm text-neutral-500">
              {{ t('setup.runtimeHint') }}
            </p>

            <div class="flex items-center gap-2 text-xs">
              <span class="text-neutral-500">{{ t('setup.mirror') }}</span>
              <button
                type="button"
                class="px-2 py-0.5 rounded border"
                :class="installMode === 'mainland'
                  ? 'bg-neutral-800 text-white border-neutral-800'
                  : 'border-neutral-200 text-neutral-600'"
                @click="installMode = 'mainland'"
              >{{ t('setup.mainland') }}</button>
              <button
                type="button"
                class="px-2 py-0.5 rounded border"
                :class="installMode === 'standard'
                  ? 'bg-neutral-800 text-white border-neutral-800'
                  : 'border-neutral-200 text-neutral-600'"
                @click="installMode = 'standard'"
              >{{ t('setup.official') }}</button>
            </div>

            <div v-if="runtimeLoading && !runtime" class="text-sm text-neutral-500">{{ t('common.detecting') }}</div>
            <div v-else-if="runtimeError" class="text-sm text-red-600">{{ runtimeError }}</div>
            <div v-else class="space-y-2">
              <!-- Python + UV -->
              <div class="border border-neutral-200 rounded p-3 space-y-2">
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0">
                    <div class="font-medium text-sm">Python + UV</div>
                    <div class="text-xs mt-1 space-y-0.5">
                      <div :class="runtime?.python3?.found ? 'text-green-700' : 'text-amber-700'">
                        Python: {{ statusLabel(runtime?.python3) }}
                      </div>
                      <div :class="runtime?.uvx?.found ? 'text-green-700' : 'text-amber-700'">
                        uvx: {{ statusLabel(runtime?.uvx) }}
                      </div>
                    </div>
                    <div v-if="runtime?.python3?.path" class="text-xs text-neutral-400 font-mono truncate mt-0.5">
                      {{ runtime.python3.path }}
                    </div>
                  </div>
                  <button
                    v-if="!pythonReady"
                    type="button"
                    class="shrink-0 px-2.5 py-1 border rounded text-xs text-neutral-600 hover:bg-neutral-50 disabled:opacity-50"
                    :disabled="!!installing['uvx-py']"
                    @click="onInstall('uvx-py')"
                  >{{ installing['uvx-py'] ? t('common.installing') : t('common.install') }}</button>
                  <span v-else class="shrink-0 text-xs text-green-700 px-1">{{ t('common.ready') }}</span>
                </div>
                <p v-if="installing['uvx-py']" class="text-xs text-neutral-500">
                  {{ t('setup.installBackground') }}
                </p>
              </div>

              <!-- Node + npx -->
              <div class="border border-neutral-200 rounded p-3 space-y-2">
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0">
                    <div class="font-medium text-sm">Node.js + npx</div>
                    <div class="text-xs mt-1 space-y-0.5">
                      <div :class="runtime?.node?.found ? 'text-green-700' : 'text-amber-700'">
                        Node: {{ statusLabel(runtime?.node) }}
                      </div>
                      <div :class="runtime?.npx?.found ? 'text-green-700' : 'text-amber-700'">
                        npx: {{ statusLabel(runtime?.npx) }}
                      </div>
                    </div>
                    <div v-if="runtime?.node?.path" class="text-xs text-neutral-400 font-mono truncate mt-0.5">
                      {{ runtime.node.path }}
                    </div>
                  </div>
                  <button
                    v-if="!nodeReady"
                    type="button"
                    class="shrink-0 px-2.5 py-1 border rounded text-xs text-neutral-600 hover:bg-neutral-50 disabled:opacity-50"
                    :disabled="!!installing['js-npx']"
                    @click="onInstall('js-npx')"
                  >{{ installing['js-npx'] ? t('common.installing') : t('common.install') }}</button>
                  <span v-else class="shrink-0 text-xs text-green-700 px-1">{{ t('common.ready') }}</span>
                </div>
                <p v-if="installing['js-npx']" class="text-xs text-neutral-500">
                  {{ t('setup.installBackground') }}
                </p>
              </div>
            </div>

            <div class="flex justify-between items-center pt-1 gap-2">
              <button type="button" class="px-3 py-1.5 border rounded text-sm" @click="step = 1">{{ t('common.back') }}</button>
              <div class="flex gap-2">
                <button
                  type="button"
                  class="px-3 py-1.5 border rounded text-sm"
                  :disabled="runtimeLoading"
                  @click="loadRuntime"
                >{{ t('common.detectAgain') }}</button>
                <button
                  type="button"
                  class="px-3 py-1.5 bg-neutral-800 text-white rounded text-sm"
                  @click="finish"
                >{{ t('common.done') }}</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
