<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import * as pdfjs from 'pdfjs-dist'

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  'pdfjs-dist/build/pdf.worker.min.mjs',
  import.meta.url,
).href

const props = defineProps<{ data: ArrayBuffer }>()

const pages = ref<string[]>([])
const loading = ref(true)
const error = ref('')

async function renderPdf() {
  loading.value = true
  error.value = ''
  pages.value = []
  try {
    const doc = await pdfjs.getDocument({ data: new Uint8Array(props.data) }).promise
    const rendered: string[] = []
    for (let i = 1; i <= doc.numPages; i++) {
      const page = await doc.getPage(i)
      const viewport = page.getViewport({ scale: 1.25 })
      const canvas = document.createElement('canvas')
      const ctx = canvas.getContext('2d')
      if (!ctx) continue
      canvas.width = viewport.width
      canvas.height = viewport.height
      await page.render({ canvas, canvasContext: ctx, viewport }).promise
      rendered.push(canvas.toDataURL('image/png'))
    }
    pages.value = rendered
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'failed to render PDF'
  } finally {
    loading.value = false
  }
}

onMounted(renderPdf)
watch(() => props.data, renderPdf)
</script>

<template>
  <div class="h-full overflow-y-auto bg-neutral-100 p-4">
    <div v-if="loading" class="text-neutral-400">Loading PDF…</div>
    <div v-else-if="error" class="text-red-600">{{ error }}</div>
    <div v-else class="flex flex-col items-center gap-4">
      <img
        v-for="(src, i) in pages"
        :key="i"
        :src="src"
        :alt="`Page ${i + 1}`"
        class="max-w-full shadow-sm bg-white"
      />
    </div>
  </div>
</template>
