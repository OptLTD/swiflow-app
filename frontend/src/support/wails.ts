import { Window, Browser } from "@wailsio/runtime";
import { Dialogs, Events } from "@wailsio/runtime";

const { location } = window || {}
export const isWails = () => {
  return location.protocol === 'wails:'
}

type WailsConfirm = {
  confirm: string
  cancel: string
}
type WailsUpload = {
  title: string
  message: string
  handle: (files: string[]) => void
}
type WailsConfig = {
  dialog: WailsConfirm
  upload: WailsUpload
}

export const setupWailsEvents = (config: WailsConfig) => {
  if (!isWails()) {
    return
  }

  setupAutoZoom()
  setupDragDrop()
  setupDialogs(config.dialog)
  setupUpload(config.upload)
}

const setupAutoZoom = () => {
  const header = document.querySelector('#top-header')
  if (!header || !Window.Get('Swiflow')) {
    return
  }
  header.addEventListener('dblclick', async () => {
    if (await Window.IsMaximised()) {
      await Window.UnMaximise()
    } else {
      await Window.Maximise()
    }
  })
}

const setupDialogs = (config: WailsConfirm) => {
  // @ts-ignore
  window.open = Browser.OpenURL
  // @ts-ignore
  window.confirm = async (msg: string) => {
    const confirm = config.confirm || '确认'
    const cancel = config.cancel || '取消'
    const resp = await Dialogs.Warning({
      Message: msg, Title: confirm, Buttons: [
        { Label: confirm, IsDefault: true },
        { Label: cancel, IsCancel: true },
      ] as Dialogs.Button[],
    })
    return resp === confirm
  }
}

const setupUpload = (config: WailsUpload) => {
  const id = '#file-upload-input'
  const ele = document.querySelector(id)
  const input = ele as HTMLInputElement
  input?.addEventListener('click', async () => {
    const upload = config.title || '上传文件'
    const result = await Dialogs.OpenFile({
      CanChooseFiles: true,
      AllowsMultipleSelection: true,
      Message: config.message || "上传文件", 
      Title: upload, ButtonText: upload,
    })
    Events.Emit('app:FileSelected', result)
  })
  Events.On('app:Uploaded', ({ data }: any) => {
    if (!data || !data.length) {
      return
    }
    const {result, errors} = data[0] || {}
    if (result && result.length > 0) {
      config.handle && config.handle(result)
    }
    if (errors && errors.length > 0) {
      Dialogs.Error({
        Title: '上传失败', 
        Message: errors.join('\n'), 
        Buttons: [
          { Label: '确认', IsDefault: true },
        ] as Dialogs.Button[],
      })
      return
    }
    console.log(result, errors, 'upload info')
  });
}

const setupDragDrop = () => {
  document.elementFromPoint = (_x: number, _y: number) => {
    return document.querySelector(`[data-wails-dropzone]`)
  }

  const id = '#file-drop-zone'
  const ele = document.querySelector(id)
  const dropzone = ele as HTMLInputElement
  const { Mac, Windows, Common } = Events.Types
  Events.On(Mac.WindowFileDraggingEntered, (_) => {
    dropzone.style.display = 'flex'
  })
  Events.On(Mac.WindowFileDraggingPerformed, (_) => {
    dropzone.style.display = 'none'
  })
  Events.On(Mac.WindowFileDraggingExited, (_) => {
    dropzone.style.display = 'none'
  })
  // windows
  Events.On(Windows.WindowDragEnter, (_) => {
    dropzone.style.display = 'flex'
  })
  Events.On(Windows.WindowDragOver, (_) => {
    dropzone.style.display = 'flex'
  })
  Events.On(Windows.WindowDragLeave, (_) => {
    dropzone.style.display = 'none'
  })
  Events.On(Windows.WindowDragDrop, (_) => {
    dropzone.style.display = 'none'
  })

  // common
  Events.On(Common.WindowFilesDropped, (_) => {
    // dropzone.style.display = 'flex'
  })
}