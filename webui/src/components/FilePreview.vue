<script setup lang="ts">
import { defineAsyncComponent, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'
import { previewKind, type ExcelSheet } from '../lib/filePreview'
import { useToastStore } from '../stores/toast'
import LocalSvgIcon from './LocalSvgIcon.vue'

const TextPreview = defineAsyncComponent(() => import('./preview/TextPreview.vue'))
const MarkdownPreview = defineAsyncComponent(() => import('./preview/MarkdownPreview.vue'))
const ExcelPreview = defineAsyncComponent(() => import('./preview/ExcelPreview.vue'))
const PdfPreview = defineAsyncComponent(() => import('./preview/PdfPreview.vue'))
const DocPreview = defineAsyncComponent(() => import('./preview/DocPreview.vue'))
const ImagePreview = defineAsyncComponent(() => import('./preview/ImagePreview.vue'))

const props = defineProps<{ path: string }>()
const toast = useToastStore()
const { t } = useI18n()

const loading = ref(true)
const error = ref('')
const kind = ref(previewKind(props.path))
const textContent = ref('')
const binaryData = ref<ArrayBuffer | null>(null)
const excelSheets = ref<ExcelSheet[]>([])
const mdMode = ref<'preview' | 'source'>('preview')

async function copyText() {
  const text = textContent.value
  if (!text) {
    toast.error(t('filePreview.nothingToCopy'))
    return
  }
  try {
    await navigator.clipboard.writeText(text)
    toast.success(t('filePreview.copied'))
  } catch {
    toast.error(t('filePreview.copyFailed'))
  }
}

async function load() {
  if (!props.path) return
  loading.value = true
  error.value = ''
  textContent.value = ''
  binaryData.value = null
  excelSheets.value = []
  kind.value = previewKind(props.path)
  if (kind.value === 'markdown') mdMode.value = 'preview'

  try {
    if (kind.value === 'text' || kind.value === 'markdown') {
      const r = await api.readWorkspaceFile(props.path)
      textContent.value = r.content
      if (r.truncated) {
        textContent.value += '\n\n...[truncated]'
      }
      return
    }
    if (kind.value === 'unsupported') {
      error.value = t('filePreview.unsupported')
      return
    }
    const data = await api.downloadWorkspaceFile(props.path)
    if (kind.value === 'excel') {
      const { parseExcel } = await import('../lib/parseExcel')
      excelSheets.value = parseExcel(data)
      return
    }
    binaryData.value = data
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : t('common.failedToLoad')
  } finally {
    loading.value = false
  }
}

watch(() => props.path, load, { immediate: true })
</script>

<template>
  <div class="h-full flex flex-col min-h-0 bg-neutral-50">
    <!-- Path / actions bar (excel has its own sheet-tab actions) -->
    <div
      v-if="kind !== 'excel'"
      class="shrink-0 h-9 border-b border-neutral-200 px-3 flex items-center gap-2 text-xs text-neutral-500"
    >
      <span class="font-mono truncate flex-1 min-w-0">{{ path }}</span>
      <div v-if="kind === 'markdown'" class="shrink-0 flex items-center gap-0.5 mr-0.5">
        <button
          type="button"
          class="px-2 py-0.5 rounded"
          :class="mdMode === 'preview' ? 'bg-neutral-900 text-white' : 'hover:bg-neutral-100 hover:text-neutral-800'"
          @click="mdMode = 'preview'"
        >
          Preview
        </button>
        <button
          type="button"
          class="px-2 py-0.5 rounded"
          :class="mdMode === 'source' ? 'bg-neutral-900 text-white' : 'hover:bg-neutral-100 hover:text-neutral-800'"
          @click="mdMode = 'source'"
        >
          Source
        </button>
      </div>
      <button
        v-if="kind === 'text' || kind === 'markdown'"
        type="button"
        class="shrink-0 w-7 h-7 flex items-center justify-center rounded hover:bg-neutral-100 hover:text-neutral-800"
        :title="t('common.copy')"
        :disabled="loading || !textContent"
        @click="copyText"
      >
        <LocalSvgIcon name="copy" :size="15" />
      </button>
      <button
        type="button"
        class="shrink-0 w-7 h-7 flex items-center justify-center rounded hover:bg-neutral-100 hover:text-neutral-800"
        :title="t('common.refresh')"
        :disabled="loading"
        @click="load"
      >
        <LocalSvgIcon name="refresh" :size="15" />
      </button>
    </div>
    <div class="flex-1 min-h-0 overflow-hidden">
      <div v-if="loading" class="p-6 text-neutral-400">{{ t('common.loading') }}</div>
      <div v-else-if="error" class="p-6 text-red-600">{{ error }}</div>
      <MarkdownPreview
        v-else-if="kind === 'markdown'"
        :path="path"
        :content="textContent"
        :mode="mdMode"
      />
      <TextPreview v-else-if="kind === 'text'" :path="path" :content="textContent" />
      <ExcelPreview
        v-else-if="kind === 'excel'"
        :path="path"
        :sheets="excelSheets"
        @refresh="load"
      />
      <PdfPreview v-else-if="kind === 'pdf' && binaryData" :data="binaryData" />
      <DocPreview v-else-if="kind === 'doc' && binaryData" :path="path" :data="binaryData" />
      <ImagePreview v-else-if="kind === 'image' && binaryData" :path="path" :data="binaryData" />
    </div>
  </div>
</template>
