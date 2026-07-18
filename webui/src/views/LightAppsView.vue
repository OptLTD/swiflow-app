<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'
import { useLightAppsStore } from '../stores/lightapps'
import { openLightApp } from '../lib/openLightApp'

const store = useLightAppsStore()
const launching = ref<Record<string, boolean>>({})
const deleting = ref<Record<string, boolean>>({})

// env
const env = ref<Record<string, string>>({})
const newKey = ref('')
const newValue = ref('')
const savingEnv = ref(false)

onMounted(async () => {
  await store.load()
  await loadEnv()
})

async function loadEnv() {
  const r = await api.listLightAppEnv()
  env.value = r.env ?? {}
}

async function addEnv() {
  if (!newKey.value.trim()) return
  savingEnv.value = true
  try {
    await api.setLightAppEnv(newKey.value.trim(), newValue.value)
    newKey.value = ''
    newValue.value = ''
    await loadEnv()
  } finally {
    savingEnv.value = false
  }
}

async function removeEnv(key: string) {
  await api.deleteLightAppEnv(key)
  await loadEnv()
}

async function launch(id: string) {
  launching.value[id] = true
  try {
    const r = await api.launchLightApp(id)
    await store.load()
    const app = store.apps.find((a) => a.id === id)
    await openLightApp(r.url, app?.name || 'Light App')
  } finally {
    launching.value[id] = false
  }
}

async function open(id: string) {
  const app = store.apps.find((a) => a.id === id)
  if (app?.port) await openLightApp(`http://127.0.0.1:${app.port}`, app.name || 'Light App')
}

async function stop(id: string) {
  await api.stopLightApp(id)
  await store.load()
}

async function remove(id: string) {
  deleting.value[id] = true
  try {
    await api.deleteLightApp(id)
    await store.load()
  } finally {
    deleting.value[id] = false
  }
}
</script>

<template>
  <div class="p-6 space-y-6">
    <!-- Apps list -->
    <div class="space-y-3">
      <div class="flex items-center justify-between">
        <h2 class="text-base font-semibold text-neutral-900">Light Apps</h2>
      </div>

      <div v-if="!store.loaded" class="text-sm text-neutral-400">Loading…</div>

      <div v-else-if="store.apps.length === 0" class="text-sm text-neutral-400">
        No light apps yet. Ask the agent to build one using the <code class="font-mono bg-neutral-100 px-1 rounded">build-light-app</code> skill.
      </div>

      <div v-else class="divide-y divide-neutral-100 border border-neutral-200 rounded-lg overflow-hidden">
        <div
          v-for="app in store.apps"
          :key="app.id"
          class="flex items-center gap-3 px-4 py-3 bg-white hover:bg-neutral-50 transition-colors"
        >
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium text-neutral-900 truncate">{{ app.name }}</span>
              <span
                class="shrink-0 text-xs px-1.5 py-0.5 rounded font-mono"
                :class="app.runtime === 'python'
                  ? 'bg-blue-50 text-blue-700'
                  : 'bg-amber-50 text-amber-700'"
              >{{ app.runtime }}</span>
              <span
                class="shrink-0 text-xs px-1.5 py-0.5 rounded"
                :class="app.status === 'running'
                  ? 'bg-green-50 text-green-700'
                  : app.status === 'error'
                    ? 'bg-red-50 text-red-700'
                    : 'bg-neutral-100 text-neutral-500'"
              >{{ app.status }}</span>
            </div>
            <p v-if="app.description" class="text-xs text-neutral-400 truncate mt-0.5">{{ app.description }}</p>
          </div>

          <div class="shrink-0 flex items-center gap-2">
            <button
              v-if="app.status === 'running'"
              class="text-xs px-2.5 py-1 rounded border border-neutral-200 text-neutral-700 hover:bg-neutral-100 transition-colors"
              @click="open(app.id)"
            >Open</button>
            <button
              v-if="app.status === 'running'"
              class="text-xs px-2.5 py-1 rounded border border-neutral-200 text-neutral-600 hover:bg-neutral-100 transition-colors"
              @click="stop(app.id)"
            >Stop</button>
            <button
              v-if="app.status !== 'running'"
              class="text-xs px-2.5 py-1 rounded border border-neutral-200 text-neutral-700 hover:bg-neutral-100 transition-colors disabled:opacity-50"
              :disabled="launching[app.id]"
              @click="launch(app.id)"
            >{{ launching[app.id] ? 'Launching…' : 'Launch' }}</button>
            <button
              class="text-xs px-2.5 py-1 rounded border border-red-100 text-red-500 hover:bg-red-50 transition-colors disabled:opacity-50"
              :disabled="deleting[app.id]"
              @click="remove(app.id)"
            >Delete</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Global env vars -->
    <div class="space-y-3">
      <div>
        <h3 class="text-sm font-semibold text-neutral-900">Environment Variables</h3>
        <p class="text-xs text-neutral-400 mt-0.5">Injected at launch — static: <code class="font-mono">window.swiflow.env('KEY')</code> (throws if missing); Python: <code class="font-mono">os.environ['KEY']</code>. Restart after changing.</p>
      </div>

      <div v-if="Object.keys(env).length > 0" class="divide-y divide-neutral-100 border border-neutral-200 rounded-lg overflow-hidden">
        <div
          v-for="(value, key) in env"
          :key="key"
          class="flex items-center gap-3 px-4 py-2.5 bg-white hover:bg-neutral-50 transition-colors"
        >
          <span class="text-xs font-mono text-neutral-700 w-40 shrink-0 truncate">{{ key }}</span>
          <span class="text-xs font-mono text-neutral-400 flex-1 truncate">{{ value }}</span>
          <button
            class="shrink-0 text-xs px-2 py-0.5 rounded border border-red-100 text-red-500 hover:bg-red-50 transition-colors"
            @click="removeEnv(key)"
          >Remove</button>
        </div>
      </div>
      <p v-else class="text-xs text-neutral-400">No environment variables set.</p>

      <!-- Add row -->
      <div class="flex items-center gap-2">
        <input
          v-model="newKey"
          placeholder="KEY"
          class="text-xs font-mono border border-neutral-200 rounded px-2.5 py-1.5 w-36 focus:outline-none focus:ring-1 focus:ring-neutral-300"
          @keydown.enter="addEnv"
        />
        <input
          v-model="newValue"
          placeholder="value"
          class="text-xs font-mono border border-neutral-200 rounded px-2.5 py-1.5 flex-1 focus:outline-none focus:ring-1 focus:ring-neutral-300"
          @keydown.enter="addEnv"
        />
        <button
          class="text-xs px-3 py-1.5 rounded border border-neutral-200 text-neutral-700 hover:bg-neutral-100 transition-colors disabled:opacity-50"
          :disabled="savingEnv || !newKey.trim()"
          @click="addEnv"
        >Add</button>
      </div>
    </div>
  </div>
</template>
