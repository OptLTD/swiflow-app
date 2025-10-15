<template>
  <div class="doc-viewer">
    <div class="doc-header">
      <h3>Word 文档查看器</h3>
      <p>文件名: {{ fileName }}</p>
    </div>
    <div class="doc-content">
      <div class="doc-placeholder">
        <div class="doc-icon">📄</div>
        <h4>Word 文档预览</h4>
        <p>此文件是 Word 文档格式 (.doc/.docx)</p>
        <p>暂不支持在线预览，请下载后使用 Word 打开</p>
        <button class="btn-download" @click="downloadFile">
          下载文件
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { toast } from 'vue3-toastify'

defineProps<{
  fileUrl: string
  fileName: string
}>()

const downloadFile = () => {
  // 这里可以实现文件下载逻辑
  toast.info('下载功能开发中...')
}
</script>

<style scoped>
.doc-viewer {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.doc-header {
  padding: 16px;
  border-bottom: 1px solid #e0e0e0;
  background: #f8f9fa;
}

.doc-header h3 {
  margin: 0 0 8px 0;
  color: #333;
}

.doc-header p {
  margin: 0;
  color: #666;
  font-size: 14px;
}

.doc-content {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
}

.doc-placeholder {
  text-align: center;
  max-width: 400px;
}

.doc-icon {
  font-size: 64px;
  margin-bottom: 16px;
}

.doc-placeholder h4 {
  margin: 0 0 16px 0;
  color: #333;
}

.doc-placeholder p {
  margin: 0 0 8px 0;
  color: #666;
  line-height: 1.5;
}

.btn-download {
  margin-top: 24px;
  padding: 12px 24px;
  background: #007bff;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.btn-download:hover {
  background: #0056b3;
}
</style> 