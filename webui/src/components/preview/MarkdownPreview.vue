<script setup lang="ts">
import { computed } from 'vue'
import { renderMarkdown } from '../../lib/markdown'
import TextPreview from './TextPreview.vue'

const props = defineProps<{
  content: string
  path: string
  mode: 'preview' | 'source'
}>()

const html = computed(() =>
  props.mode === 'preview' ? renderMarkdown(props.content) : '',
)
</script>

<template>
  <div class="h-full min-h-0 bg-white">
    <div v-if="mode === 'preview'" class="h-full overflow-y-auto px-6 py-4">
      <article class="prose-swiflow w-full max-w-[960px] mx-auto" v-html="html" />
    </div>
    <div v-else class="h-full overflow-hidden">
      <TextPreview :path="path" :content="content" />
    </div>
  </div>
</template>
