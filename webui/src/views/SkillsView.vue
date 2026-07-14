<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSkillsStore } from '../stores/skills'
import { api } from '../api'
import type { SkillDraft } from '../types'

const skillsStore = useSkillsStore()
const error = ref('')
const drafts = ref<SkillDraft[]>([])
const preview = ref<SkillDraft | null>(null)

onMounted(load)
async function load() {
  try {
    await skillsStore.load()
    const r = await api.listSkillDrafts()
    drafts.value = r.drafts || []
  } catch (e: any) {
    error.value = e.message
  }
}
async function toggle(slug: string, enabled: boolean) {
  try {
    await api.setSkill(slug, !enabled)
    await load()
  } catch (e: any) {
    error.value = e.message
  }
}
async function reload() {
  try {
    await api.reloadSkills()
    await load()
  } catch (e: any) {
    error.value = e.message
  }
}
async function acceptDraft(id: string) {
  try {
    await api.acceptSkillDraft(id)
    preview.value = null
    await load()
  } catch (e: any) {
    error.value = e.message
  }
}
async function rejectDraft(id: string) {
  try {
    await api.deleteSkillDraft(id)
    preview.value = null
    await load()
  } catch (e: any) {
    error.value = e.message
  }
}
</script>

<template>
  <div class="p-6 max-w-[960px] mx-auto">
    <div class="flex justify-between items-center mb-4">
      <h1 class="text-xl font-bold">Skills</h1>
      <button class="px-3 py-1 bg-neutral-200 rounded text-sm" @click="reload">Reload</button>
    </div>
    <div v-if="error" class="text-red-600 mb-2">{{ error }}</div>

    <div v-if="drafts.length" class="mb-8">
      <h2 class="text-sm font-semibold text-neutral-500 uppercase tracking-wide mb-3">Pending drafts</h2>
      <div class="space-y-2">
        <div
          v-for="d in drafts"
          :key="d.id"
          class="border border-amber-200 bg-amber-50/50 rounded p-3 flex justify-between gap-3"
        >
          <div class="min-w-0">
            <div class="font-mono font-semibold">{{ d.slug }}</div>
            <div class="text-xs text-neutral-500">{{ d.note || 'No note' }} · {{ d.created_at }}</div>
          </div>
          <div class="flex gap-2 shrink-0">
            <button class="text-sm text-neutral-600 hover:underline" @click="preview = d">Preview</button>
            <button class="text-sm text-green-700" @click="acceptDraft(d.id)">Accept</button>
            <button class="text-sm text-red-600" @click="rejectDraft(d.id)">Reject</button>
          </div>
        </div>
      </div>
      <div v-if="preview" class="mt-3 border border-neutral-200 rounded p-3 bg-white">
        <div class="text-xs text-neutral-500 mb-2">Draft {{ preview.id }}</div>
        <pre class="text-xs whitespace-pre-wrap max-h-64 overflow-auto">{{ preview.content }}</pre>
      </div>
    </div>

    <div class="space-y-2">
      <div v-for="s in skillsStore.skills" :key="s.slug" class="border border-neutral-200 rounded p-3 bg-white flex justify-between">
        <div>
          <div class="font-mono font-semibold">{{ s.slug }} <span class="text-xs text-neutral-400">[{{ s.source }}]</span></div>
          <div class="text-sm text-neutral-600">{{ s.name }} — {{ s.description }}</div>
        </div>
        <button class="text-sm" :class="s.enabled ? 'text-red-600' : 'text-green-600'" @click="toggle(s.slug, s.enabled)">
          {{ s.enabled ? 'disable' : 'enable' }}
        </button>
      </div>
      <div v-if="!skillsStore.skills.length" class="text-neutral-500 text-sm">No skills discovered.</div>
    </div>
  </div>
</template>
