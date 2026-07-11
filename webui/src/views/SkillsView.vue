<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api'

const skills = ref<any[]>([])
const error = ref('')

onMounted(load)
async function load() {
  try { skills.value = (await api.listSkills()).skills || [] }
  catch (e: any) { error.value = e.message }
}
async function toggle(slug: string, enabled: boolean) {
  try { await api.setSkill(slug, !enabled); await load() }
  catch (e: any) { error.value = e.message }
}
async function reload() {
  try { await api.reloadSkills(); await load() }
  catch (e: any) { error.value = e.message }
}
</script>

<template>
  <div class="p-6 max-w-[960px] mx-auto">
    <div class="flex justify-between items-center mb-4">
      <h1 class="text-xl font-bold">Skills</h1>
      <button class="px-3 py-1 bg-neutral-200 rounded text-sm" @click="reload">Reload</button>
    </div>
    <div v-if="error" class="text-red-600 mb-2">{{ error }}</div>
    <div class="space-y-2">
      <div v-for="s in skills" :key="s.slug" class="border border-neutral-200 rounded p-3 bg-white flex justify-between">
        <div>
          <div class="font-mono font-semibold">{{ s.slug }} <span class="text-xs text-neutral-400">[{{ s.source }}]</span></div>
          <div class="text-sm text-neutral-600">{{ s.name }} — {{ s.description }}</div>
        </div>
        <button class="text-sm" :class="s.enabled ? 'text-red-600' : 'text-green-600'" @click="toggle(s.slug, s.enabled)">
          {{ s.enabled ? 'disable' : 'enable' }}
        </button>
      </div>
      <div v-if="!skills.length" class="text-neutral-500 text-sm">No skills discovered.</div>
    </div>
  </div>
</template>
