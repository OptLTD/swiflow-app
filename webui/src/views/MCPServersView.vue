<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useMCPStore } from '../stores/mcp'
import { api } from '../api'
import type { MCPServer } from '../types'
import MCPDialog from '../components/MCPDialog.vue'

const mcpStore = useMCPStore()
const error = ref('')
const showForm = ref(false)
const editingId = ref('')
const capsServer = ref<MCPServer | null>(null)
const capsOpen = ref(false)
const form = ref({
  name: '',
  type: 'stdio' as MCPServer['type'],
  cmd: '',
  argsText: '',
  url: '',
  enabled: true,
})

onMounted(load)
async function load() {
  try { await mcpStore.load() }
  catch (e: any) { error.value = e.message }
}
function parseArgs(): string[] {
  return form.value.argsText.split('\n').map((s) => s.trim()).filter(Boolean)
}
function resetForm() {
  form.value = { name: '', type: 'stdio', cmd: '', argsText: '', url: '', enabled: true }
}
async function create() {
  try {
    await api.createMCPServer({
      name: form.value.name,
      type: form.value.type,
      cmd: form.value.cmd,
      args: parseArgs(),
      url: form.value.url,
      enabled: form.value.enabled,
    })
    showForm.value = false
    resetForm()
    await load()
  } catch (e: any) { error.value = e.message }
}
function startEdit(s: MCPServer) {
  editingId.value = s.id
  form.value = {
    name: s.name,
    type: s.type,
    cmd: s.cmd || '',
    argsText: (s.args || []).join('\n'),
    url: s.url || '',
    enabled: s.enabled,
  }
}
async function saveEdit() {
  try {
    await api.updateMCPServer(editingId.value, {
      type: form.value.type,
      cmd: form.value.cmd,
      args: parseArgs(),
      url: form.value.url,
      enabled: form.value.enabled,
    })
    editingId.value = ''
    resetForm()
    await load()
  } catch (e: any) { error.value = e.message }
}
async function remove(id: string) {
  try { await api.deleteMCPServer(id); await load() }
  catch (e: any) { error.value = e.message }
}
async function reload() {
  try { await api.reloadMCP(); await load() }
  catch (e: any) { error.value = e.message }
}
function openCapabilities(s: MCPServer) {
  capsServer.value = s
  capsOpen.value = true
}
function closeCapabilities() {
  capsOpen.value = false
  capsServer.value = null
}
</script>

<template>
  <div class="p-6 max-w-[960px] mx-auto">
    <div class="flex justify-between items-center mb-4">
      <h1 class="text-xl font-bold">MCP Servers</h1>
      <div class="flex gap-2">
        <button class="px-3 py-1 bg-neutral-200 rounded text-sm" @click="reload">Reload</button>
        <button class="px-3 py-1 bg-neutral-800 text-white rounded text-sm" @click="showForm = !showForm">+ New</button>
      </div>
    </div>
    <p class="text-sm text-neutral-500 mb-4">Connect MCP servers and inspect their tools, resources, and templates per server.</p>
    <div v-if="error" class="text-red-600 mb-2">{{ error }}</div>
    <div v-if="showForm" class="border border-neutral-200 rounded p-4 mb-4 bg-white space-y-2">
      <input v-model="form.name" placeholder="name (unique)" class="w-full border rounded px-2 py-1" />
      <select v-model="form.type" class="w-full border rounded px-2 py-1">
        <option value="stdio">stdio</option>
        <option value="sse">sse</option>
        <option value="streamable">streamable</option>
      </select>
      <template v-if="form.type === 'stdio'">
        <input v-model="form.cmd" placeholder="cmd (e.g. npx)" class="w-full border rounded px-2 py-1" />
        <textarea v-model="form.argsText" placeholder="args (one per line)" class="w-full border rounded px-2 py-1" rows="3"></textarea>
      </template>
      <template v-else>
        <input v-model="form.url" placeholder="url" class="w-full border rounded px-2 py-1" />
      </template>
      <button class="px-3 py-1 bg-neutral-800 text-white rounded text-sm" @click="create">Create</button>
    </div>
    <div class="space-y-2">
      <div v-for="s in mcpStore.servers" :key="s.id" class="border border-neutral-200 rounded p-3 bg-white">
        <template v-if="editingId === s.id">
          <select v-model="form.type" class="w-full border rounded px-2 py-1 mb-1">
            <option value="stdio">stdio</option>
            <option value="sse">sse</option>
            <option value="streamable">streamable</option>
          </select>
          <input v-if="form.type === 'stdio'" v-model="form.cmd" class="w-full border rounded px-2 py-1 mb-1" />
          <textarea v-if="form.type === 'stdio'" v-model="form.argsText" class="w-full border rounded px-2 py-1 mb-1" rows="2"></textarea>
          <input v-else v-model="form.url" class="w-full border rounded px-2 py-1 mb-1" />
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
              <div class="font-mono font-semibold">{{ s.name }}
                <span v-if="!s.enabled" class="text-red-600 text-xs">(disabled)</span>
                <span class="text-xs text-neutral-400 ml-1">{{ s.type }}</span>
              </div>
              <div class="text-xs text-neutral-400 font-mono truncate">{{ s.type === 'stdio' ? s.cmd + ' ' + (s.args || []).join(' ') : s.url }}</div>
            </div>
            <div class="flex gap-2 items-start">
              <button class="text-sm text-blue-600" @click="openCapabilities(s)">capabilities</button>
              <button class="text-sm text-neutral-600" @click="startEdit(s)">edit</button>
              <button class="text-red-600 text-sm" @click="remove(s.id)">delete</button>
            </div>
          </div>
        </template>
      </div>
      <div v-if="!mcpStore.servers.length" class="text-neutral-500 text-sm">No MCP servers configured.</div>
    </div>
    <MCPDialog :open="capsOpen" :server="capsServer" @close="closeCapabilities" />
  </div>
</template>
