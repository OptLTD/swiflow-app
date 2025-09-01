<script setup lang="ts">
import { toast } from 'vue3-toastify'
import { doUploadFiles } from '@/logics/chat'
import { ref, onMounted } from 'vue'
import { useDropZone } from '@vueuse/core'

const props = defineProps<{
  bot?: BotEntity
}>()

const emit = defineEmits<{
  filesUploaded: [uploads: string[]]
  filesDropped: [files: File[]]
}>()

const appContainer = ref<HTMLElement>()

// 使用useDropZone处理拖拽，使用#app作为容器
const { isOverDropZone } = useDropZone(appContainer, (files: File[] | null): void => {
  if (files && files.length > 0) {
    handleFilesDropped(files as File[])
  }
})

const handleFilesDropped = async (files: File[]) => {
  emit('filesDropped', files)
  
  try {
    // 使用传入的bot对象
    if (!props.bot?.uuid) {
      toast.error('请先选择一个bot')
      return
    }
    
    const resp = await doUploadFiles(props.bot.uuid, files)
    
    if (resp.errmsg) {
      toast.error(`文件上传失败: ${resp.errmsg}`)
      return
    }
    
    const uploads: string[] = []
    for (const key in resp) {
      uploads.push(`[${key}](${resp[key]})`)
    }
    
    emit('filesUploaded', uploads)
    toast.success(`成功上传 ${files.length} 个文件到 ${props.bot.name || 'bot'} 目录`)
  } catch (error) {
    console.error('文件上传错误:', error)
    toast.error('文件上传失败')
  }
}

// 设置app容器引用
onMounted(() => {
  appContainer.value = document.querySelector('#app') as HTMLElement
})
</script>

<template>
  <div 
    v-if="isOverDropZone" 
    id="file-container"
    class="file-drop-zone"
  >
    <div class="drop-content">
      <div class="drop-icon">📁</div>
      <div class="drop-text">拖拽文件到此处上传</div>
    </div>
  </div>
</template>

<style scoped>
.file-drop-zone {
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