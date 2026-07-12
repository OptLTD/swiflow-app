<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAgentsStore } from '../stores/agents'
import { useProvidersStore } from '../stores/providers'
import { api } from '../api'
import type { Agent } from '../types'

const agentsStore = useAgentsStore()
const providersStore = useProvidersStore()
const error = ref('')
const showForm = ref(false)
const editingKey = ref('')
const form = ref({ key: '', display_name: '', provider: '', model: 'gpt-4o-mini', system_extra: '' })

onMounted(load)
async function load() {
  try {
    await Promise.all([agentsStore.load(), providersStore.load()])
  } catch (e: any) { error.value = e.message }
}
async function create() {
  try {
    await api.createAgent(form.value)
    showForm.value = false
    resetForm()
    await load()
  } catch (e: any) { error.value = e.message }
}
function resetForm() {
  form.value = { key: '', display_name: '', provider: '', model: 'gpt-4o-mini', system_extra: '' }
}
function startEdit(a: Agent) {
  editingKey.value = a.key
  form.value = {
    key: a.key,
    display_name: a.display_name || '',
    provider: a.provider,
    model: a.model,
    system_extra: a.system_extra || '',
  }
}
async function saveEdit() {
  try {
    await api.updateAgent(editingKey.value, {
      display_name: form.value.display_name,
      provider: form.value.provider,
      model: form.value.model,
      system_extra: form.value.system_extra,
    })
    editingKey.value = ''
    resetForm()
    await load()
  } catch (e: any) { error.value = e.message }
}
function cancelEdit() {
  editingKey.value = ''
  resetForm()
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
        <option v-for="p in providersStore.providers" :key="p.id" :value="p.name">{{ p.name }}</option>
      </select>
      <input v-model="form.model" placeholder="model" class="w-full border rounded px-2 py-1" />
      <textarea v-model="form.system_extra" placeholder="system_extra (optional)" class="w-full border rounded px-2 py-1" rows="2"></textarea>
      <button class="px-3 py-1 bg-neutral-800 text-white rounded text-sm" @click="create">Create</button>
    </div>
    <div class="space-y-2">
      <div v-for="a in agentsStore.agents" :key="a.id" class="border border-neutral-200 rounded p-3 bg-white">
        <template v-if="editingKey === a.key">
          <input v-model="form.display_name" class="w-full border rounded px-2 py-1 mb-1" placeholder="display name" />
          <select v-model="form.provider" class="w-full border rounded px-2 py-1 mb-1">
            <option v-for="p in providersStore.providers" :key="p.id" :value="p.name">{{ p.name }}</option>
          </select>
          <input v-model="form.model" class="w-full border rounded px-2 py-1 mb-1" />
          <textarea v-model="form.system_extra" class="w-full border rounded px-2 py-1 mb-1" rows="2"></textarea>
          <div class="flex gap-2">
            <button class="px-3 py-1 bg-neutral-800 text-white rounded text-sm" @click="saveEdit">Save</button>
            <button class="px-3 py-1 border rounded text-sm" @click="cancelEdit">Cancel</button>
          </div>
        </template>
        <template v-else>
          <div class="flex justify-between">
            <div>
              <div class="font-mono font-semibold">{{ a.key }}</div>
              <div class="text-sm text-neutral-600">{{ a.display_name || '—' }} · {{ a.provider }} / {{ a.model }}</div>
            </div>
            <button class="text-sm text-neutral-600" @click="startEdit(a)">edit</button>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>
