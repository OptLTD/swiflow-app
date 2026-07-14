<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { imageMimeType } from '../../lib/filePreview'

const props = defineProps<{ path: string; data: ArrayBuffer }>()

const url = ref('')
const error = ref('')

function revoke() {
  if (url.value) {
    URL.revokeObjectURL(url.value)
    url.value = ''
  }
}

function mount() {
  revoke()
  error.value = ''
  try {
    const mime = imageMimeType(props.path)
    const blob = new Blob([props.data], { type: mime })
    url.value = URL.createObjectURL(blob)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'failed to open image'
  }
}

watch(() => [props.path, props.data], mount, { immediate: true })
onBeforeUnmount(revoke)

function onImgError() {
  error.value = '无法显示该图片'
}
</script>

<template>
  <div class="h-full overflow-auto bg-neutral-100 flex items-center justify-center p-4">
    <div v-if="error" class="text-red-600 text-sm">{{ error }}</div>
    <img
      v-else-if="url"
      :src="url"
      :alt="path"
      class="max-w-full max-h-full object-contain shadow-sm bg-white"
      @error="onImgError"
    />
  </div>
</template>
