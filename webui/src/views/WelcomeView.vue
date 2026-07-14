<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { useAgentsStore } from '../stores/agents'
import { useChatStore } from '../stores/chat'
import { useLayoutStore } from '../stores/layout'
import LocalSvgIcon from '../components/LocalSvgIcon.vue'
import type { Session } from '../types'

const layout = useLayoutStore()
const chatStore = useChatStore()
const agentsStore = useAgentsStore()
const sessions = ref<Session[]>([])

function openChat(key?: string) {
  if (!key) return
  const s = sessions.value.find((x) => x.key === key)
  const title = s?.title || ''
  chatStore.setSession(key, title)
  layout.openChatTab(key, title)
}

function startNewChat() {
  const key = 'sess-' + Math.random().toString(36).slice(2, 10)
  chatStore.setSession(key, '')
  layout.openChatTab(key, 'New Chat')
}

onMounted(async () => {
  try {
    const r = await api.listSessions()
    sessions.value = (r.sessions || []).slice(0, 8)
  } catch {}
  agentsStore.load().catch(() => {})
})
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

      <!-- Recent Sessions -->
      <div v-if="sessions.length" class="mb-10">
        <h2 class="text-sm font-semibold text-neutral-500 uppercase tracking-wide mb-3">Recent Sessions</h2>
        <div class="space-y-1">
          <button
            v-for="s in sessions"
            :key="s.key"
            type="button"
            class="cursor-pointer w-full text-left px-3 py-2 rounded hover:bg-neutral-100 flex items-center justify-between"
            @click="openChat(s.key)"
          >
            <span class="truncate text-sm">{{ s.title || s.key }}</span>
            <span class="text-xs text-neutral-400 shrink-0 ml-2">{{ s.agent_key }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
