<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useToolsStore } from '../stores/tools'
import { api } from '../api'

const auth = useAuthStore()
const toolsStore = useToolsStore()
const token = ref(auth.token)
const status = ref('')
const error = ref('')

onMounted(async () => {
  if (auth.isAuthed) {
    try { await toolsStore.load() } catch (e: any) { error.value = e.message }
  }
})

async function save() {
  status.value = ''
  auth.login(token.value)
  try {
    await api.listAgents()
    status.value = 'connected'
    await toolsStore.load()
  } catch (e: any) {
    status.value = 'unauthorized / error: ' + e.message
    auth.logout()
  }
}

async function toggleTool(name: string, enabled: boolean) {
  try {
    await api.setTool(name, enabled)
    await toolsStore.load()
  } catch (e: any) {
    error.value = e.message
  }
}
</script>

<template>
  <div class="p-6 max-w-[960px] mx-auto">
    <h1 class="text-xl font-bold mb-4">Settings</h1>
    <label class="block text-sm font-medium mb-1">Auth token</label>
    <input v-model="token" type="password" class="w-full border rounded px-2 py-1 mb-2" placeholder="auth token" />
    <button class="px-3 py-1 bg-neutral-800 text-white rounded text-sm" @click="save">Save & test</button>
    <div v-if="status" class="mt-2 text-sm" :class="status === 'connected' ? 'text-green-600' : 'text-red-600'">{{ status }}</div>

    <h2 class="text-lg font-semibold mt-8 mb-2">Tools</h2>
    <div v-if="error" class="text-red-600 mb-2 text-sm">{{ error }}</div>
    <div class="space-y-2">
      <div v-for="t in toolsStore.tools" :key="t.name" class="border border-neutral-200 rounded p-3 bg-white flex justify-between items-center">
        <div>
          <div class="font-mono text-sm">{{ t.name }}</div>
          <div class="text-xs text-neutral-500">{{ t.description }}</div>
        </div>
        <label class="text-sm flex items-center gap-2">
          <input type="checkbox" :checked="t.enabled" @change="toggleTool(t.name, ($event.target as HTMLInputElement).checked)" />
          enabled
        </label>
      </div>
      <div v-if="!toolsStore.tools.length" class="text-neutral-500 text-sm">No tools loaded.</div>
    </div>
  </div>
</template>
