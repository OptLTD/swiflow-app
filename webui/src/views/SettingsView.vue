<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import { api } from '../api'

const auth = useAuthStore()
const token = ref(auth.token)
const status = ref('')

onMounted(() => {
  token.value = auth.token
})

async function save() {
  status.value = ''
  auth.login(token.value)
  try {
    await api.listAgents()
    status.value = 'connected'
  } catch (e: any) {
    status.value = 'unauthorized / error: ' + e.message
    auth.logout()
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
  </div>
</template>
