import { Window, Browser, System } from "@wailsio/runtime";
import { Dialogs, Events } from "@wailsio/runtime";

type WailsDialog = {
  confirm: string
  cancel: string
}
type WailsUpload = {
  title: string
  message: string
  handle: (files: string[]) => void
}
type WailsConfig = {
  dialog: WailsDialog
  upload: WailsUpload
}

export const setupWailsEvents = async (config: WailsConfig) => {
  const env = await System.Environment()
  console.log('env', env, env.PlatformInfo)
  if (!env || !env.OSInfo) {
    return
  }

  setupDragDrop()
  setupPageHeader()
  setupWinControl()
  setupDialogs(config.dialog)
  setupUpload(config.upload)
}

const setupDialogs = (config: WailsDialog) => {
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

const setupPageHeader = () => {
  const header = document.querySelector('#top-header') as HTMLElement | null
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
  var app = document.querySelector('#app') as HTMLElement
  app?.style.setProperty('--default-contextmenu', 'show')
}

const setupWinControl = () => {
  const header = document.querySelector('#top-header') as HTMLElement
  if (!header || !System.IsWindows()) {
    return
  }
  document.body.classList.add('in-windows')
  if (!header.querySelector('.win-controls')) {
    const controls = document.createElement('div')
    controls.className = 'win-controls'
    controls.style.cssText = [
      'position:absolute',
      'right:8px',
      'top:0',
      'height:var(--nav-height)',
      'display:flex',
      'align-items:center',
      'gap:8px',
      'padding:0 8px',
      'z-index:10',
      '--wails-draggable:no-drag'
    ].join(';')

    const mkBtn = (clz: string, title: string, label: string) => {
      const btn = document.createElement('button')
      btn.className = `win-btn ${clz}`
      btn.title = title
      btn.textContent = label
      btn.style.cssText = [
        'width:28px',
        'height:22px',
        'display:flex',
        'align-items:center',
        'justify-content:center',
        'border:none',
        'border-radius:4px',
        'background:var(--bg-menu)',
        'color:var(--text-color)',
        'cursor:pointer',
        'outline:none',
        'font-size:12px',
        '--wails-draggable:no-drag'
      ].join(';')
      btn.onmouseenter = () => btn.style.background = 'var(--bg-dark)'
      btn.onmouseleave = () => btn.style.background = 'var(--bg-menu)'
      return btn
    }

    // 最小化
    const btnMin = mkBtn('win-min', '最小化', '—')
    btnMin.addEventListener('click', async (e) => {
      e.stopPropagation()
      try {
        await Window.Minimise()
      } catch (err) {
        console.error('Minimise error:', err)
      }
    })

    // 最大化 / 还原
    const btnMax = mkBtn('win-max', '最大化', '▢')
    btnMax.addEventListener('click', async (e) => {
      e.stopPropagation()
      try {
        if (await Window.IsMaximised()) {
          await Window.UnMaximise()
        } else {
          await Window.Maximise()
        }
      } catch (err) {
        console.error('Maximise error:', err)
      }
    })

    // 关闭
    const btnClose = mkBtn('win-close', '关闭', '✕')
    btnClose.addEventListener('click', async (e) => {
      e.stopPropagation()
      try {
        await Window.Close()
      } catch (err) {
        console.error('Close error:', err)
      }
    })

    controls.append(btnMin, btnMax, btnClose)
    header.appendChild(controls)
  }
}