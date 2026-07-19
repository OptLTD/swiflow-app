<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'
import { useLayoutStore } from '../stores/layout'
import { useUploadStore } from '../stores/upload'
import LocalSvgIcon from '../components/LocalSvgIcon.vue'
import type { WorkspaceEntry } from '../types'

const props = defineProps<{ path?: string }>()
const layout = useLayoutStore()
const upload = useUploadStore()
const { t } = useI18n()

const entries = ref<WorkspaceEntry[]>([])
const currentPath = ref('.')
const loading = ref(true)
const error = ref('')

async function loadEntries(path: string) {
  loading.value = true
  error.value = ''
  try {
    const r = await api.listWorkspace(path)
    currentPath.value = r.path || path
    layout.setExplorePath(currentPath.value)
    entries.value = r.entries || []
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'failed to load directory'
    entries.value = []
  } finally {
    loading.value = false
  }
}

watch(
  () => props.path,
  (path) => {
    loadEntries(path || '.')
  },
  { immediate: true },
)

watch(
  () => upload.refreshSeq,
  () => {
    loadEntries(currentPath.value)
  },
)

function openEntry(entry: WorkspaceEntry) {
  if (entry.is_dir) {
    layout.openExplore(entry.path)
    return
  }
  layout.openFile(entry.path)
}
</script>

<template>
  <div class="h-full flex flex-col bg-white">
    <!-- Path bar -->
    <div class="shrink-0 h-9 border-b border-neutral-200 px-3 flex items-center gap-2 text-sm">
      <LocalSvgIcon name="folder-open" class="text-neutral-400 shrink-0" />
      <span class="font-mono text-neutral-700 truncate flex-1 min-w-0">{{ currentPath }}</span>
      <button
        type="button"
        class="cursor-pointer shrink-0 w-7 h-7 flex items-center justify-center rounded text-neutral-500 hover:bg-neutral-100 hover:text-neutral-800 disabled:cursor-default disabled:opacity-50"
        :title="t('explore.refresh')"
        :disabled="loading"
        @click="loadEntries(currentPath)"
      >
        <LocalSvgIcon name="refresh" :size="15" />
      </button>
    </div>

    <!-- File list -->
    <div class="flex-1 overflow-y-auto">
      <div v-if="loading" class="p-6 text-neutral-400">Loading…</div>
      <div v-else-if="error" class="p-6 text-red-600">{{ error }}</div>
      <div v-else-if="!entries.length" class="p-6 text-neutral-400">
        Empty directory
      </div>
      <div v-else>
        <button
          v-for="entry in entries"
          :key="entry.path + entry.name"
          type="button"
          class="cursor-pointer w-full px-4 py-1.5 flex items-center gap-2.5 text-sm hover:bg-neutral-50 border-b border-neutral-100"
          @click="openEntry(entry)"
        >
          <LocalSvgIcon :name="entry.is_dir ? 'folder' : 'file'" class="text-neutral-500" />
          <span class="truncate" :class="entry.is_dir ? 'font-medium' : ''">{{ entry.name }}</span>
          <span v-if="entry.is_dir && entry.name !== '..'" class="text-neutral-400">/</span>
          <span v-else-if="entry.size != null" class="ml-auto text-xs text-neutral-400 tabular-nums">
            {{ entry.size < 1024 ? entry.size + ' B' : Math.round(entry.size / 1024) + ' KB' }}
          </span>
        </button>
      </div>
    </div>
  </div>
</template>
