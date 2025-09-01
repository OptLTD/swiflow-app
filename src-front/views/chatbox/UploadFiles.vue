<template>
  <div class="upload-files" v-if="files?.length">
    <span v-for="(upload, i) in files" :key="i">
      <label>{{ getFileName(upload) }}</label>
      <a @click="$emit('remove', i)">x</a>
    </span>
  </div>
</template>
<script setup lang="ts">
defineProps<{
  files: string[]
}>()

const getFileName = (upload: string) => {
  // 从 [filename](url) 格式中提取文件名
  const match = upload.match(/\[(.*?)\]/)
  return match ? match[1] : upload
}
</script>
<style>
@import "./index.css";
</style> 