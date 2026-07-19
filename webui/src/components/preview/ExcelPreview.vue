<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import jspreadsheet from 'jspreadsheet-ce'
import LocalSvgIcon from '../LocalSvgIcon.vue'
import type { ExcelSheet } from '../../lib/filePreview'
import 'jsuites/dist/jsuites.css'
import 'jspreadsheet-ce/dist/jspreadsheet.css'

const LARGE_CELL_THRESHOLD = 50_000

const props = defineProps<{ sheets: ExcelSheet[]; path: string }>()
const emit = defineEmits<{ refresh: [] }>()
const { t } = useI18n()
const host = ref<HTMLDivElement | null>(null)

function sheetCellCount(data: string[][]): number {
  const rows = data.length
  const cols = Math.max(...data.map((r) => r.length), 0)
  return rows * cols
}

const totalCells = computed(() =>
  props.sheets.reduce((sum, s) => sum + sheetCellCount(s.data), 0),
)

const isLarge = computed(() => totalCells.value > LARGE_CELL_THRESHOLD)

/** Large workbooks open read-only by default; smaller ones start editable. */
const editableMode = ref(false)

function resetEditableDefault() {
  editableMode.value = !isLarge.value
}

function columnsFor(data: string[][]) {
  const cols = Math.max(...data.map((row) => row.length), 1)
  return Array.from({ length: cols }, () => ({ width: 120 }))
}

function minRows(data: string[][]) {
  return Math.max(data.length, 12)
}

function minCols(data: string[][]) {
  return Math.max(...data.map((row) => row.length), 6)
}

function buildWorksheets() {
  const editable = editableMode.value
  return props.sheets.map((sheet) => ({
    worksheetName: sheet.name,
    data: sheet.data,
    columns: columnsFor(sheet.data),
    editable,
    allowInsertRow: editable,
    allowInsertColumn: editable,
    allowDeleteRow: editable,
    allowDeleteColumn: editable,
    minDimensions: [minCols(sheet.data), minRows(sheet.data)] as [number, number],
  }))
}

function destroyGrid() {
  if (!host.value) return
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  jspreadsheet.destroy(host.value as any)
}

function mountGrid() {
  if (!host.value || !props.sheets.length) return
  destroyGrid()
  jspreadsheet(host.value, {
    tabs: true,
    worksheets: buildWorksheets(),
  })
}

function toggleEditable() {
  editableMode.value = !editableMode.value
  mountGrid()
}

watch(
  () => [props.path, props.sheets] as const,
  () => {
    resetEditableDefault()
    mountGrid()
  },
  { deep: true },
)
onMounted(() => {
  resetEditableDefault()
  mountGrid()
})
onBeforeUnmount(destroyGrid)
</script>

<template>
  <div class="h-full min-h-0 overflow-hidden bg-white excel-preview-root relative">
    <div ref="host" class="excel-host h-full min-h-0 overflow-hidden" />
    <div class="excel-actions absolute top-0 right-0 z-10 h-9 flex items-center border-l border-neutral-200 bg-neutral-50">
      <button
        type="button"
        class="h-9 w-9 flex items-center justify-center hover:bg-neutral-100"
        :class="editableMode ? 'text-blue-600' : 'text-neutral-500 hover:text-neutral-800'"
        :title="editableMode ? t('filePreview.excelReadonly') : t('filePreview.excelEditable')"
        @click="toggleEditable"
      >
        <LocalSvgIcon :name="editableMode ? 'edit' : 'lock'" :size="15" />
      </button>
      <button
        type="button"
        class="h-9 w-9 flex items-center justify-center text-neutral-500 hover:bg-neutral-100 hover:text-neutral-800"
        :title="t('common.refresh')"
        @click="emit('refresh')"
      >
        <LocalSvgIcon name="refresh" :size="15" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.excel-preview-root :deep(.jtabs) {
  display: flex;
  flex-direction: column;
  height: 100%;
  max-width: none;
  overflow: hidden;
}

/* Keep sheet tabs fixed; only the grid scrolls horizontally/vertically */
.excel-preview-root :deep(.jtabs-headers-container) {
  flex-shrink: 0;
  height: 36px;
  box-sizing: border-box;
  border-bottom: 1px solid #e5e5e5;
  background: #fafafa;
  overflow-x: auto;
  overflow-y: hidden;
  /* space for edit + refresh */
  padding-right: 72px;
}

.excel-preview-root :deep(.jtabs-headers > div:not(.jtabs-border)) {
  padding: 0 14px;
  height: 36px;
  display: flex;
  align-items: center;
  font-size: 13px;
  background: transparent;
  margin: 0;
}

.excel-preview-root :deep(.jtabs-headers > div.jtabs-selected) {
  background: #fff;
  border-bottom: 2px solid #525252;
}

.excel-preview-root :deep(.jtabs-controls) {
  display: none;
}

.excel-preview-root :deep(.jtabs-content) {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.excel-preview-root :deep(.jss_container),
.excel-preview-root :deep(.jss) {
  font-size: 13px;
}
</style>
