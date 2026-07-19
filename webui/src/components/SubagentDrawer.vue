<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'
import { renderMarkdown } from '../lib/markdown'
import ToolCallBlock from './ToolCallBlock.vue'
import ThinkingBlock from './ThinkingBlock.vue'
import type { Message } from '../types'

const props = defineProps<{ sessionKey: string | null }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()

interface Row {
  role: string
  content: string
  thinking?: string
  tool_name?: string
  id?: string
  arguments?: Record<string, unknown>
  isError?: boolean
  startedAt?: number
  endedAt?: number
}

function parseTs(s?: string): number | undefined {
  if (!s) return undefined
  const t = Date.parse(s)
  return Number.isNaN(t) ? undefined : t
}

const rows = ref<Row[]>([])
const loading = ref(false)
const error = ref('')

function render(md: string): string {
  return renderMarkdown(md)
}

function looksLikeToolError(content: string | undefined): boolean {
  if (!content) return false
  return /^error:/i.test(content.trim())
}

// Pair assistant tool_calls with their tool-result messages (same as ChatPanel).
function mapMessages(raw: Message[]): Row[] {
  const out: Row[] = []
  const toolByID = new Map<string, Row>()
  for (const m of raw) {
    if (m.role === 'assistant' && m.tool_calls?.length) {
      if (m.content || m.thinking) out.push({ role: 'assistant', content: m.content, thinking: m.thinking })
      const startedAt = parseTs(m.created_at)
      for (const tc of m.tool_calls) {
        const entry: Row = { role: 'tool', id: tc.id, tool_name: tc.name, arguments: tc.arguments, content: '', isError: false, startedAt }
        out.push(entry)
        if (tc.id) toolByID.set(tc.id, entry)
      }
      continue
    }
    if (m.role === 'tool') {
      const isErr = looksLikeToolError(m.content)
      const existing = m.tool_call_id ? toolByID.get(m.tool_call_id) : undefined
      if (existing) {
        existing.content = m.content || ''
        existing.isError = isErr
        existing.endedAt = parseTs(m.created_at)
        if (m.tool_name) existing.tool_name = m.tool_name
      } else {
        out.push({ role: 'tool', id: m.tool_call_id, tool_name: m.tool_name, content: m.content || '', isError: isErr, endedAt: parseTs(m.created_at) })
      }
      continue
    }
    out.push({ role: m.role, content: m.content, thinking: m.thinking })
  }
  for (const m of out) {
    if (m.role === 'tool' && !m.content && m.endedAt == null) {
      m.content = t('subagent.toolResultMissing')
      m.isError = true
      m.endedAt = m.startedAt
    }
  }
  return out
}

async function load(key: string) {
  loading.value = true
  error.value = ''
  rows.value = []
  try {
    const r = await api.getSession(key)
    rows.value = mapMessages(r.messages || [])
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

watch(
  () => props.sessionKey,
  (key) => {
    if (key) load(key)
  },
  { immediate: true },
)
</script>

<template>
  <div class="fixed inset-0 z-40 flex justify-end" @click.self="emit('close')">
    <div class="absolute inset-0 bg-black/30"></div>
    <div class="relative z-10 w-full max-w-[520px] h-full bg-white shadow-xl flex flex-col">
      <div class="shrink-0 flex items-center justify-between px-4 py-2 border-b border-neutral-200">
        <div class="min-w-0">
          <div class="text-sm font-medium text-neutral-800">{{ t('subagent.title') }}</div>
          <div class="text-xs text-neutral-400 font-mono truncate">{{ sessionKey }}</div>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <button
            class="text-xs text-neutral-500 hover:text-neutral-800 hover:underline disabled:opacity-40"
            :disabled="loading || !sessionKey"
            @click="sessionKey && load(sessionKey)"
          >{{ t('subagent.refresh') }}</button>
          <button class="text-neutral-400 hover:text-neutral-700 text-lg leading-none px-1" @click="emit('close')">×</button>
        </div>
      </div>
      <div class="flex-1 overflow-y-auto p-3 space-y-3">
        <div v-if="loading" class="text-sm text-neutral-400">{{ t('subagent.loading') }}</div>
        <div v-else-if="error" class="text-sm text-red-600">{{ error }}</div>
        <div v-else-if="!rows.length" class="text-sm text-neutral-400">{{ t('subagent.empty') }}</div>
        <template v-for="(m, i) in rows" :key="i">
          <div v-if="m.role === 'user'" class="flex justify-end">
            <div
              v-if="m.content"
              class="max-w-[85%] bg-neutral-100 text-neutral-700 text-sm rounded-lg px-3 py-2 whitespace-pre-wrap"
            >{{ m.content }}</div>
          </div>
          <div v-else-if="m.role === 'assistant'">
            <ThinkingBlock v-if="m.thinking" :content="m.thinking" />
            <div v-if="m.content" class="prose-swiflow" v-html="render(m.content)"></div>
          </div>
          <div v-else-if="m.role === 'tool'">
            <ToolCallBlock
              :name="m.tool_name || ''"
              :args="m.arguments"
              :content="m.content"
              :is-error="m.isError"
              :started-at="m.startedAt"
              :ended-at="m.endedAt"
            />
          </div>
        </template>
      </div>
    </div>
  </div>
</template>
