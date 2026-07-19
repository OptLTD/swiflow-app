import { defineStore } from 'pinia'
import { t } from '../i18n'
import { api } from '../api'
import { onDesktopWorkspaceUploaded, patchDesktopNativeDrag } from '../lib/desktopUpload'
import { isDesktop } from '../lib/desktop'
import { useLayoutStore } from './layout'
import { useChatStore } from './chat'
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

/** Attach uploaded workspace files to the active chat pending bar when chat is in context. */
function maybeAttachToChat(uploaded: { name: string; path: string }[]) {
  if (!uploaded.length) return
  const layout = useLayoutStore()
  if (!layout.showChatSidebar && !layout.isChatTabActive) return

  const chat = useChatStore()
  let sessionKey = ''
  if (layout.isChatTabActive && layout.activeTab.path) {
    sessionKey = layout.activeTab.path
  } else if (layout.showChatSidebar) {
    sessionKey = chat.currentKey
    if (!sessionKey) {
      sessionKey = 'sess-' + Math.random().toString(36).slice(2, 10)
      chat.setSession(sessionKey, '')
    }
  }
  if (!sessionKey) return
  chat.addPending(sessionKey, uploaded)
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
        const loc = data.path === '.' ? t('upload.root') : data.path
        const names = (data.uploaded || []).map((f) => f.name).join('、')
        const preview = names.length > 48 ? `${names.slice(0, 48)}…` : names
        toast.success(t('upload.success', {
          count: data.uploaded.length,
          loc,
          preview: preview ? `：${preview}` : '',
        }))
        this.refreshSeq += 1
        maybeAttachToChat(data.uploaded || [])
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
        const loc = path === '.' ? t('upload.root') : path
        const names = r.uploaded.map((f) => f.name).join('、')
        const preview = names.length > 48 ? `${names.slice(0, 48)}…` : names
        toast.success(t('upload.success', {
          count: r.uploaded.length,
          loc,
          preview: preview ? `：${preview}` : '',
        }))
        this.refreshSeq += 1
        maybeAttachToChat(r.uploaded)
      } catch (err: unknown) {
        toast.error(err instanceof Error ? err.message : t('upload.failed'))
      } finally {
        this.uploading = false
      }
    },
  },
})
