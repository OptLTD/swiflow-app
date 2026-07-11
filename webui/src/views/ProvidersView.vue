<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api'

const providers = ref<any[]>([])
const error = ref('')
const showForm = ref(false)
const form = ref({ name: '', display_name: '', api_base: 'https://api.openai.com/v1', api_key: '', enabled: true })

onMounted(load)
async function load() {
  try { providers.value = (await api.listProviders()).providers || [] }
  catch (e: any) { error.value = e.message }
}
async function create() {
  try {
    await api.createProvider(form.value)
    showForm.value = false
    form.value = { name: '', display_name: '', api_base: 'https://api.openai.com/v1', api_key: '', enabled: true }
    await load()
  } catch (e: any) { error.value = e.message }
}
async function remove(id: string) {
  try { await api.deleteProvider(id); await load() }
  catch (e: any) { error.value = e.message }
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
      <div v-for="p in providers" :key="p.id" class="border border-neutral-200 rounded p-3 bg-white flex justify-between">
        <div>
          <div class="font-mono font-semibold">{{ p.name }} <span v-if="!p.enabled" class="text-red-600 text-xs">(disabled)</span></div>
          <div class="text-sm text-neutral-600">{{ p.api_base }}</div>
        </div>
        <button class="text-red-600 text-sm" @click="remove(p.id)">delete</button>
      </div>
    </div>
  </div>
</template>
