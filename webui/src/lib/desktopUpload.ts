import { isDesktop } from './desktop'

type NativeDragHook = {
  setNativeDragging: (active: boolean) => void
}

export interface WorkspaceUploadedPayload {
  path: string
  uploaded: { name: string; path: string; size: number }[]
  error?: string
}

type WailsEvents = {
  On: (name: string, cb: (ev: { data: WorkspaceUploadedPayload }) => void) => () => void
}

interface WailsDragRuntime {
  handleDragEnter?: () => void
  handleDragLeave?: () => void
  __swiflowDragPatched?: boolean
}

/** Hook Wails native drag enter/leave so Vue can show the drop overlay on macOS/Linux. */
export function patchDesktopNativeDrag(hook: NativeDragHook) {
  if (!isDesktop()) return

  const tryPatch = () => {
    const wails = (window as Window & { _wails?: WailsDragRuntime })._wails
    if (!wails?.handleDragEnter || !wails?.handleDragLeave) {
      requestAnimationFrame(tryPatch)
      return
    }
    if (wails.__swiflowDragPatched) return

    const origEnter = wails.handleDragEnter.bind(wails)
    const origLeave = wails.handleDragLeave.bind(wails)
    wails.handleDragEnter = () => {
      hook.setNativeDragging(true)
      origEnter()
    }
    wails.handleDragLeave = () => {
      hook.setNativeDragging(false)
      origLeave()
    }
    wails.__swiflowDragPatched = true
  }

  tryPatch()
}

/** Subscribe to desktop native file-drop results from Go. No-op in browser. */
export async function onDesktopWorkspaceUploaded(
  cb: (data: WorkspaceUploadedPayload) => void,
): Promise<() => void> {
  if (!isDesktop()) return () => {}

  try {
    const loader = new Function('return import("/wails/runtime.js")') as () => Promise<{ Events?: WailsEvents }>
    const mod = await loader()
    if (!mod.Events?.On) return () => {}
    return mod.Events.On('workspace-uploaded', (ev) => cb(ev.data))
  } catch {
    return () => {}
  }
}
