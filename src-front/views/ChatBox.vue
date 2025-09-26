<script setup lang="ts">
import { nanoid } from 'nanoid'
import { toast } from 'vue3-toastify'
import { throttle } from 'lodash-es'
import { ref, unref, watch } from 'vue'
import {  onMounted, onUnmounted } from 'vue'
import { request } from '@/support/index'
import { useAppStore } from '@/stores/app'
import { useMsgStore } from '@/stores/msg'
import { eventEmitter } from '@/stores/msg'
import { useWebSocket } from '@/hooks/index'
import { useTaskStore } from '@/stores/task'
import { useViewStore } from '@/stores/view'
import { getProviders } from '@/config/models'
import ChatInput from './chatbox/ChatInput.vue'
import ChatHeader from './chatbox/ChatHeader.vue'
import ChatMsgList from './chatbox/ChatMsgList.vue'
import ShowSelect from './chatbox/ShowSelect.vue'
import McpToolList from './chatbox/McpToolList.vue'
import UploadFiles from './chatbox/UploadFiles.vue'
import { setBotTools, setBotProvider } from '@/logics/chat'
import { showDisplayAct, autoScrollToEnd } from '@/logics/chat'

const app = useAppStore()
const msg = useMsgStore()
const task = useTaskStore()
const view = useViewStore()

const socket = useWebSocket()
const taskInfo = ref<TaskEntity>()
const inputMsg = ref({} as InputMsg)
const messages = ref<ActionMsg[]>([])
const refMcpTool = ref<typeof McpToolList>()
const workerMaps = ref<BotEntity>()
const emit = defineEmits(['new-chat'])

const handleSend = async() => {
  const { content } = inputMsg.value
  if (!content.trim()) {
    return
  }
  // 如果缺失必要命令，则抖动warning
  const tool = refMcpTool.value
  if (tool?.getMcpReady() == false) {
    tool.shakeElement()
    return;
  }
  const socketMsg: SocketMsg = {
    method: "message", 
    action: "user-input",
    taskid: msg.getChatId,
    detail: inputMsg.value
  }
  if (msg.getChatId == '') {
    const newChatId = nanoid(12)
    msg.setTaskId(newChatId)
    socketMsg.taskid = newChatId
    inputMsg.value.newTask = 'yes'
  }

  if (app.getUploads.length > 0) {
    inputMsg.value.uploads = app.getUploads
    app.setUploads([])
  }

  const conn = socket.getConnect()
  conn!.send(JSON.stringify(socketMsg))
  inputMsg.value = {} as InputMsg
}

const handleRemoveUpload = (index: number) => {
  const uploads = [...app.getUploads]
  uploads.splice(index, 1)
  app.setUploads(uploads)
}

const handleStop = async() => {
  if (!msg.isRunning) {
    return
  }
  try {
    const url = `/execute?act=stop&uuid=${msg.getChatId}`
    const resp = await request.post(url) as any
    if (resp?.errmsg) {
      console.log("cancel error", resp)
      return toast.error(resp.errmsg)
    }
    msg.setNextMsg(null)
    msg.setRunning(false)
    msg.clearStream(msg.getChatId)
  } catch (err) {
    console.error('use bot:', err)
  }
}

const onProviderChange = async (item: OptMeta) => {
  if (!workerMaps.value) {
    return
  }
  try {
    const uuid = unref(workerMaps)?.uuid as string
    const resp = await setBotProvider(uuid, item.value)
    if (!resp?.errmsg && workerMaps.value) {
      workerMaps.value.provider = item.value
    } else {
      throw resp.errmsg
    }
  } catch (err) {
    toast.error(err)
  }
}

const loadTaskInfo = async (uuid: string) => {
  try {
    const url = `/task?act=get-task&uuid=${uuid}`
    const resp = await request.post(url) as any
    if (resp?.errmsg) {
      console.log("cancel error", resp)
      return toast.error(resp.errmsg)
    }
    taskInfo.value = resp as TaskEntity
    msg.setRunning(resp.state == 'running')
    msg.setSubtasks(taskInfo.value.subtasks)
  } catch (e) {
    console.error('use bot:', e)
  }
}
const loadTaskMsgs = async (task: string) => {
  try {
    const url = `/msgs?task=${task}`
    const resp = await request.post<any>(url)
    if (resp?.errmsg) {
      console.log("cancel error", resp)
      return toast.error(resp.errmsg)
    }

    const msgs = resp as ActionMsg[]
    messages.value = msgs as ActionMsg[]
    const last = msgs[msgs.length - 1]
    last && startPlayAction(last, true)
  } catch (err) {
    console.error('use bot:', err)
  } finally {
    msg.setTaskId(task)
    setTimeout(() => {
      autoScrollToEnd(true)
    }, 240)
  }
}

const handleGoHome = () => {
  if (app.getContent && app.getAction != 'browser') {
    app.setAction('browser')
    return
  }
  if (app.getContent && app.getAction == 'browser') {
    app.setContent(false)
    return
  }
  if (!app.getContent) {
    app.setAction('browser')
    app.setContent(true)
    return
  }
}

const handleDisplay = (act: MsgAct) => {
  const show = showDisplayAct(act)
  if (!act.result && show === false) {
    return
  }
  view.setAction(act)
  app.setContent(true) 
  app.setAction('default')
}

const handleCheck = (val: string) => {
  let text = unref(inputMsg).content || ''
  if (text && text.includes(val)) {
    return
  }
  inputMsg.value.content = `${text}\n${val.trim()}`.trim()
}
const handleTools = (tools: string[]) => {
  workerMaps.value!.tools = tools
  const uuid = unref(workerMaps)?.uuid
  setBotTools(uuid as string, tools.join())
}
const handleSwitch = (tid: string) => {
  if (!tid) {
    msg.setTaskId('')
    msg.setErrMsg('')
    msg.setRunning(false)
    app.setContent(false)
    view.setAction(null)
    messages.value = []
    return;
  }
  if (tid != msg.getChatId) {
    msg.setErrMsg('')
    msg.setNextMsg(null)
    msg.setRunning(false)
    loadTaskMsgs(tid)
    loadTaskInfo(tid)
  }
}

const autoScroll = throttle(autoScrollToEnd, 500)

const startPlayAction = (msg: ActionMsg, force: boolean = false) => {
  if (!msg || !msg.actions?.length) {
    force && app.getContent && app.setContent(false)
    return null
  }
  const find = msg.actions.find(x => {
    return showDisplayAct(x)
  })
  if (!find && force && app.getContent) {
    view.setAction(null)
    app.setContent(false)
    return;
  }
  find && handleDisplay(find)
}

// Handle file click from UploadFiles component
const handleViewUpload = (filePath: string) => {
  if (filePath) {
    // Decode the file path to prevent double encoding in Browser.vue
    const decodedPath = decodeURIComponent(filePath)
    const detail = { path: decodedPath }
    app.setContent(true)
    app.setAction('browser')
    view.setChange(detail)
  }
}

watch(() => task.getActive, (uuid) => {
  handleSwitch(uuid)
})
onMounted(() => {
  // Listen to UI-specific events from msg store
  eventEmitter.on('user-input', (socketMsg: SocketMsg) => {
    if (msg.getChatId != socketMsg.taskid) {
      return
    }
    if (!task.getActive && msg.getChatId) {
      task.setActive(socketMsg.taskid)
    }
    // Add user input message to local messages
    messages.value.push({ 
      actions: [{
        type: 'user-input', 
        ...socketMsg.detail
      }]
    } as ActionMsg)
    setTimeout(() => autoScroll(true), 150)
  })
  
  eventEmitter.on('respond', (socketMsg: SocketMsg) => {
    // Add response message to local messages
    messages.value.push(socketMsg.detail)
    startPlayAction(socketMsg.detail)
  })
  
  eventEmitter.on('next-msg', (data: any) => {
    if (data.nextMsg) {
      startPlayAction(data.nextMsg)
      setTimeout(() => autoScroll(false), 150)
    }
  })

  inputMsg.value.placeholder = `
    输入消息内容，按下回车键发送
    拖拽文件到此处即可上传文件
  `.replace(/\n\s+/g, '\n').trim()

  // Load task data when component mounts
  if (task.getActive) {
    loadTaskMsgs(task.getActive)
    loadTaskInfo(task.getActive)
  }
})

onUnmounted(() => {
  // Clean up event listeners
  eventEmitter.off('next-msg', () => {})
  eventEmitter.off('respond', () => {})
  eventEmitter.off('user-input', () => {})
})

// Method to set message content from external components
const setMsgContent = (content: string) => {
  if (content && content.trim()) {
    inputMsg.value.content = content.trim()
  }
}

// Expose methods for parent component access
defineExpose({
  setMsgContent
})
</script>

<template>
  <ChatHeader @new-chat="() => task.setActive('')" />
  <div class="chat-container">
    <div class="list-container">
      <ChatMsgList
        :errmsg="msg.getErrMsg"
        :loading="msg.getNextMsg"
        :messages="messages"
        @check="handleCheck"
        @display="handleDisplay"
      >
      <div class="img-box">
        <img src="/assets/here.png" />
      </div>
     </ChatMsgList>
    </div>
    <div class="input-container">
      <ChatInput :running="msg.isRunning"
        v-model:content="inputMsg.content"
        :placeholder=" inputMsg.placeholder"
        @send="handleSend" @stop="handleStop"
      >
        <template #header>
          <UploadFiles 
            :files="app.getUploads" 
            @remove="handleRemoveUpload" 
            @detail="handleViewUpload"
          />
        </template>
        <template #tools>
          <button @click="handleGoHome"
            class="btn-icon btn-home"
            v-tippy="$t('tips.browserTips')"
          />
          <McpToolList v-if="app.getLoaded" 
            ref="refMcpTool" @change="handleTools"
            :tools="workerMaps?.tools" :enable="!task.getActive">
            <button class="btn-icon btn-tools"/>
          </McpToolList>
          <ShowSelect v-if="app.multi" :items="getProviders()"
            :active="workerMaps?.provider" @select="onProviderChange">
            <button class="btn-icon btn-switch"/>
          </ShowSelect>
        </template>
      </ChatInput>
    </div>
  </div>
</template>

<style>
@import "./chatbox/index.css";
.img-box {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
}
.img-box img {
  max-width: 75%;
  max-height: 75%;
  object-fit: contain;
  display: block;
}

[data-theme="dark"] .img-box img {
  filter: invert(0.4);
}
</style>
