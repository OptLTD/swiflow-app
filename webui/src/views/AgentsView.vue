<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api'

const agents = ref<any[]>([])
const providers = ref<any[]>([])
const error = ref('')
const showForm = ref(false)
const form = ref({ key: '', display_name: '', provider: '', model: 'gpt-4o-mini', system_extra: '' })

onMounted(load)
async function load() {
  try {
    const [a, p] = await Promise.all([api.listAgents(), api.listProviders()])
    agents.value = a.agents || []
    providers.value = p.providers || []
  } catch (e: any) { error.value = e.message }
}
async function create() {
  try {
    await api.createAgent(form.value)
    showForm.value = false
    form.value = { key: '', display_name: '', provider: '', model: 'gpt-4o-mini', system_extra: '' }
    await load()
  } catch (e: any) { error.value = e.message }
}
</script>

<template>
  <div class="p-6 max-w-[960px] mx-auto">
    <div class="flex justify-between items-center mb-4">
      <h1 class="text-xl font-bold">Agents</h1>
      <button class="px-3 py-1 bg-neutral-800 text-white rounded text-sm" @click="showForm = !showForm">+ New</button>
    </div>
    <div v-if="error" class="text-red-600 mb-2">{{ error }}</div>
    <div v-if="showForm" class="border border-neutral-200 rounded p-4 mb-4 bg-white space-y-2">
      <input v-model="form.key" placeholder="key (e.g. default)" class="w-full border rounded px-2 py-1" />
      <input v-model="form.display_name" placeholder="display name" class="w-full border rounded px-2 py-1" />
      <select v-model="form.provider" class="w-full border rounded px-2 py-1">
        <option value="" disabled>select provider</option>
        <option v-for="p in providers" :key="p.id" :value="p.name">{{ p.name }}</option>
      </select>
      <input v-model="form.model" placeholder="model" class="w-full border rounded px-2 py-1" />
      <textarea v-model="form.system_extra" placeholder="system_extra (optional)" class="w-full border rounded px-2 py-1" rows="2"></textarea>
      <button class="px-3 py-1 bg-neutral-800 text-white rounded text-sm" @click="create">Create</button>
    </div>
    <div class="space-y-2">
      <div v-for="a in agents" :key="a.id" class="border border-neutral-200 rounded p-3 bg-white">
        <div class="font-mono font-semibold">{{ a.key }}</div>
        <div class="text-sm text-neutral-600">{{ a.display_name || '—' }} · {{ a.provider }} / {{ a.model }}</div>
      </div>
    </div>
  </div>
</template>
