<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useProvidersStore } from '../stores/providers'
import { api } from '../api'
import type { Provider } from '../types'

const providersStore = useProvidersStore()
const error = ref('')
const showForm = ref(false)
const editingId = ref('')
const form = ref({ name: '', display_name: '', api_base: 'https://api.openai.com/v1', api_key: '', enabled: true })

onMounted(load)
async function load() {
  try { await providersStore.load() }
  catch (e: any) { error.value = e.message }
}
async function create() {
  try {
    await api.createProvider(form.value)
    showForm.value = false
    resetForm()
    await load()
  } catch (e: any) { error.value = e.message }
}
function resetForm() {
  form.value = { name: '', display_name: '', api_base: 'https://api.openai.com/v1', api_key: '', enabled: true }
}
async function remove(id: string) {
  try { await api.deleteProvider(id); await load() }
  catch (e: any) { error.value = e.message }
}
function startEdit(p: Provider) {
  editingId.value = p.id
  form.value = {
    name: p.name,
    display_name: p.display_name || '',
    api_base: p.api_base,
    api_key: '',
    enabled: p.enabled,
  }
}
async function saveEdit() {
  const body: Record<string, unknown> = {
    display_name: form.value.display_name,
    api_base: form.value.api_base,
    enabled: form.value.enabled,
  }
  if (form.value.api_key) body.api_key = form.value.api_key
  try {
    await api.updateProvider(editingId.value, body)
    editingId.value = ''
    resetForm()
    await load()
  } catch (e: any) { error.value = e.message }
}
function cancelEdit() {
  editingId.value = ''
  resetForm()
}
</script>

<template>
  <div class="p-6 max-w-[960px] mx-auto">
    <div class="flex justify-between items-center mb-4">
      <h1 class="text-xl font-bold">Providers</h1>
      <button class="px-3 py-1 bg-neutral-800 text-white rounded text-sm" @click="showForm = !showForm">+ New</button>
    </div>
    <div v-if="error" class="text-red-600 mb-2">{{ error }}</div>
    <div v-if="showForm" class="border border-neutral-200 rounded p-4 mb-4 bg-white space-y-2">
      <input v-model="form.name" placeholder="name (e.g. openai)" class="w-full border rounded px-2 py-1" />
      <input v-model="form.display_name" placeholder="display name" class="w-full border rounded px-2 py-1" />
      <input v-model="form.api_base" placeholder="api_base" class="w-full border rounded px-2 py-1" />
      <input v-model="form.api_key" placeholder="api_key" type="password" class="w-full border rounded px-2 py-1" />
      <button class="px-3 py-1 bg-neutral-800 text-white rounded text-sm" @click="create">Create</button>
    </div>
    <div class="space-y-2">
      <div v-for="p in providersStore.providers" :key="p.id" class="border border-neutral-200 rounded p-3 bg-white">
        <template v-if="editingId === p.id">
          <input v-model="form.display_name" class="w-full border rounded px-2 py-1 mb-1" />
          <input v-model="form.api_base" class="w-full border rounded px-2 py-1 mb-1" />
          <input v-model="form.api_key" type="password" placeholder="new api_key (optional)" class="w-full border rounded px-2 py-1 mb-1" />
          <label class="text-sm flex items-center gap-2 mb-1">
            <input v-model="form.enabled" type="checkbox" /> enabled
          </label>
          <div class="flex gap-2">
            <button class="px-3 py-1 bg-neutral-800 text-white rounded text-sm" @click="saveEdit">Save</button>
            <button class="px-3 py-1 border rounded text-sm" @click="cancelEdit">Cancel</button>
          </div>
        </template>
        <template v-else>
          <div class="flex justify-between">
            <div>
              <div class="font-mono font-semibold">{{ p.name }} <span v-if="!p.enabled" class="text-red-600 text-xs">(disabled)</span></div>
              <div class="text-sm text-neutral-600">{{ p.api_base }}</div>
            </div>
            <div class="flex gap-2">
              <button class="text-sm text-neutral-600" @click="startEdit(p)">edit</button>
              <button class="text-red-600 text-sm" @click="remove(p.id)">delete</button>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>
