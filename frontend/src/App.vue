<script setup lang="ts">
import { watch, defineAsyncComponent } from 'vue'
import { ref, onMounted, onUnmounted } from 'vue'
import { ModalsContainer } from 'vue-final-modal';
import { useAppStore } from '@/stores/app'
import { useMsgStore } from '@/stores/msg'
import { useWebSocket } from '@/hooks/index'
import { showWelcomeModal } from '@/logics/index';
import { request, toast, } from '@/support/index';
import { getWebSocketUrl } from '@/support/consts';
import { setupWailsEvents } from '@/support/wails'
import { isFromApp, getAppTags } from '@/support/consts';
import Default from '@/layouts/Default.vue'
import NavBar from '@/views/NavBar.vue'
import ChatBox from '@/views/ChatBox.vue'
import DropZone from '@/widgets/DropZone.vue'
import Fireworks from '@/widgets/Fireworks.vue'
const Content = defineAsyncComponent(() => {
  return import('@/views/Content.vue')
})
const Browser = defineAsyncComponent(() => {
  return import('@/views/Browser.vue')
})
const Setting = defineAsyncComponent(() => {
  return import('@/views/Setting.vue')
})
const SetBot = defineAsyncComponent(() => {
  return import('@/views/SetBot.vue')
})
const SetMcp = defineAsyncComponent(() => {
  return import('@/views/SetMcp.vue')
})
const SetMem = defineAsyncComponent(() => {
  return import('@/views/SetMem.vue')
})
const SetTodo = defineAsyncComponent(() => {
  return import('@/views/SetTodo.vue')
})
const SetTool = defineAsyncComponent(() => {
  return import('@/views/SetTool.vue')
})

const app = useAppStore()
const msg = useMsgStore()
const socket = useWebSocket()

// 全局文件上传状态
const globalFiles = ref<File[]>([])
const globalUploads = ref<string[]>([])
// menubar
watch(() => app.getMenuBar, (val) => {
  var clz = 'hide-menu'
  var list = document.body.classList
  val ? list.remove(clz) : list.add(clz)
})
watch(() => app.getRefresh, (val) => {
  val === true && loadGlobal()
})
onMounted(async () => {
  if (!window.location.hash) {
    await listenEvent()
    await loadGlobal()
    await showDialog()
  } else {
    await listenEvent()
    onHashChange(new HashChangeEvent('hashchange', {
      newURL: window.location.hash, oldURL: '',
    }))
  }
  socket.doConnect(getWebSocketUrl())
  socket.useHandle('system', onSystemMsg)
  socket.useHandle('message', onMessage)
  // 所有初始化事件处理完毕后移除 loading
  setTimeout(() => {
    if (isFromApp() && getAppTags()) {
      document.body.classList.add(getAppTags())
    }
    const removeLoading = (window as any).__removeLoading
    removeLoading && removeLoading.constructor && removeLoading()
  }, 100)
  if (app.getUseModel) {
    setTimeout(() => {
      loadRelease()
    }, 3000)
  }
})
onUnmounted(() => {
  socket.disconnect()
})

const loadGlobal = async () => {
  try {
    const resp = await request.get('/global')
    const info = (resp || {}) as GlobalResp
    app.setLogin(info.login || {})
    app.setAuthGate(info.authGate)
    app.setInDocker(info.inDocker)
    if (info.active) {
      app.setActive(info.active)
    }
    if (info.setup) {
      app.setSetup(info.setup)
    }
    if (info.useModel) {
      app.useModel(info.useModel)
    }
    if (info.bots?.length > 0) {
      app.setBotList(info.bots)
    }
    if (info.epigraph) {
      app.setEpigraphText(info.epigraph)
      const today = (new Date()).toLocaleDateString()
      if (localStorage.getItem('last-shown') !== today) {
        app.setShowEpigraph(true)
        localStorage.setItem('last-shown', today)
      }
    }
  } catch (err) {
    console.log('failed to load global:', err)
  } finally {
    app.setRefresh(false)
    app.setLoaded(true)
    const wailsConfig = {
      dialog: {
        confirm: '确认',
        cancel: '取消',
      },
      upload: {
        title: '上传文件',
        message: '上传文件到Swiflow',
        handle: (files: string[]) => {
          console.log('upload', files)
        },
      },
    }
    setupWailsEvents(wailsConfig)
  }
}

const loadRelease = async () => {
  try {
    const url = '/toolenv?act=release'
    const resp = await request.get<any>(url)
    if (resp && resp.errmsg) {
      throw resp.errmsg
    }
    if (resp && resp.url) {
      app.setRelease(resp)
    }
  } catch (err) {
    console.log(err)
  }
}

const showDialog = async () => {
  const welcome = localStorage.getItem('welcome')
  if (welcome === 'login-success') {
    showWelcomeModal(app.authGateway, {
      currentStep: 2,
      login: app.getLogin,
      selectedMode: 'trial',
    })
    setTimeout(() => {
      localStorage.removeItem('welcome')
    }, 1000)
  } else if (welcome === 'python-install') {
    showWelcomeModal(app.authGateway, {
      currentStep: 3,
      pyInstalling: true,
    })
    setTimeout(() => {
      localStorage.removeItem('welcome')
    }, 1000)
  } else if (!app.getUseModel) {
    showWelcomeModal(app.authGateway, {})
  }
}

const listenEvent = async () => {
  window.addEventListener('resize', onResize as EventListener);
  window.addEventListener('storage', (_: StorageEvent) => null);
  window.addEventListener('welcome', onWelcome as EventListener);
  window.addEventListener('dispatch', onDispatch as EventListener);
  window.addEventListener("hashchange", onHashChange as EventListener);

}

const onResize = (_: UIEvent) => {
  const innerWidth = window.innerWidth
  if (innerWidth < 960 && app.getContent) {
    app.setContent(false)
  }
}

const onSystemMsg = (socketMsg: SocketMsg) => {
  switch (socketMsg.action) {
    case 'errors': {
      return toast.error(socketMsg.detail || 'System Error!');
    }
    case 'welcome': {
      // return task.setActive('')
    }
  }
}

const onMessage = (socketMsg: SocketMsg) => {
  // Process all chat messages through the msg store
  msg.processMessage(socketMsg)
}

const onHashChange = (e: HashChangeEvent) => {
  // url = /#/auth?token=xxxx&sign=xxxx&time=123
  const hash = e.newURL.replace(/^#\/?/, '')
  if (!hash) {
    return
  }
  const url = new URL(`http://localhost/${hash}`)
  const path = url.pathname.substring(1)
  const query = {} as Record<string, any>
  url.searchParams.forEach((val, key) => {
    query[key] = val
  })

  var matched = false
  switch (path) {
    case "auth": {
      handleAuth(query)
      matched = true
      break
    }
    case 'chat': {
      handleChat(query)
      matched = true
      break
    }
    default: {
      console.log('undefined hash path', hash, url)
      // 保持原有的 setAction 行为作为默认处理
      app.setAction(hash)
    }
  }
  if (matched) {
    location.hash = ''
  }
}

const onDispatch = (e: CustomEvent) => {
  const url = new URL(e.detail as string)
  const path = url.hostname + url.pathname
  const query = {} as Record<string, any>
  url.searchParams.forEach((val, key) => {
    query[key] = val
  })
  switch (path) {
    case "auth": {
      return handleAuth(query)
    }
    case 'chat': {
      return handleChat(query)
    }
    default: {
      console.log('undefined', e.detail, url)
    }
  }
}

const handleAuth = async (data: Record<string, any>) => {
  try {
    const url = '/sign-in?act=success'
    const resp = await request.post<any>(url, data)
    if (resp && resp.errmsg) {
      toast.error(resp.errmsg)
      return
    }
    // first login, continue welcome
    if (!app.getUseModel) {
      localStorage.setItem(
        'welcome', 'login-success'
      )
    }
    toast.success('Login Success!')
    return loadGlobal()
  } catch (err) {
    console.error('auth failed', err)
  }
}
const handleChat = (_: Record<string, any>) => {

}

// Component refs for method calls
const navBarRef = ref<typeof NavBar>()
const chatBoxRef = ref<typeof ChatBox>()
const onWelcome = (e: CustomEvent) => {
  const { botKey, prompt } = e.detail
  if (botKey && navBarRef.value) {
    const targetBot = app.getBotList.find(bot => bot.uuid === botKey)
    if (targetBot) {
      navBarRef.value.setActiveBot(targetBot).then(() => {
        console.info('Switched to bot:', targetBot)
      }).catch((err: any) => {
        console.error('Failed to switch bot:', err)
      })
    }
  }

  // Set the prompt to chatbox input
  if (prompt && prompt.trim() && chatBoxRef.value) {
    chatBoxRef.value.setMsgContent(prompt.trim())
  }
}

// 处理全局文件上传
const handleFilesDropped = (files: File[]) => {
  globalFiles.value = files
}

const handleFilesUploaded = (uploads: string[]) => {
  globalUploads.value = uploads
  app.setUploads(uploads)
}

// Handle agent import and navigate to Bot Set page
const handleAgentImported = async (imported: string[]) => {
  if (!imported.length) {
    return
  }
  const url = '/bot?act=get-bots'
  const resp = await request.get(url)
  const bots = resp as BotEntity[]
  if (bots && bots.length) {
    app.setBotList(bots)
    app.setContent(true)
    app.setChatBar(false)
    app.setAction('set-bot')
  }
}
</script>

<template>
  <Default>
    <template #nav>
      <NavBar v-if="app.getLoaded" ref="navBarRef"/>
    </template>
    <template #left>
      <ChatBox v-if="app.getLoaded" ref="chatBoxRef"/>
    </template>
    <template #main>
      <Content v-if="app.getAction == 'default'" />
      <Browser v-if="app.getAction == 'browser'" />
      <Setting v-if="app.getAction == 'setting'" />
      <SetBot v-if="app.getAction == 'set-bot'" />
      <SetMcp v-if="app.getAction == 'set-mcp'" />
      <SetMem v-if="app.getAction == 'set-mem'" />
      <SetTodo v-if="app.getAction == 'set-todo'" />
      <SetTool v-if="app.getAction == 'set-tool'" />
    </template>
  </Default>
  <ModalsContainer />
  <Fireworks v-if="app.display.showEpigraph" 
    @close="app.setShowEpigraph(false)" 
  />
  <DropZone :bot="app.getActive"
    @files-dropped="handleFilesDropped" 
    @files-uploaded="handleFilesUploaded" 
    @agent-imported="handleAgentImported" 
  />
</template>
