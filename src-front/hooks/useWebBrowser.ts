import { Window } from '@tauri-apps/api/window';
import { TauriEvent } from '@tauri-apps/api/event';
import { WebviewWindow } from '@tauri-apps/api/webviewWindow';


export const useWebBrowser = () => {
  var mainWindow = Window.getCurrent()
  mainWindow.listen('tauri://close-requested', async (e: any) => {
    const windows = await Window.getAll();
    for (const window of windows) {
      if (window.label !== mainWindow.label) {
        try {
          await window.close();
        } catch (e) {
          console.error(`Failed to close window ${window.label}:`, e);
        }
      }
    }
    mainWindow.destroy()
  });
  window.addEventListener('message', function (e) { 
    console.log('message', e)
  })

  return {
    handle(msg: SocketMsg) {
      switch (msg.action) {
        case "goto": {
          // 打开新窗口
          const show = 1
          const options = { 
            url: msg.detail, height: 750 * show, width: 1000 * show,
            alwaysOnBottom: true, visible: false, hiddenTitle: true,
          }
          const webview = new WebviewWindow(msg.chatid, options);
          webview.once(TauriEvent.WEBVIEW_CREATED, () => {
            webview.reparent(mainWindow)
            !show && webview.hide()
          })
          webview.once(TauriEvent.WINDOW_CREATED, (e:any) => {
            console.log('命令执行结果:', e);

            const content = document.documentElement.outerHTML; // 获取整个页面HTML
            console.log(msg.action, msg.detail, content)
          })
          webview.once('tauri://error', function (e) {
            console.error('Failed to create window:', e);
          });
        }
      }
      // const client = this as unknown as WebSocket
      // setTimeout(() => {
      //   client.send(JSON.stringify(msg))
      // }, 1000)
      // console.log(msg, this, 'browser msg')
    },
  };
}