<script setup lang="ts">
defineProps<{
  files: string[]
}>()

const emit = defineEmits<{
  remove: [index: number]
  detail: [filePath: string]
}>()

const getFileName = (upload: string) => {
  // Extract filename from [filename](url) format
  const match = upload.match(/\[(.*?)\]/)
  return match ? match[1] : upload
}

const getFilePath = (upload: string) => {
  // Extract file path from [filename](url) format
  const match = upload.match(/\[.*?\]\((.*)\)/)
  return match ? match[1] : upload
}

// Handle file label click to emit file path
const handleFileClick = (upload: string) => {
  const filePath = getFilePath(upload)
  if (filePath) {
    emit('detail', filePath)
  }
}
</script>
<template>
  <div class="upload-files" v-if="files?.length">
    <span class="file-count">{{ files.length }}个文件:</span>
    <div class="files-list">
      <span v-for="(upload, i) in files" :key="i" class="file-item">
        <label @click="handleFileClick(upload)">{{ getFileName(upload) }}</label>
        <a @click="$emit('remove', i)">x</a>
      </span>
    </div>
  </div>
</template>

<style scoped>
.upload-files {
  display: flex;
  margin-right: 2rem;
  width: -webkit-fill-available;
}

.upload-files .file-count {
  margin-right: 8px;
  flex-shrink: 0;
}

.upload-files .files-list {
  display: flex;
  overflow: hidden;
  flex: 1;
}

.upload-files .file-item {
  margin-left: 10px;
  white-space: nowrap;
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.upload-files .file-item:first-child {
  margin-left: 0;
}

.upload-files .file-item label {
  cursor: pointer;
  max-width: 5rem;
  overflow: hidden;
  white-space: nowrap;
  display: inline-block;
  text-overflow: ellipsis;
}
.upload-files .file-item:hover label{
  max-width: 8rem;
}

.upload-files .file-item>a {
  display: none;
}

.upload-files .file-item:hover>a {
  display: inline-flex;
  cursor: pointer;
  margin: 0 -5px;
  padding: 0 5px;
  align-items: center;
}
</style>