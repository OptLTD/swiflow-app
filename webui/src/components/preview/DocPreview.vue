<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import mammoth from 'mammoth'
import { fileExtension } from '../../lib/filePreview'

const props = defineProps<{ data: ArrayBuffer; path: string }>()

const html = ref('')
const loading = ref(true)
const error = ref('')

async function renderDoc() {
  loading.value = true
  error.value = ''
  html.value = ''
  const ext = fileExtension(props.path)
  if (ext === 'doc') {
    error.value = '旧版 .doc 格式暂不支持预览，请转换为 .docx'
    loading.value = false
    return
  }
  try {
    const result = await mammoth.convertToHtml({ arrayBuffer: props.data.slice(0) })
    html.value = result.value
    if (result.messages.length) {
      console.warn('docx preview warnings', result.messages)
    }
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'failed to render document'
  } finally {
    loading.value = false
  }
}

onMounted(renderDoc)
watch(() => [props.data, props.path], renderDoc)
</script>

<template>
  <div class="h-full overflow-y-auto bg-white p-6">
    <div v-if="loading" class="text-neutral-400">Loading document…</div>
    <div v-else-if="error" class="text-red-600">{{ error }}</div>
    <article v-else class="doc-preview prose prose-neutral max-w-none text-sm" v-html="html" />
  </div>
</template>

<style scoped>
.doc-preview :deep(p) {
  margin: 0.5em 0;
}
.doc-preview :deep(h1),
.doc-preview :deep(h2),
.doc-preview :deep(h3) {
  margin: 1em 0 0.5em;
  font-weight: 600;
}
.doc-preview :deep(table) {
  border-collapse: collapse;
  width: 100%;
}
.doc-preview :deep(td),
.doc-preview :deep(th) {
  border: 1px solid #e5e5e5;
  padding: 0.35rem 0.5rem;
}
</style>
