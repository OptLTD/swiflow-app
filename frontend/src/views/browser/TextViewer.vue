<template>
  <div class="text-viewer">
    <div v-if="loading" class="loading-content">
      <p>{{ $t('common.loading') }}</p>
    </div>
    <div v-else-if="content" class="content-container">
      <!-- 代码文件使用 CodeMirror -->
      <CodeMirror
        v-if="isCodeFile"
        v-model="content"
        :disabled="true"
        class="code-editor"
      />
      <!-- 普通文本文件使用 pre 标签 -->
      <pre v-else class="file-content">{{ content }}</pre>
    </div>
    <div v-else class="empty-content">
      <p>{{ $t('common.emptyFileContent') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { toast } from 'vue3-toastify'
import { ref, onMounted, computed, defineAsyncComponent } from 'vue'
const CodeMirror = defineAsyncComponent(() => import('@/widgets/CodeMirror.vue'))

const { t } = useI18n()
const props = defineProps<{
  fileUrl: string
  fileName: string
}>()

const content = ref<string>('')
const loading = ref<boolean>(false)

// 常见代码文件扩展名
const codeExtensions = [
  'js', 'jsx', 'ts', 'tsx', 'vue', 'html', 'css', 'scss', 'sass', 'less',
  'py', 'java', 'cpp', 'c', 'cs', 'php', 'rb', 'go', 'rs', 'swift', 'kt',
  'json', 'xml', 'yaml', 'yml', 'toml', 'ini', 'conf', 'config',
  'sh', 'bash', 'zsh', 'ps1', 'bat', 'cmd',
  'sql', 'md', 'markdown', 'tex', 'r', 'm', 'pl', 'lua', 'dart'
]

// 判断是否为代码文件
const isCodeFile = computed(() => {
  const extension = props.fileName.split('.').pop()?.toLowerCase()
  return extension ? codeExtensions.includes(extension) : false
})

const loadFileContent = async () => {
  loading.value = true
  try {
    const response = await fetch(props.fileUrl)
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    
    const text = await response.text()
    content.value = text
  } catch (error) {
    console.error('加载文件内容失败:', error)
    toast.error(t('common.loadFileContentFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadFileContent()
})
</script>

<style scoped>
@import url('@/styles/viewer.css');
</style>