<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useUploadStore } from '../stores/upload'
import LocalSvgIcon from './LocalSvgIcon.vue'

const upload = useUploadStore()
const { t } = useI18n()
</script>

<template>
  <div
    class="workspace-drop-zone h-full flex flex-col overflow-hidden relative"
    data-file-drop-target
    :data-upload-path="upload.targetPath"
    :class="{ 'is-html-dragging': upload.isDragging, 'is-uploading': upload.uploading }"
    @dragenter="upload.onDragEnter"
    @dragover="upload.onDragOver"
    @dragleave="upload.onDragLeave"
    @drop="upload.onDrop"
  >
    <slot />

    <div
      class="drop-overlay"
      :class="{ 'drop-overlay--visible': upload.showDropOverlay }"
      aria-hidden="true"
    >
      <div class="text-center px-6">
        <LocalSvgIcon name="folder-open" :size="36" class="text-blue-500 mx-auto mb-2" />
        <div class="text-sm font-medium text-blue-700">
          {{ upload.uploading ? t('dropzone.uploading') : t('dropzone.release') }}
        </div>
        <div class="text-xs text-blue-500 mt-1 font-mono">
          {{ upload.targetPath === '.' ? t('common.rootDir') : upload.targetPath }}
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.drop-overlay {
  position: absolute;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgb(239 246 255 / 0.85);
  border: 2px dashed rgb(96 165 250);
  pointer-events: none;
  opacity: 0;
  visibility: hidden;
  transition: opacity 0.12s ease, visibility 0.12s ease;
}

.drop-overlay--visible,
.workspace-drop-zone.file-drop-target-active .drop-overlay,
.workspace-drop-zone.is-html-dragging .drop-overlay,
.workspace-drop-zone.is-uploading .drop-overlay {
  opacity: 1;
  visibility: visible;
}
</style>
