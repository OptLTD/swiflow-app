<script setup lang="ts">
import { ref, watch } from 'vue'
import { api } from '../api'
import { previewKind, type ExcelSheet } from '../lib/filePreview'
import { parseExcel } from '../lib/parseExcel'
import TextPreview from './preview/TextPreview.vue'
import ExcelPreview from './preview/ExcelPreview.vue'
import PdfPreview from './preview/PdfPreview.vue'
import DocPreview from './preview/DocPreview.vue'

const props = defineProps<{ path: string }>()

const loading = ref(true)
const error = ref('')
const kind = ref(previewKind(props.path))
const textContent = ref('')
const binaryData = ref<ArrayBuffer | null>(null)
const excelSheets = ref<ExcelSheet[]>([])

async function load() {
  if (!props.path) return
  loading.value = true
  error.value = ''
  textContent.value = ''
  binaryData.value = null
  excelSheets.value = []
  kind.value = previewKind(props.path)

  try {
    if (kind.value === 'text') {
      const r = await api.readWorkspaceFile(props.path)
      textContent.value = r.content
      if (r.truncated) {
        textContent.value += '\n\n...[truncated]'
      }
      return
    }
    if (kind.value === 'unsupported') {
      error.value = '此文件类型暂不支持预览'
      return
    }
    const data = await api.downloadWorkspaceFile(props.path)
    if (kind.value === 'excel') {
      excelSheets.value = parseExcel(data)
      return
    }
    binaryData.value = data
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'failed to load file'
  } finally {
    loading.value = false
  }
}

watch(() => props.path, load, { immediate: true })
</script>

<template>
  <div class="h-full flex flex-col min-h-0 bg-neutral-50">
    <div
      v-if="kind !== 'excel'"
      class="shrink-0 h-9 border-b border-neutral-200 px-4 flex items-center text-xs text-neutral-500 font-mono truncate"
    >
      {{ path }}
    </div>
    <div class="flex-1 min-h-0 overflow-hidden">
      <div v-if="loading" class="p-6 text-neutral-400">Loading…</div>
      <div v-else-if="error" class="p-6 text-red-600">{{ error }}</div>
      <TextPreview v-else-if="kind === 'text'" :path="path" :content="textContent" />
      <ExcelPreview
        v-else-if="kind === 'excel'"
        :path="path"
        :sheets="excelSheets"
        @refresh="load"
      />
      <PdfPreview v-else-if="kind === 'pdf' && binaryData" :data="binaryData" />
      <DocPreview v-else-if="kind === 'doc' && binaryData" :path="path" :data="binaryData" />
    </div>
  </div>
</template>
