

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { toast } from 'vue3-toastify'

const props = defineProps<{
  fileUrl: string
  fileName: string
}>()

const error = ref('')
const loading = ref(true)
const imageSrc = ref(props.fileUrl)

const onLoad = () => {
  console.log('图片加载成功')
  loading.value = false
}

const onError = () => {
  loading.value = false
  error.value = '无法加载图片文件'
  toast.error('无法加载图片文件')
}

const reload = () => {
  loading.value = true
  error.value = ''
  // 加随机参数以避免缓存影响
  imageSrc.value = `${props.fileUrl}&_=${Date.now()}`
}

onMounted(() => {
  // 初次渲染由 img 的 load/error 管理状态
  loading.value = false
})
</script>

<template>
  <div class="image-viewer">
    <div v-if="loading" class="loading">
      <div class="spinner"></div>
      <p>正在加载图片...</p>
    </div>

    <div v-else-if="error" class="error">
      <div class="error-icon">❌</div>
      <h4>加载失败</h4>
      <p>{{ error }}</p>
      <button class="btn-retry" @click="reload">
        重试
      </button>
    </div>

    <div class="image-container">
      <img class="image-content" 
        :src="imageSrc" :alt="fileName" 
        @load="onLoad" @error="onError" 
      />
    </div>
  </div>
</template>

<style scoped>
@import url('@/styles/viewer.css');

.image-viewer {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #f5f5f5;
}

.image-container {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  overflow: auto;
}

.image-content {
  max-width: 100%;
  max-height: calc(100vh - var(--nav-height) - 80px);
  object-fit: contain;
  background: white;
  border-radius: 4px;
  box-shadow: 0 2px 6px rgba(0,0,0,0.08);
}
</style>