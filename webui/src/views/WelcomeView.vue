<script setup lang="ts">
import { computed, ref, onMounted, watch } from 'vue'
import { api } from '../api'
import { useChatStore } from '../stores/chat'
import { useLayoutStore } from '../stores/layout'
import { useSetupStore } from '../stores/setup'
import { useLightAppsStore } from '../stores/lightapps'
import { openLightApp } from '../lib/openLightApp'
import LocalSvgIcon from '../components/LocalSvgIcon.vue'
import type { Session } from '../types'

const layout = useLayoutStore()
const chatStore = useChatStore()
const setup = useSetupStore()
const lightApps = useLightAppsStore()
const sessions = ref<Session[]>([])
const draft = ref('')
const launching = ref<Record<string, boolean>>({})

const SUGGESTIONS = [
  {
    label: '做一个对账应用',
    prompt: '我想做一个对账应用',
  },
  {
    label: '分析瑞幸业绩',
    prompt: '我想分析最近一年瑞幸的业绩状况',
  },
  {
    label: '整理会议纪要',
    prompt: '帮我把最近的会议录音或笔记整理成结构化纪要，并列出待办事项',
  },
] as const

const previewApps = computed(() => lightApps.apps.slice(0, 5))

function openChat(id?: string) {
  if (!id) return
  const s = sessions.value.find((x) => x.id === id)
  const title = s?.title || ''
  chatStore.setSession(id, title)
  layout.openChatTab(id, title)
}

function applySuggestion(prompt: string) {
  draft.value = prompt
}

function submitDraft() {
  const text = draft.value.trim()
  if (!text) return
  const key = 'sess-' + Math.random().toString(36).slice(2, 10)
  chatStore.setPendingPrompt(text)
  chatStore.setSession(key, '')
  layout.openChatTab(key, 'New Chat')
  draft.value = ''
}

function onDraftKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    submitDraft()
  }
}

function appInitial(name: string) {
  const t = name.trim()
  return t ? t.slice(0, 1).toUpperCase() : 'A'
}

async function loadSessions() {
  try {
    const r = await api.listSessions()
    sessions.value = (r.sessions || []).slice(0, 8)
  } catch {}
}

onMounted(async () => {
  if (!setup.checked) {
    try {
      await setup.check()
    } catch {
      setup.checked = true
    }
  }
  if (!setup.needsSetup) {
    await loadSessions()
    await lightApps.load()
  }
})

watch(
  () => setup.needsSetup,
  (need) => {
    if (!need && setup.checked) {
      loadSessions()
      lightApps.load()
    }
  },
)

async function onAppClick(id: string) {
  const app = lightApps.apps.find((a) => a.id === id)
  if (!app) return
  if (app.status === 'running' && app.port) {
    await openLightApp(`http://127.0.0.1:${app.port}`, app.name || 'Light App')
    return
  }
  launching.value[id] = true
  try {
    const r = await api.launchLightApp(id)
    await lightApps.load()
    const fresh = lightApps.apps.find((a) => a.id === id)
    await openLightApp(r.url, fresh?.name || app.name || 'Light App')
  } finally {
    launching.value[id] = false
  }
}
</script>

<template>
  <div class="h-full overflow-y-auto overscroll-contain">
    <div class="max-w-[600px] mx-auto px-8 py-16">
      <!-- Brand -->
      <div class="flex items-center justify-center gap-3 mb-10">
        <img src="/images/icon-dark.svg" alt="Swiflow" class="w-12 h-12 shrink-0" />
        <div>
          <h1 class="text-2xl font-bold text-neutral-900 leading-tight">Swiflow</h1>
          <p class="text-neutral-500 text-sm mt-0.5">Self-hosted AI Agent Runtime</p>
        </div>
      </div>

      <template v-if="setup.checked && !setup.needsSetup">
        <!-- Composer -->
        <div class="mb-10">
          <div class="flex flex-wrap gap-2 mb-3 justify-center">
            <button
              v-for="s in SUGGESTIONS"
              :key="s.label"
              type="button"
              class="text-xs px-3 py-1.5 rounded-full border border-neutral-200 text-neutral-600 hover:border-neutral-300 hover:bg-neutral-50 transition-colors"
              @click="applySuggestion(s.prompt)"
            >{{ s.label }}</button>
          </div>
          <div class="relative border border-neutral-200 rounded-2xl bg-white shadow-sm focus-within:border-neutral-300 transition-colors">
            <textarea
              v-model="draft"
              rows="3"
              class="w-full resize-none bg-transparent px-4 pt-3.5 pb-12 text-sm text-neutral-900 placeholder:text-neutral-400 outline-none"
              placeholder="描述你想做的事…"
              @keydown="onDraftKeydown"
            />
            <div class="absolute right-2.5 bottom-2.5 flex items-center gap-2">
              <button
                type="button"
                class="inline-flex items-center justify-center w-8 h-8 rounded-lg bg-neutral-900 text-white hover:bg-neutral-800 disabled:opacity-40 transition-colors"
                :disabled="!draft.trim()"
                title="发送"
                @click="submitDraft"
              >
                <LocalSvgIcon name="send" :size="15" />
              </button>
            </div>
          </div>
        </div>

        <!-- Light Apps as icon row -->
        <div v-if="previewApps.length" class="mb-10">
          <div class="flex items-center justify-between mb-3">
            <h2 class="text-xs font-semibold text-neutral-500 uppercase tracking-wide">Light Apps</h2>
            <button
              type="button"
              class="text-xs text-neutral-400 hover:text-neutral-700 transition-colors"
              @click="layout.openLightApps()"
            >Manage</button>
          </div>
          <div class="flex items-start gap-4">
            <button
              v-for="app in previewApps"
              :key="app.id"
              type="button"
              class="w-14 flex flex-col items-center gap-1.5 group disabled:opacity-50"
              :disabled="!!launching[app.id]"
              :title="app.name"
              @click="onAppClick(app.id)"
            >
              <span
                class="relative w-12 h-12 rounded-2xl border border-neutral-200 bg-neutral-50 text-neutral-800 text-base font-semibold flex items-center justify-center group-hover:border-neutral-300 group-hover:bg-white transition-colors"
              >
                {{ launching[app.id] ? '…' : appInitial(app.name) }}
                <span
                  class="absolute -top-0.5 -right-0.5 w-2.5 h-2.5 rounded-full border-2 border-white"
                  :class="app.status === 'running' ? 'bg-emerald-500' : 'bg-neutral-300'"
                />
              </span>
              <span class="text-[11px] text-neutral-600 truncate w-full text-center leading-tight">{{ app.name }}</span>
            </button>
          </div>
        </div>

        <!-- Recent Chats -->
        <!-- <section v-if="sessions.length" class="mb-6 border border-neutral-200 rounded-lg overflow-hidden bg-white">
          <div class="px-4 py-2.5 border-b border-neutral-100 bg-neutral-50/80">
            <h2 class="text-xs font-semibold text-neutral-500 uppercase tracking-wide">Recent Chats</h2>
          </div>
          <ul class="divide-y divide-neutral-100">
            <li v-for="s in sessions" :key="s.id">
              <button
                type="button"
                class="cursor-pointer w-full text-left px-4 py-3 flex items-center gap-3 hover:bg-neutral-50/80 transition-colors"
                @click="openChat(s.id)"
              >
                <LocalSvgIcon name="chat" class="shrink-0 text-neutral-400" :size="15" />
                <span class="truncate text-sm text-neutral-900 flex-1 min-w-0">{{ s.title || s.id }}</span>
                <span class="text-xs text-neutral-400 shrink-0 tabular-nums">{{ s.agent }}</span>
              </button>
            </li>
          </ul>
        </section> -->
      </template>
    </div>
  </div>
</template>
