<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { api } from '../api'
import { useChatStore } from '../stores/chat'
import { useLayoutStore } from '../stores/layout'
import { useSetupStore } from '../stores/setup'
import { useLightAppsStore } from '../stores/lightapps'
import { openExternalURL } from '../lib/openExternal'
import LocalSvgIcon from '../components/LocalSvgIcon.vue'
import type { Session } from '../types'

const layout = useLayoutStore()
const chatStore = useChatStore()
const setup = useSetupStore()
const lightApps = useLightAppsStore()
const sessions = ref<Session[]>([])

function openChat(id?: string) {
  if (!id) return
  const s = sessions.value.find((x) => x.id === id)
  const title = s?.title || ''
  chatStore.setSession(id, title)
  layout.openChatTab(id, title)
}

function startNewChat() {
  const key = 'sess-' + Math.random().toString(36).slice(2, 10)
  chatStore.setSession(key, '')
  layout.openChatTab(key, 'New Chat')
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

const launching = ref<Record<string, boolean>>({})

async function launchApp(id: string) {
  launching.value[id] = true
  try {
    const r = await api.launchLightApp(id)
    await lightApps.load()
    await openExternalURL(r.url)
  } finally {
    launching.value[id] = false
  }
}

async function openApp(id: string) {
  const app = lightApps.apps.find((a) => a.id === id)
  if (app?.port) await openExternalURL(`http://127.0.0.1:${app.port}`)
}
</script>

<template>
  <div class="h-full overflow-y-auto overscroll-contain">
    <div class="max-w-[600px] mx-auto px-8 py-16">
      <!-- Brand -->
      <div class="flex items-center justify-center gap-3 mb-12">
        <img src="/images/icon-dark.svg" alt="Swiflow" class="w-12 h-12 shrink-0" />
        <div>
          <h1 class="text-2xl font-bold text-neutral-900 leading-tight">Swiflow</h1>
          <p class="text-neutral-500 text-sm mt-0.5">Self-hosted AI Agent Runtime</p>
        </div>
      </div>

      <template v-if="setup.checked && !setup.needsSetup">
        <!-- Quick Actions -->
        <div class="grid grid-cols-2 gap-3 mb-10">
          <button
            type="button"
            class="cursor-pointer border border-neutral-200 rounded-lg p-4 text-left hover:bg-neutral-50 transition-colors flex items-start gap-3"
            @click="startNewChat"
          >
            <LocalSvgIcon name="chat" class="text-neutral-700 mt-0.5" :size="18" />
            <div class="min-w-0">
              <div class="font-medium text-sm">New Chat</div>
              <div class="text-xs text-neutral-400">Start a conversation</div>
            </div>
          </button>
          <button
            type="button"
            class="cursor-pointer border border-neutral-200 rounded-lg p-4 text-left hover:bg-neutral-50 transition-colors flex items-start gap-3"
            @click="layout.openExplore()"
          >
            <LocalSvgIcon name="folder" class="text-neutral-700 mt-0.5" :size="18" />
            <div class="min-w-0">
              <div class="font-medium text-sm">Explore</div>
              <div class="text-xs text-neutral-400">Browse workspace</div>
            </div>
          </button>
        </div>

        <!-- Light Apps -->
        <div v-if="lightApps.apps.length" class="mb-10">
          <div class="flex items-center justify-between mb-3">
            <h2 class="text-sm font-semibold text-neutral-500 uppercase tracking-wide">Light Apps</h2>
            <button class="text-xs text-neutral-400 hover:text-neutral-700 transition-colors" @click="layout.openLightApps()">Manage</button>
          </div>
          <div class="space-y-1">
            <div
              v-for="app in lightApps.apps"
              :key="app.id"
              class="w-full px-3 py-2 rounded hover:bg-neutral-100 flex items-center justify-between gap-2"
            >
              <div class="flex items-center gap-2 min-w-0">
                <span class="truncate text-sm">{{ app.name }}</span>
                <span
                  class="shrink-0 text-xs px-1.5 py-0.5 rounded"
                  :class="app.status === 'running' ? 'bg-green-50 text-green-700' : 'bg-neutral-100 text-neutral-500'"
                >{{ app.status }}</span>
              </div>
              <div class="shrink-0 flex items-center gap-2">
                <button
                  v-if="app.status === 'running'"
                  class="text-xs px-2.5 py-1 rounded border border-neutral-200 text-neutral-700 hover:bg-neutral-100 transition-colors"
                  @click="openApp(app.id)"
                >Open</button>
                <button
                  v-else
                  class="text-xs px-2.5 py-1 rounded border border-neutral-200 text-neutral-700 hover:bg-neutral-100 transition-colors disabled:opacity-50"
                  :disabled="launching[app.id]"
                  @click="launchApp(app.id)"
                >{{ launching[app.id] ? 'Launching…' : 'Launch' }}</button>
              </div>
            </div>
          </div>
        </div>

        <!-- Recent Sessions -->
        <div v-if="sessions.length" class="mb-10">
          <h2 class="text-sm font-semibold text-neutral-500 uppercase tracking-wide mb-3">Recent Sessions</h2>
          <div class="space-y-1">
            <button
              v-for="s in sessions"
              :key="s.id"
              type="button"
              class="cursor-pointer w-full text-left px-3 py-2 rounded hover:bg-neutral-100 flex items-center justify-between"
              @click="openChat(s.id)"
            >
              <span class="truncate text-sm">{{ s.title || s.id }}</span>
              <span class="text-xs text-neutral-400 shrink-0 ml-2">{{ s.agent }}</span>
            </button>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
