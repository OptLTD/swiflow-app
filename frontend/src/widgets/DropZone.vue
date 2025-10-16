<script setup lang="ts">
import { toast } from 'vue3-toastify'
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDropZone } from '@vueuse/core'
import { doUploadFiles } from '@/logics/chat'
import { doImportFiles } from '@/logics/chat'

const { t } = useI18n()

const props = defineProps<{
  bot?: BotEntity
}>()

const emit = defineEmits<{
  filesDropped: [files: File[]]
  filesUploaded: [uploads: string[]]
  agentImported: [imported: string[]]
}>()

const appContainer = ref<HTMLElement>()

// Use useDropZone to handle drag and drop, using #app as container
const { isOverDropZone } = useDropZone(appContainer, (files: File[] | null): void => {
  if (files && files.length > 0) {
    handleFilesDropped(files as File[])
  }
})

const handleFilesDropped = async (files: File[]) => {
  emit('filesDropped', files)

  try {
    const { agentFiles, regularFiles } = separateFiles(files)
    // If there are .agent files, prioritize them and discard regular files
    if (agentFiles.length > 0) {
      await handleAgentFiles(agentFiles, regularFiles)
      return
    }

    // Handle regular files only if no .agent files are present
    if (regularFiles.length > 0) {
      await handleRegularFiles(regularFiles)
    }
  } catch (error) {
    console.error(t('dropzone.fileProcessError'), error)
    toast.error(t('dropzone.fileProcessFailed'))
  }
}

// Separate .agent files from regular files
const separateFiles = (files: File[]): { agentFiles: File[], regularFiles: File[] } => {
  const agentFiles: File[] = []
  const regularFiles: File[] = []

  files.forEach(file => {
    if (file.name.endsWith('.agent')) {
      agentFiles.push(file)
    } else {
      regularFiles.push(file)
    }
  })

  return { agentFiles, regularFiles }
}

// Handle agent files import
const handleAgentFiles = async (agentFiles: File[], regularFiles: File[]) => {
  if (regularFiles.length > 0) {
    toast.info(t('dropzone.agentFilesDetected', {
      count: agentFiles.length,
    }))
  }

  const resp = await doImportFiles(agentFiles)
  if (resp.errmsg) {
    toast.error(t('dropzone.agentImportFailed', {
      error: resp.errmsg,
    }))
  } else {
    toast.success(t('dropzone.agentImportSuccess', {
      count: resp.count,
      names: resp.imported.join(', ')
    }))
    emit('agentImported', resp.imported)
  }
}

// Handle regular files upload
const handleRegularFiles = async (regularFiles: File[]) => {
  // Use the passed bot object
  if (!props.bot?.uuid) {
    toast.error(t('dropzone.selectBotFirst'))
    return
  }

  const resp = await doUploadFiles(props.bot.uuid, regularFiles)

  if (resp.errmsg) {
    toast.error(t('dropzone.fileUploadFailed', {
      error: resp.errmsg,
    }))
    return
  }

  const uploads: string[] = []
  for (const key in resp) {
    uploads.push(`[${key}](${resp[key]})`)
  }

  emit('filesUploaded', uploads)
  toast.success(t('dropzone.fileUploadSuccess', {
    count: regularFiles.length,
    path: props.bot.name || 'bot'
  }))
}

// Set app container reference
onMounted(() => {
  appContainer.value = document.querySelector('#app') as HTMLElement
})

// Handle hidden input change (click-to-upload)
const onFileInputChange = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const files = Array.from(input.files || [])
  if (!files.length) return

  try {
    // Reuse existing flow
    const { agentFiles, regularFiles } = separateFiles(files)
    if (agentFiles.length > 0) {
      await handleAgentFiles(agentFiles, regularFiles)
    } else if (regularFiles.length > 0) {
      await handleRegularFiles(regularFiles)
    }
  } catch (error) {
    console.error(t('dropzone.fileProcessError'), error)
    toast.error(t('dropzone.fileProcessFailed'))
  } finally {
    // reset to allow re-select same files
    input.value = ''
  }
}
</script>

<template>
  <div id="file-drop-zone" v-show="isOverDropZone">
    <div class="drop-content">
      <div class="drop-icon">📁</div>
      <div class="drop-text">
        {{ t('dropzone.dropFilesHere') }}
      </div>
    </div>
  </div>
  <!-- hidden input for click-to-upload -->
  <input id="file-upload-input" type="file"
    multiple style="display:none" 
    @change="onFileInputChange" 
  />
</template>

<style scoped>
#file-drop-zone {
  display: flex;
  font-size: 1.75rem;
  align-items: center;
  justify-content: center;
  width: 100vw;
  height: 100vh;
  z-index: 2025;
  position: fixed;
  top: 0;
  left: 0;
  border: 2px dashed #4a90e2;
  background-color: rgba(74, 144, 226, 0.1);
  backdrop-filter: blur(2px);
}

.drop-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 2rem;
  background-color: rgba(255, 255, 255, 0.95);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
}

.drop-icon {
  font-size: 4rem;
  opacity: 0.8;
}

.drop-text {
  font-size: 1.5rem;
  color: #333;
  font-weight: 500;
}

[data-theme="dark"] .drop-content {
  background-color: rgba(30, 30, 30, 0.95);
  color: #fff;
}

[data-theme="dark"] .drop-text {
  color: #fff;
}
</style>