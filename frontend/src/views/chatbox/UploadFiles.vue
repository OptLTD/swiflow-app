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

const handleUploadFile = () => {
  const input = document.querySelector('#global-file-input') as HTMLInputElement | null
  input?.click()
}
</script>
<template>
  <div class="empty-files" v-if="!files.length">
    <icon icon="icon-attach" size="mini" 
      text="点击或拖拽文件到此处即可上传文件"
      @click="() => handleUploadFile()"
    />
  </div>
  <div class="upload-files" v-else>
    <span class="file-count">文件:</span>
    <div class="files-list">
      <span v-for="(upload, i) in files" :key="i" class="file-item">
        <label @click="handleFileClick(upload)">{{ getFileName(upload) }}</label>
        <a @click="$emit('remove', i)">x</a>
      </span>
    </div>
  </div>
</template>

<style scoped>
.empty-files,
.upload-files {
  display: flex;
  padding: 5px 12px;
  width: -webkit-fill-available;
}
.empty-files{
  font-size: 13px;
  padding: 5px 5px;
  color: var(--color-secondary);
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