<template>
  <div class="xls-viewer">
    <!-- 悬浮的工作表选择器 -->
    <div class="sheet-selector">
      <FormKit
        type="select"
        name="sheetSelector"
        :options="sheetOptions"
        v-model="currentSheet"
        @change="switchSheet"
        :placeholder="'选择工作表'"
        :disabled="!sheetNames.length"
      />
    </div>
    
    <div class="xls-content">
      <div v-if="loading" class="loading">
        <div class="spinner"></div>
        <p>正在加载 Excel 文件...</p>
      </div>
      
      <div v-else-if="error" class="error">
        <div class="error-icon">❌</div>
        <h4>加载失败</h4>
        <p>{{ error }}</p>
        <button class="btn-retry" @click="loadFile">
          重试
        </button>
      </div>
      
      <div v-else-if="!tableData.length" class="empty">
        <div class="empty-icon">📊</div>
        <h4>无数据</h4>
        <p>此工作表没有数据</p>
      </div>
      
      <div v-else class="table-container">
        <table class="xls-table">
          <thead>
            <tr>
              <th v-for="(header, index) in tableHeaders" :key="index">
                {{ header }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, rowIndex) in tableData" :key="rowIndex">
              <td v-for="(cell, cellIndex) in row" :key="cellIndex">
                {{ cell }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { read, utils } from 'xlsx'
import { toast } from 'vue3-toastify'
import { ref, onMounted, computed } from 'vue'

interface Props {
  fileUrl: string
  fileName: string
}

const props = defineProps<Props>()

// 响应式数据
const loading = ref(true)
const error = ref('')
const workbook = ref<any>(null)
const currentSheet = ref('')
const sheetNames = ref<string[]>([])
const tableData = ref<any[][]>([])
const tableHeaders = ref<string[]>([])

// 计算属性 - 为 FormKit select 生成选项
const sheetOptions = computed(() => {
  return sheetNames.value.map(name => ({
    label: name,
    value: name
  }))
})

// 加载 Excel 文件
const loadFile = async () => {
  try {
    loading.value = true
    error.value = ''
    
    // 获取文件数据
    const response = await fetch(props.fileUrl)
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    
    const arrayBuffer = await response.arrayBuffer()
    
    // 解析 Excel 文件
    workbook.value = read(arrayBuffer)
    sheetNames.value = workbook.value.SheetNames
    
    if (sheetNames.value.length === 0) {
      throw new Error('Excel 文件中没有工作表')
    }
    
    // 设置默认工作表
    currentSheet.value = sheetNames.value[0]
    await switchSheet()
    
  } catch (err) {
    console.error('加载 Excel 文件失败:', err)
    error.value = err instanceof Error ? err.message : '未知错误'
    toast.error('加载 Excel 文件失败')
  } finally {
    loading.value = false
  }
}

// 切换工作表
const switchSheet = async () => {
  if (!workbook.value || !currentSheet.value) return
  
  try {
    const worksheet = workbook.value.Sheets[currentSheet.value]
    
    // 将工作表转换为数组格式
    const data = utils.sheet_to_json(worksheet, { header: 1 }) as any[][]
    
    if (data.length === 0) {
      tableData.value = []
      tableHeaders.value = []
      return
    }
    
    // 第一行作为表头
    tableHeaders.value = data[0].map((cell: any) => cell?.toString() || '')
    
    // 其余行作为数据
    tableData.value = data.slice(1).map((row: any[]) => 
      row.map((cell: any) => cell?.toString() || '')
    )
    
  } catch (err) {
    console.error('切换工作表失败:', err)
    toast.error('切换工作表失败')
  }
}

// 组件挂载时加载文件
onMounted(() => {
  loadFile()
})
</script>

<style scoped>
@import url('@/styles/viewer.css');

.xls-viewer {
  width: 100%;
  position: relative;
}

.xls-content {
  width: 100%;
  height: 100%;
}

.loading,
.error,
.empty {
  width: 100%;
  text-align: center;
  padding: 20px;
}

.error-icon,
.empty-icon {
  font-size: 40px;
  margin-bottom: 12px;
}

.loading p,
.error h4,
.error p,
.empty h4,
.empty p {
  margin: 0 0 6px 0;
  color: #666;
  font-size: 14px;
}

.error h4,
.empty h4 {
  color: #333;
  font-size: 16px;
}

.btn-retry {
  background: #ffc107;
  color: #212529;
  margin-top: 12px;
}

.btn-retry:hover {
  background: #e0a800;
}
</style> 