<template>
  <div class="pdf-viewer">
    <div class="pdf-content">
      <div v-if="loading" class="loading">
        <div class="spinner"></div>
        <p>正在加载PDF文件...</p>
      </div>
      
      <div v-else-if="error" class="error">
        <div class="error-icon">❌</div>
        <h4>加载失败</h4>
        <p>{{ error }}</p>
        <button class="btn-retry" @click="loadPdf">
          重试
        </button>
      </div>
      
      <VuePDF 
        v-else
        :pdf="pdf" 
        text-layer 
        annotation-layer 
        class="pdf-component"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { toast } from 'vue3-toastify'
import { VuePDF, usePDF } from '@tato30/vue-pdf'
import '@tato30/vue-pdf/style.css'

const props = defineProps<{
  fileUrl: string
  fileName: string
}>()

// 响应式数据
const loading = ref(true)
const error = ref('')

// 使用vue-pdf的usePDF
const { pdf } = usePDF(props.fileUrl)

// 监听PDF加载状态
const loadPdf = async () => {
  try {
    loading.value = true
    error.value = ''
    
    // 等待PDF加载完成
    await pdf.value
    loading.value = false
  } catch (err: any) {
    loading.value = false
    error.value = err.message || '加载PDF文件失败'
    toast.error(err.message || '加载PDF文件失败')
  }
}



// 生命周期
onMounted(() => {
  loadPdf()
})
</script>

<style scoped>
@import url('@/styles/viewer.css');

.pdf-viewer {
  display: flex;
  flex-direction: column;
  background: #f5f5f5;
}

.loading, .error {
  height: 100%;
  color: #666;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #f3f3f3;
  border-top: 4px solid #007bff;
  margin-bottom: 16px;
}

.error-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.btn-retry {
  padding: 12px 24px;
  font-size: 14px;
}
</style> 