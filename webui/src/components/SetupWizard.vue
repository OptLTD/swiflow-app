<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { api } from '../api'
import { useAgentsStore } from '../stores/agents'
import { useProvidersStore } from '../stores/providers'
import { useToastStore } from '../stores/toast'
import { DEFAULT_AGENT_KEY, DEFAULT_PROVIDER_NAME } from '../constants/defaults'
import { PROVIDER_PRESETS } from '../constants/providerPresets'
import { PROMPT_STYLE_PRESETS, guessPromptStyleId } from '../constants/promptStyles'
import type { RuntimeBinary, RuntimeInfo } from '../types'

const emit = defineEmits<{ done: [] }>()

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
  sys_prompt: '',
})
const activePromptStyle = ref('none')

const runtime = ref<RuntimeInfo | null>(null)
const runtimeLoading = ref(false)
const runtimeError = ref('')

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

watch(step, (s) => {
  error.value = ''
  if (s === 1 && defaultAgent.value) {
    agentForm.value = {
      display: defaultAgent.value.display || 'Default Agent',
      sys_prompt: defaultAgent.value.sys_prompt || '',
    }
    activePromptStyle.value = guessPromptStyleId(agentForm.value.sys_prompt)
  }
  if (s === 2) loadRuntime()
})

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
  agentForm.value.sys_prompt = preset.prompt
}

function onPromptInput() {
  activePromptStyle.value = guessPromptStyleId(agentForm.value.sys_prompt)
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
        error.value = 'API Key 必填'
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
      error.value = 'Provider 保存失败，请重试'
      return
    }
    step.value = 1
  } catch (e: any) {
    error.value = e?.name === 'TimeoutError' || e?.name === 'AbortError'
      ? '请求超时，请确认后端已启动'
      : (e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function saveAgent() {
  if (!defaultProvider.value) {
    error.value = '请先完成 LLM 配置'
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
        sys_prompt: agentForm.value.sys_prompt,
      })
    } else {
      await api.createAgent({
        key: DEFAULT_AGENT_KEY,
        display: agentForm.value.display || 'Default Agent',
        txt_model: DEFAULT_PROVIDER_NAME,
        sys_prompt: agentForm.value.sys_prompt,
      })
    }
    await agentsStore.load()
    if (!defaultAgent.value) {
      error.value = 'Agent 保存失败，请重试'
      return
    }
    step.value = 2
  } catch (e: any) {
    error.value = e?.name === 'TimeoutError' || e?.name === 'AbortError'
      ? '请求超时，请确认后端已启动'
      : (e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function loadRuntime() {
  runtimeLoading.value = true
  runtimeError.value = ''
  try {
    runtime.value = await api.getRuntime()
  } catch (e: any) {
    runtimeError.value = e.message
    runtime.value = null
  } finally {
    runtimeLoading.value = false
  }
}

function onInstall(name: string) {
  toast.error(`${name} 自动安装暂未实现，请自行安装后点「重新检测」`)
}

function finish() {
  emit('done')
}

function statusLabel(b?: RuntimeBinary | null) {
  if (!b) return '—'
  if (!b.found) return '未检测到'
  return b.version || '已安装'
}

onMounted(() => {
  syncLLMFromProvider()
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
          <h2 id="setup-wizard-title" class="font-semibold text-neutral-900">初始化设置</h2>
          <p class="text-xs text-neutral-500 mt-0.5">配置 LLM、Agent 与运行环境后即可开始对话</p>
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
              >{{ preset.label }}</button>
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
                :placeholder="defaultProvider ? '留空则不修改' : 'sk-…'"
              />
            </div>
            <div class="flex justify-end pt-1">
              <button
                type="submit"
                class="px-3 py-1.5 bg-neutral-800 text-white rounded text-sm disabled:opacity-50"
                :disabled="saving"
              >{{ saving ? 'Saving…' : '下一步' }}</button>
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
                v-model="agentForm.sys_prompt"
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
                >{{ preset.label }}</button>
              </div>
            </div>
            <div class="flex justify-between pt-1">
              <button type="button" class="px-3 py-1.5 border rounded text-sm" @click="step = 0">上一步</button>
              <button
                type="submit"
                class="px-3 py-1.5 bg-neutral-800 text-white rounded text-sm disabled:opacity-50"
                :disabled="saving"
              >{{ saving ? 'Saving…' : '下一步' }}</button>
            </div>
          </form>

          <!-- Step 3: Env -->
          <div v-else class="space-y-3">
            <p class="text-sm text-neutral-500">
              检测本机 Python3 / Node.js。Agent 执行脚本时常需要它们，可稍后再装。
            </p>
            <div v-if="runtimeLoading" class="text-sm text-neutral-500">检测中…</div>
            <div v-else-if="runtimeError" class="text-sm text-red-600">{{ runtimeError }}</div>
            <div v-else class="space-y-2">
              <div
                v-for="item in [
                  { key: 'python3', label: 'Python3', bin: runtime?.python3 },
                  { key: 'node', label: 'Node.js', bin: runtime?.node },
                ]"
                :key="item.key"
                class="border border-neutral-200 rounded p-3 flex items-start justify-between gap-3"
              >
                <div class="min-w-0">
                  <div class="font-medium text-sm">{{ item.label }}</div>
                  <div class="text-xs mt-0.5" :class="item.bin?.found ? 'text-green-700' : 'text-amber-700'">
                    {{ statusLabel(item.bin) }}
                  </div>
                  <div v-if="item.bin?.found && item.bin.path" class="text-xs text-neutral-400 font-mono truncate mt-0.5">
                    {{ item.bin.path }}
                  </div>
                </div>
                <button
                  v-if="!item.bin?.found"
                  type="button"
                  class="shrink-0 px-2.5 py-1 border rounded text-xs text-neutral-600 hover:bg-neutral-50"
                  @click="onInstall(item.label)"
                >安装</button>
              </div>
            </div>
            <div class="flex justify-between items-center pt-1 gap-2">
              <button type="button" class="px-3 py-1.5 border rounded text-sm" @click="step = 1">上一步</button>
              <div class="flex gap-2">
                <button
                  type="button"
                  class="px-3 py-1.5 border rounded text-sm"
                  :disabled="runtimeLoading"
                  @click="loadRuntime"
                >重新检测</button>
                <button
                  type="button"
                  class="px-3 py-1.5 bg-neutral-800 text-white rounded text-sm"
                  @click="finish"
                >完成</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

