import { defineStore } from 'pinia'
import { api } from '../api'
import { onDesktopWorkspaceUploaded, patchDesktopNativeDrag } from '../lib/desktopUpload'
import { isDesktop } from '../lib/desktop'
import { useAuthStore } from './auth'
import { useLayoutStore } from './layout'
import { useToastStore } from './toast'

function hasFiles(e: DragEvent) {
  const types = e.dataTransfer?.types
  return !!types && Array.from(types).includes('Files')
}

function collectFiles(dt: DataTransfer | null): File[] {
  if (!dt?.files?.length) return []
  const out: File[] = []
  for (const file of Array.from(dt.files)) {
    if (file.name && !file.name.startsWith('.')) out.push(file)
  }
  return out
}

export const useUploadStore = defineStore('upload', {
  state: () => ({
    dragDepth: 0,
    refreshSeq: 0,
    uploading: false,
    desktopBound: false,
    nativeDragging: false,
  }),
  getters: {
    isDragging: (s) => s.dragDepth > 0,
    showDropOverlay: (s) => s.dragDepth > 0 || s.nativeDragging || s.uploading,
    targetPath(): string {
      const layout = useLayoutStore()
      if (layout.activeTab.type === 'explore') {
        return layout.explorePath || layout.activeTab.path || '.'
      }
      return '.'
    },
  },
  actions: {
    setNativeDragging(active: boolean) {
      this.nativeDragging = active
    },
    bindDesktopDrop() {
      if (this.desktopBound || !isDesktop()) return
      this.desktopBound = true
      patchDesktopNativeDrag(this)
      void onDesktopWorkspaceUploaded((data) => {
        this.dragDepth = 0
        this.nativeDragging = false
        this.uploading = false
        const toast = useToastStore()
        if (data.error) {
          toast.error(data.error)
          return
        }
        const loc = data.path === '.' ? 'workspace 根目录' : data.path
        const names = (data.uploaded || []).map((f) => f.name).join('、')
        const preview = names.length > 48 ? `${names.slice(0, 48)}…` : names
        toast.success(`已上传 ${data.uploaded.length} 个文件到 ${loc}${preview ? `：${preview}` : ''}`)
        this.refreshSeq += 1
      })
    },
    onDragEnter(e: DragEvent) {
      if (!hasFiles(e)) return
      e.preventDefault()
      this.dragDepth += 1
    },
    onDragOver(e: DragEvent) {
      if (!hasFiles(e)) return
      e.preventDefault()
      if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
    },
    onDragLeave(e: DragEvent) {
      if (!hasFiles(e)) return
      e.preventDefault()
      this.dragDepth = Math.max(0, this.dragDepth - 1)
    },
    async onDrop(e: DragEvent) {
      if (!hasFiles(e)) return
      e.preventDefault()
      this.dragDepth = 0

      // Wails desktop uses native file paths; HTML5 File objects are empty on macOS/Linux.
      if (isDesktop()) {
        this.uploading = true
        return
      }

      const auth = useAuthStore()
      if (!auth.isAuthed) {
        useToastStore().error('请先登录后再上传文件')
        return
      }

      const files = collectFiles(e.dataTransfer)
      if (!files.length) return
      await this.uploadFiles(files)
    },
    async uploadFiles(files: File[]) {
      if (!files.length || this.uploading) return

      const path = this.targetPath
      this.uploading = true
      const toast = useToastStore()
      try {
        const r = await api.uploadWorkspace(path, files)
        const loc = path === '.' ? 'workspace 根目录' : path
        const names = r.uploaded.map((f) => f.name).join('、')
        const preview = names.length > 48 ? `${names.slice(0, 48)}…` : names
        toast.success(`已上传 ${r.uploaded.length} 个文件到 ${loc}${preview ? `：${preview}` : ''}`)
        this.refreshSeq += 1
      } catch (err: unknown) {
        toast.error(err instanceof Error ? err.message : '上传失败')
      } finally {
        this.uploading = false
      }
    },
  },
})
