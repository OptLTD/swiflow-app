<script setup lang="ts">
import FilePreview from '../components/FilePreview.vue'

defineProps<{ path: string }>()
</script>

<template>
  <FilePreview :path="path" />
</template>
