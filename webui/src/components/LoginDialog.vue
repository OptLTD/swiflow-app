<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { api } from '../api'

const auth = useAuthStore()
const token = ref('')
const error = ref('')
const loading = ref(false)
const emit = defineEmits<{ ok: [] }>()

async function connect() {
  if (!token.value.trim()) {
    error.value = 'token required'
    return
  }
  loading.value = true
  error.value = ''
  // Validate against an authed endpoint before accepting the token.
  auth.login(token.value.trim())
  try {
    await api.listAgents()
    emit('ok')
  } catch (e: any) {
    error.value = e.message || 'invalid token'
    auth.logout()
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
    <div class="bg-white rounded-lg shadow-xl p-6 w-80">
      <div class="flex items-center gap-3 mb-4">
        <img src="/images/icon-dark.svg" alt="" class="w-10 h-10" />
        <div>
          <div class="font-bold text-lg leading-tight">Swiflow</div>
          <div class="text-sm text-neutral-500">Enter your auth token to continue.</div>
        </div>
      </div>
      <input
        v-model="token"
        type="password"
        class="w-full border border-neutral-300 rounded px-3 py-2 mb-2 focus:outline-none focus:border-neutral-500"
        placeholder="auth token"
        @keydown.enter="connect"
      />
      <div v-if="error" class="text-red-600 text-sm mb-2">{{ error }}</div>
      <button
        class="w-full px-3 py-2 bg-neutral-800 text-white rounded disabled:opacity-50"
        :disabled="loading"
        @click="connect"
      >
        {{ loading ? 'Connecting…' : 'Connect' }}
      </button>
    </div>
  </div>
</template>
