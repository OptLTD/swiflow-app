<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { toast } from 'vue3-toastify'
import { request } from '@/support/index'
import { BASE_ADDR } from '@/support/consts'
import { useAppStore } from '@/stores/app'
import { useTaskStore } from '@/stores/task'
import { useViewStore } from '@/stores/view'
import { ref, onMounted, watch, nextTick } from 'vue'
import DocViewer from './browser/DocViewer.vue'
import XlsViewer from './browser/XlsViewer.vue'
import PdfViewer from './browser/PdfViewer.vue'
import TextViewer from './browser/TextViewer.vue'

const { t } = useI18n()
const app = useAppStore()
const task = useTaskStore()
const view = useViewStore()

// 文件浏览器状态
const currDir = ref('.')
const loading = ref(false)
const preview = ref(false)
const selected = ref<string>('')
const fileList = ref<any[]>([])
const forceUpdateKey = ref(0) // 强制更新组件的key

// 智能返回/关闭
const close = () => {
  // 文件状态：关闭文件回到目录
   if (preview.value) {
    closeFilePreview()
  } else if (currDir.value === '.') {
    app.toggleContent() // 根目录：关闭浏览器
  } else {
    goBack() // 目录状态：返回上一级
  }
}

// 加载文件列表或文件内容
const loadFileData = async (path: string = '.') => {
  loading.value = true
  try {
    const uuid = app.getActive.uuid || ''
    const encodePath =encodeURIComponent(path)
    const url = `/browser?uuid=${uuid}&path=${encodePath}`
    const resp = await request.get(url) as any
    if (resp && resp.errmsg) {
      toast.error(resp.errmsg)
      return 
    }
    if (resp && Array.isArray(resp)) {
      // 后端返回的是数组格式
      const files = resp.map((item: any) => ({
        mode: item.mode || '-rw-r--r--',
        size: item.size || '0',
        time: item.time || '',
        name: item.name || '',
        isDir: (item.mode || '').startsWith('d'),
        isFile: (item.mode || '').startsWith('-'),
      }))
      currDir.value = path
      preview.value = false
      fileList.value = files
    }
  } catch (error) {
    console.error('加载数据失败:', error)
    toast.error(t('common.loadFileListFailed'))
  } finally {
    loading.value = false
  }
}

// 导航到指定路径
const navigateTo = (path: string) => {
  loadFileData(path)
}

// 刷新当前目录
const doRefresh = () => {
  loadFileData(currDir.value)
}

// 返回上一级目录
const goBack = () => {
  if (currDir.value === '.') return
  
  const pathParts = currDir.value.split('/')
  pathParts.pop()
  const parentPath = pathParts.length === 0 ? '.' : pathParts.join('/')
  navigateTo(parentPath)
}



// 单击文件或目录
const onItemClick = async (item: any) => {
  if (item.isDir) {
    navigateTo(`${currDir.value}/${item.name}`)
  } else {
    preview.value = true
    selected.value = item.name
  }
}

// 关闭文件预览
const closeFilePreview = () => {
  preview.value = false
  selected.value = ''
}

// 格式化文件大小
const formatFileSize = (size: number) => {
  if (size === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(size) / Math.log(k))
  return parseFloat((size / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 获取文件类型
const getFileType = (fileName: string) => {
  const name = fileName.toLowerCase()
  
  // 文档类型
  if (name.endsWith('.pdf')) return 'pdf'
  if (name.endsWith('.doc') || name.endsWith('.docx')) return 'doc'
  if (name.endsWith('.xls') || name.endsWith('.xlsx')) return 'xls'
  
  // 文本类型
  if (name.endsWith('.html') || name.endsWith('.css')) return 'web'
  if (name.endsWith('.txt') || name.endsWith('.md') || name.endsWith('.log')) return 'text'
  if (name.endsWith('.js') || name.endsWith('.ts') || name.endsWith('.py') || name.endsWith('.go')) return 'code'
  if (name.endsWith('.json') || name.endsWith('.yaml') || name.endsWith('.yml') || name.endsWith('.xml')) return 'config'
  
  return 'unknown'
}

// 获取文件图标
const getFileIcon = (item: any) => {
  if (item.isDir) {
    return '📁'
  }
  
  const fileType = getFileType(item.name)
  switch (fileType) {
    case 'doc': return '📄'
    case 'xls': return '📊'
    case 'pdf': return '📕'
    case 'text': return '📝'
    case 'code': return '📜'
    case 'config': return '⚙️'
    case 'web': return '🌐'
    default: return '📄'
  }
}

// 获取文件 URL - Use full HTTP URL for Tauri compatibility
const getFileUrl = (path: string, name: string) => {
  const encodePath = encodeURIComponent(`${path}/${name}`)
  return `${BASE_ADDR}/browser?uuid=${app.getActive.uuid}&path=${encodePath}`
}

// 获取查看器组件
const getViewerComponent = (fileName: string) => {
  const fileType = getFileType(fileName)
  switch (fileType) {
    case 'doc': return DocViewer
    case 'xls': return XlsViewer
    case 'pdf': return PdfViewer
    case 'text':
    case 'code':
    case 'config':
    case 'web':
    default: return TextViewer
  }
}

// 初始化
onMounted(() => {
  const change = view.getChange
  if (!change || !change.path) {
    loadFileData()
    return
  }
  if (change.type === 'directory') {
    navigateTo(change.path)
    return
  }
 
  preview.value = true
  selected.value = change.path
})

// 监听任务变化
watch(() => task.getActive, () => {
  if (task.getActive) {
    doRefresh()
  }
})

// 监听view.getChange的变化
watch(() => view.getChange, (change) => {
  if (!change || !change.path) {
    return
  }
  if (change.type === 'directory') {
    navigateTo(change.path)
    return
  }
 
  preview.value = true
  selected.value = change.path
  // 强制更新组件
  forceUpdateComponent()
})

// 强制更新组件的方法
const forceUpdateComponent = () => {
  forceUpdateKey.value++
  nextTick(() => {
    // 确保DOM更新完成
    console.log('Browser component force updated')
  })
}
</script>

<template>
  <div id="main-header">
    <button class="btn-back" @click="close">
      <svg class="icon" viewBox="0 0 24 24">
        <path d="M15.41 7.41L14 6l-6 6 6 6 1.41-1.41L10.83 12z"/>
      </svg>
    </button>
    <h3 class="main-title">{{ preview ? selected : $t('common.browser') }}</h3>
    <button v-if="!preview" class="btn-refresh" @click="doRefresh">
      <svg class="icon" viewBox="0 0 24 24">
        <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57
          8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 
          0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"
        />
      </svg>
    </button>
  </div>
  
  <!-- 文件浏览器内容 -->
  <div id="main-content" class="file-browser" :key="forceUpdateKey">
    <!-- 加载状态 -->
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>{{ $t('common.loading') }}</p>
    </div>

    <!-- 文件列表 -->
    <div v-else-if="!preview" class="file-list">
      <div class="file-list-header">
        <div class="header-back" @click="goBack">
          {{ currDir === '.' ? '.' : '..' }}
        </div>
        <div class="header-name">名称</div>
        <div class="header-size">大小</div>
        <div class="header-time">修改时间</div>
      </div>
      
      <div class="file-list-body">
        <div 
          @click="onItemClick(item)"
          v-for="item in fileList" 
          :key="item.name" class="file-item"
          :class="{ 'is-directory': item.isDir }"
        >
          <div class="file-icon">{{ getFileIcon(item) }}</div>
          <div class="file-name">{{ item.name }}</div>
          <div class="file-size">
            {{ item.isDir ? '-' : formatFileSize(parseInt(item.size) || 0) }}
          </div>
          <div class="file-time">{{ item.time }}</div>
        </div>
        
        <div v-if="fileList.length === 0" class="empty-state">
          <p>{{ $t('common.emptyDirectory') }}</p>
        </div>
      </div>
    </div>

    <!-- 文件预览 -->
    <div v-else class="file-preview">
      <component :fileName="selected"
        :is="getViewerComponent(selected)" 
        :fileUrl="getFileUrl(currDir, selected)"
      />
    </div>
  </div>
</template>

<style scoped>
#main-content {
  min-width: 0;
  padding: 0px 0px;
  overflow-y: scroll;
  margin-bottom: 15px;
  height: calc(100vh - var(--nav-height));
}

.file-browser {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: #666;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #f3f3f3;
  border-top: 4px solid #007bff;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 16px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.file-list {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.file-list-header {
  display: grid;
  grid-template-columns: 30px 1fr 80px 120px;
  padding: 8px 16px;
  background: #f8f9fa;
  border-bottom: 1px solid #e0e0e0;
  font-weight: 500;
  color: #666;
  font-size: 13px;
  align-items: center;
}

.header-back {
  cursor: pointer;
  color: #007bff;
  font-weight: bold;
  text-align: center;
  padding: 0px 6px;
  width: fit-content;
  border-radius: 3px;
  transition: background-color 0.2s;
}

.header-back:hover {
  background-color: #e9ecef;
}

.file-list-body {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.file-item {
  display: grid;
  grid-template-columns: 30px 1fr 80px 120px;
  padding: 6px 16px;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  transition: background-color 0.2s;
  align-items: center;
  min-height: 32px;
}

.file-item:hover {
  background-color: #f8f9fa;
}

.file-item.is-directory {
  font-weight: 500;
}

.file-icon {
  font-size: 16px;
  text-align: center;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.file-name {
  font-size: 13px;
  color: #333;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding-right: 8px;
}

.file-size {
  font-size: 11px;
  color: #666;
  text-align: right;
  padding-right: 8px;
}

.file-time {
  font-size: 11px;
  color: #666;
  text-align: right;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 120px;
  color: #999;
  font-size: 13px;
}

.file-preview {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.preview-header {
  display: flex;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #e0e0e0;
  background: #f8f9fa;
}

.btn-back-preview {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border: none;
  background: #007bff;
  color: white;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.btn-back-preview:hover {
  background: #0056b3;
}

.preview-title {
  margin: 0 0 0 16px;
  color: #333;
  font-size: 16px;
}

.preview-content {
  flex: 1;
  overflow: auto;
  padding: 16px;
}

.file-content {
  background: #f8f9fa;
  border: 1px solid #e0e0e0;
  border-radius: 4px;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-wrap: break-word;
  max-height: none;
  overflow: auto;
}

.empty-content {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: #999;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .file-list-header {
    grid-template-columns: 30px 1fr 80px;
  }
  
  .file-item {
    grid-template-columns: 30px 30px 1fr 80px;
  }
  
  .file-time {
    display: none;
  }
  
  .file-list-header .header-time {
    display: none;
  }
}
</style>
