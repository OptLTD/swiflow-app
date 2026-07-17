<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { api } from '../api'
import type { RunSnapshot } from '../types'

const props = defineProps<{ open: boolean; focusSession?: string | null }>()
const emit = defineEmits<{ close: [] }>()

const runs = ref<RunSnapshot[]>([])
const selected = ref<RunSnapshot | null>(null)
const children = ref<RunSnapshot[]>([])
const loading = ref(false)
const error = ref('')

async function refresh() {
  loading.value = true
  error.value = ''
  try {
    const r = await api.listRuns()
    runs.value = r.runs || []
    const focus = props.focusSession
    if (focus) {
      await select(focus)
    } else if (selected.value) {
      await select(selected.value.session_id)
    }
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

async function select(id: string) {
  try {
    const r = await api.getRun(id)
    selected.value = r.run
    children.value = r.children || []
  } catch {
    selected.value = runs.value.find((x) => x.session_id === id) || null
    children.value = []
  }
}

function fmtWall(ms?: number): string {
  if (ms == null || ms < 0) return ''
  const s = Math.floor(ms / 1000)
  if (s < 60) return `${s}s`
  return `${Math.floor(s / 60)}m${s % 60}s`
}

watch(
  () => props.open,
  (v) => {
    if (v) refresh()
  },
)

onMounted(() => {
  if (props.open) refresh()
})
</script>

<template>
  <div v-if="open" class="fixed inset-0 z-40 flex justify-end" @click.self="emit('close')">
    <div class="absolute inset-0 bg-black/30"></div>
    <div class="relative z-10 w-full max-w-[560px] h-full bg-white shadow-xl flex flex-col">
      <div class="shrink-0 flex items-center justify-between px-4 py-2 border-b border-neutral-200">
        <div>
          <div class="text-sm font-medium text-neutral-800">Runtime Harness</div>
          <div class="text-xs text-neutral-400">观测主/子会话与偏离（一期仅人工审查）</div>
        </div>
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="text-xs text-neutral-500 hover:underline disabled:opacity-40"
            :disabled="loading"
            @click="refresh"
          >刷新</button>
          <button type="button" class="text-neutral-400 hover:text-neutral-700 text-lg leading-none px-1" @click="emit('close')">×</button>
        </div>
      </div>

      <div class="flex-1 min-h-0 flex">
        <div class="w-[40%] border-r border-neutral-100 overflow-y-auto">
          <div v-if="error" class="p-3 text-xs text-red-600">{{ error }}</div>
          <div v-else-if="loading && !runs.length" class="p-3 text-xs text-neutral-400">加载中…</div>
          <div v-else-if="!runs.length" class="p-3 text-xs text-neutral-400">暂无运行记录</div>
          <button
            v-for="r in runs"
            :key="r.session_id"
            type="button"
            class="w-full text-left px-3 py-2 border-b border-neutral-50 hover:bg-neutral-50"
            :class="selected?.session_id === r.session_id ? 'bg-amber-50' : ''"
            @click="select(r.session_id)"
          >
            <div class="text-xs font-mono truncate">{{ r.session_id }}</div>
            <div class="text-[11px] text-neutral-500 flex gap-2 mt-0.5">
              <span>{{ r.status }}</span>
              <span v-if="r.drift?.length" class="text-amber-700">drift {{ r.drift.length }}</span>
            </div>
          </button>
        </div>

        <div class="flex-1 overflow-y-auto p-3 space-y-3 text-xs">
          <div v-if="!selected" class="text-neutral-400">选择左侧会话</div>
          <template v-else>
            <div>
              <div class="text-neutral-500 mb-1">Goal</div>
              <div class="whitespace-pre-wrap text-neutral-800">{{ selected.goal || '—' }}</div>
            </div>
            <div class="grid grid-cols-2 gap-2">
              <div>status: <b>{{ selected.status }}</b></div>
              <div>round: {{ selected.round }}<span v-if="selected.max_rounds">/{{ selected.max_rounds }}</span></div>
              <div>tool: {{ selected.current_tool || '—' }}</div>
              <div>wall: {{ fmtWall(selected.metrics?.wall_ms) || '—' }}</div>
              <div>tools: {{ selected.metrics?.tool_calls ?? 0 }}</div>
              <div>fail: {{ selected.metrics?.failures ?? 0 }}</div>
            </div>
            <div>
              <div class="text-neutral-500 mb-1">Last action</div>
              <div class="font-mono text-neutral-700 break-all">{{ selected.last_action || '—' }}</div>
            </div>
            <div v-if="selected.todos?.length">
              <div class="text-neutral-500 mb-1">Todos</div>
              <ul class="space-y-1">
                <li v-for="t in selected.todos" :key="t.id" class="flex gap-1">
                  <span>{{ t.done ? '✓' : '○' }}</span>
                  <span :class="t.done ? 'text-neutral-400 line-through' : ''">{{ t.text }}</span>
                </li>
              </ul>
            </div>
            <div v-if="selected.drift?.length">
              <div class="text-neutral-500 mb-1">Drift</div>
              <div
                v-for="(d, i) in selected.drift"
                :key="i"
                class="mb-1 rounded border border-amber-200 bg-amber-50 px-2 py-1"
              >
                <div class="font-mono text-amber-900">{{ d.code }} · {{ d.severity }}</div>
                <div class="text-amber-800">{{ d.message }}</div>
              </div>
            </div>
            <div v-if="children.length">
              <div class="text-neutral-500 mb-1">Children</div>
              <button
                v-for="c in children"
                :key="c.session_id"
                type="button"
                class="block w-full text-left font-mono text-neutral-700 hover:underline mb-1"
                @click="select(c.session_id)"
              >{{ c.session_id }} ({{ c.status }})</button>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>
