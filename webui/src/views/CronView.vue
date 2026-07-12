<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useCronStore } from '../stores/cron'
import { useAgentsStore } from '../stores/agents'
import { api } from '../api'
import type { CronJob } from '../types'

const cronStore = useCronStore()
const agentsStore = useAgentsStore()
const error = ref('')
const showForm = ref(false)
const editingId = ref('')
const form = ref({
  name: '',
  agent_key: 'default',
  message: '',
  schedule: '@hourly',
  enabled: true,
})

onMounted(async () => {
  try {
    await Promise.all([cronStore.load(), agentsStore.load()])
  } catch (e: any) { error.value = e.message }
})
async function load() {
  try { await cronStore.load() }
  catch (e: any) { error.value = e.message }
}
function resetForm() {
  form.value = { name: '', agent_key: 'default', message: '', schedule: '@hourly', enabled: true }
}
async function create() {
  try {
    await api.createCronJob(form.value)
    showForm.value = false
    resetForm()
    await load()
  } catch (e: any) { error.value = e.message }
}
function startEdit(j: CronJob) {
  editingId.value = j.id
  form.value = {
    name: j.name,
    agent_key: j.agent_key,
    message: j.message,
    schedule: j.schedule,
    enabled: j.enabled,
  }
}
async function saveEdit() {
  try {
    await api.updateCronJob(editingId.value, {
      agent_key: form.value.agent_key,
      message: form.value.message,
      schedule: form.value.schedule,
      enabled: form.value.enabled,
    })
    editingId.value = ''
    resetForm()
    await load()
  } catch (e: any) { error.value = e.message }
}
async function remove(id: string) {
  try { await api.deleteCronJob(id); await load() }
  catch (e: any) { error.value = e.message }
}
async function reload() {
  try { await api.reloadCron(); await load() }
  catch (e: any) { error.value = e.message }
}
</script>

<template>
  <div class="p-6 max-w-[960px] mx-auto">
    <div class="flex justify-between items-center mb-4">
      <h1 class="text-xl font-bold">Cron Jobs</h1>
      <div class="flex gap-2">
        <button class="px-3 py-1 bg-neutral-200 rounded text-sm" @click="reload">Reload</button>
        <button class="px-3 py-1 bg-neutral-800 text-white rounded text-sm" @click="showForm = !showForm">+ New</button>
      </div>
    </div>
    <p class="text-sm text-neutral-500 mb-4">Schedule agent runs with cron expressions (e.g. <code class="text-xs">0 9 * * *</code>, <code class="text-xs">@hourly</code>).</p>
    <div v-if="error" class="text-red-600 mb-2">{{ error }}</div>
    <div v-if="showForm" class="border border-neutral-200 rounded p-4 mb-4 bg-white space-y-2">
      <input v-model="form.name" placeholder="name" class="w-full border rounded px-2 py-1" />
      <select v-model="form.agent_key" class="w-full border rounded px-2 py-1">
        <option v-for="a in agentsStore.agents" :key="a.key" :value="a.key">{{ a.key }}</option>
      </select>
      <textarea v-model="form.message" placeholder="message to send to agent" class="w-full border rounded px-2 py-1" rows="3"></textarea>
      <input v-model="form.schedule" placeholder="schedule (cron or @hourly)" class="w-full border rounded px-2 py-1" />
      <button class="px-3 py-1 bg-neutral-800 text-white rounded text-sm" @click="create">Create</button>
    </div>
    <div class="space-y-2">
      <div v-for="j in cronStore.jobs" :key="j.id" class="border border-neutral-200 rounded p-3 bg-white">
        <template v-if="editingId === j.id">
          <select v-model="form.agent_key" class="w-full border rounded px-2 py-1 mb-1">
            <option v-for="a in agentsStore.agents" :key="a.key" :value="a.key">{{ a.key }}</option>
          </select>
          <textarea v-model="form.message" class="w-full border rounded px-2 py-1 mb-1" rows="2"></textarea>
          <input v-model="form.schedule" class="w-full border rounded px-2 py-1 mb-1" />
          <label class="text-sm flex items-center gap-2 mb-1">
            <input v-model="form.enabled" type="checkbox" /> enabled
          </label>
          <div class="flex gap-2">
            <button class="px-3 py-1 bg-neutral-800 text-white rounded text-sm" @click="saveEdit">Save</button>
            <button class="px-3 py-1 border rounded text-sm" @click="editingId = ''">Cancel</button>
          </div>
        </template>
        <template v-else>
          <div class="flex justify-between">
            <div>
              <div class="font-mono font-semibold">{{ j.name }}
                <span v-if="!j.enabled" class="text-red-600 text-xs">(disabled)</span>
              </div>
              <div class="text-sm text-neutral-600">{{ j.agent_key }} · <span class="font-mono">{{ j.schedule }}</span></div>
              <div class="text-xs text-neutral-500 truncate">{{ j.message }}</div>
              <div v-if="j.last_run_at" class="text-xs text-neutral-400">last run: {{ j.last_run_at }}</div>
            </div>
            <div class="flex gap-2">
              <button class="text-sm text-neutral-600" @click="startEdit(j)">edit</button>
              <button class="text-red-600 text-sm" @click="remove(j.id)">delete</button>
            </div>
          </div>
        </template>
      </div>
      <div v-if="!cronStore.jobs.length" class="text-neutral-500 text-sm">No cron jobs configured.</div>
    </div>
  </div>
</template>
