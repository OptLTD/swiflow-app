<script setup lang="ts">
import { computed, watch } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useChatStore } from './stores/chat'
import LoginDialog from './components/LoginDialog.vue'

const auth = useAuthStore()
const chat = useChatStore()
const route = useRoute()
// Phase 1 is single-tenant; this label is a placeholder until Phase 3 wires
// real tenants.
const tenantName = 'local'

const isChat = computed(() => route.path === '/')
// Close the drawer when leaving the chat page.
watch(isChat, (v) => {
  if (!v) chat.closeDrawer()
})

const menu = [
  { to: '/', label: 'Chats' },
  { to: '/agents', label: 'Agents' },
  { to: '/skills', label: 'Skills' },
  { to: '/providers', label: 'Providers' },
  { to: '/settings', label: 'Settings' },
]
</script>

<template>
  <div class="h-full flex flex-col">
    <header class="h-12 shrink-0 border-b border-neutral-200 bg-white">
      <div class="h-full max-w-[960px] mx-auto px-4 flex items-center gap-2">
        <div class="flex items-center gap-2 font-bold">
          <span class="inline-flex w-6 h-6 rounded bg-neutral-800 text-white text-xs font-bold items-center justify-center leading-none">M</span>
          Mira
        </div>
        <template v-if="isChat">
          <button
            class="inline-flex flex-col items-center justify-center gap-[3px] px-1 py-1 hover:bg-neutral-100 rounded"
            @click="chat.toggleDrawer()"
            aria-label="sessions"
          >
            <span class="block w-[18px] h-[2px] bg-neutral-700 rounded"></span>
            <span class="block w-[18px] h-[2px] bg-neutral-700 rounded"></span>
            <span class="block w-[18px] h-[2px] bg-neutral-700 rounded"></span>
          </button>
          <span class="text-sm text-neutral-600 truncate">{{ chat.label }}</span>
        </template>
        <div class="flex-1"></div>
        <div class="text-sm text-neutral-500">{{ tenantName }}</div>
        <div class="relative group">
          <button class="px-2 py-1 rounded hover:bg-neutral-100 text-lg leading-none">⚙</button>
          <div
            class="absolute right-0 top-full hidden group-hover:block bg-white border border-neutral-200 rounded shadow-lg w-40 z-20"
          >
            <RouterLink
              v-for="m in menu"
              :key="m.to"
              :to="m.to"
              class="block px-3 py-2 text-sm hover:bg-neutral-100"
            >{{ m.label }}</RouterLink>
            <button
              class="block w-full text-left px-3 py-2 text-sm text-red-600 hover:bg-neutral-100 border-t border-neutral-100"
              @click="auth.logout()"
            >Logout</button>
          </div>
        </div>
      </div>
    </header>
    <main class="flex-1 min-h-0">
      <RouterView />
    </main>
    <LoginDialog v-if="!auth.isAuthed" />
  </div>
</template>
