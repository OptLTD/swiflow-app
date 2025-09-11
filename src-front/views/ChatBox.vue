<script setup lang="ts">
import { nanoid } from 'nanoid'
import { toast } from 'vue3-toastify'
import { throttle } from 'lodash-es'
import { ref, unref } from 'vue'
import { watch, onMounted } from 'vue'
import { errors } from '@/support/index'
import { parser, request } from '@/support'
import { useAppStore } from '@/stores/app'
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
const task = useTaskStore()
const view = useViewStore()

const chatid = ref("")
const errmsg = ref("")
const running = ref(false)
const socket = useWebSocket()
const currBot = ref<BotEntity>()
const taskInfo = ref<TaskEntity>()
const inputMsg = ref({} as InputMsg)
const nextMsg = ref<ActionMsg|null>()
const messages = ref<ActionMsg[]>([])
const refMcpTool = ref<typeof McpToolList>()
const emit = defineEmits(['new-chat'])

const handleSend = async() => {
  const { content } = inputMsg.value
  if (!content.trim()) {
    return
  }
  // 如果有工具选中，缺失必要命令，则抖动工具
  const tool = refMcpTool.value
  if (tool?.getLossCmd!()) {
    tool.shakeElement()
    return;
  }
  const msg: SocketMsg = {
    method: "message", 
    action: "user-input",
    chatid: unref(chatid),
    detail: inputMsg.value
  }
  if (chatid.value == '') {
    chatid.value = nanoid(12)
    msg.chatid = chatid.value
    inputMsg.value.newTask = 'yes'
  }

  if (app.getUploads.length > 0) {
    inputMsg.value.uploads = app.getUploads
    app.setUploads([])
  }

  const conn = socket.getConnect()
  conn!.send(JSON.stringify(msg))
  inputMsg.value = {} as InputMsg
}

const handleRemoveUpload = (index: number) => {
  const uploads = [...app.getUploads]
  uploads.splice(index, 1)
  app.setUploads(uploads)
}

const handleStop = async() => {
  if (!running.value) {
    return
  }
  try {
    const url = `/execute?act=stop&uuid=${chatid.value}`
    const resp = await request.post(url) as any
    if (resp?.errmsg) {
      console.log("cancel error", resp)
      return toast.error(resp.errmsg)
    }
    nextMsg.value = null
    running.value = false
    delete(streamData.value[chatid.value])
  } catch (err) {
    console.error('use bot:', err)
  }
}

const onProviderChange = async (item: OptMeta) => {
  if (!currBot.value) {
    return
  }
  try {
    const uuid = unref(currBot)?.uuid as string
    const resp = await setBotProvider(uuid, item.value)
    if (!resp?.errmsg && currBot.value) {
      currBot.value.provider = item.value
    } else {
      throw resp.errmsg
    }
  } catch (err) {
    toast.error(err)
  }
}

const queryBot = (uuid: string) => {
  uuid = uuid || app.getActive?.uuid || '';
  return app.getBotList.find(x => x.uuid == uuid)
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
    running.value = resp.state == 'running'
  } catch (e) {
    console.error('use bot:', e)
  }
}
const loadTaskMsgs = async (uuid: string) => {
  try {
    const url = `/task?act=history&uuid=${uuid}`
    const resp = await request.post(url) as any
    if (resp?.errmsg) {
      console.log("cancel error", resp)
      return toast.error(resp.errmsg)
    }

    const msgs = resp as ActionMsg[]
    messages.value = msgs.map(x => x)
    const last = msgs[msgs.length - 1]
    last && startPlayAction(last, true)
  } catch (err) {
    console.error('use bot:', err)
  } finally {
    chatid.value = uuid
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
  if (!act.result) {
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
  currBot.value!.tools = tools
  const uuid = unref(currBot)?.uuid
  setBotTools(uuid as string, tools.join())
}
const handleSwitch = (tid: string) => {
  if (!tid) {
    errmsg.value = ''
    chatid.value = ''
    messages.value = []
    running.value = false
    view.setAction(null)
    return;
  }
  if (tid != unref(chatid)) {
    errmsg.value = ''
    running.value = false
    nextMsg.value = null
    loadTaskMsgs(tid)
    loadTaskInfo(tid)
  }
}

const autoScroll = throttle(autoScrollToEnd, 500)
const setNextMsg = throttle((stream: any  = {}) => {
  if (!running.value) {
    return
  }
  var data = ''
  for (var i=1; i < 50000; i++) {
    if (stream.hasOwnProperty(i)) {
      data += stream[i]
    } else {
      break
    }
  }
  const next = parser.Parse(data)
  nextMsg.value = next as any as ActionMsg;
  startPlayAction(next as any as ActionMsg)
  setTimeout(() => autoScroll(false), 150)
}, 180)

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

const streamData = ref<any>({})
const onMessage =  (msg: SocketMsg) => {
  switch (msg.action) {
    case "user-input": {
      nextMsg.value = {
        actions:[] as MsgAct[]
      } as unknown as ActionMsg
      streamData.value[msg.chatid] = {}
      messages.value.push({ actions: [{
        type: 'user-input', ...msg.detail
      }]} as ActionMsg)
      setTimeout(() => autoScroll(true), 150)
      break
    }
    case 'control': {
      if (msg.detail == "running") {
        running.value = true
      }
      if (msg.detail != "running") {
        nextMsg.value = null
        running.value = false
        delete(streamData.value[msg.chatid])
        setTimeout(() => {
          if (!running.value) {
            nextMsg.value = null
          }
        }, 500)
      }
      
      // 更新任务状态到history中
      const current = task.getHistory.find(t => {
        return t.uuid === (msg.chatid || task.getActive)
      })
      if (current && current.state != msg.detail) {
        current.state = msg.detail
      }
      break;
    }
    case 'respond': {
      errmsg.value = ''
      nextMsg.value = {
        actions: [] as MsgAct[]
      } as unknown as ActionMsg
      if (!task.getActive) {
        task.setActive(msg.chatid)
      }
      startPlayAction(msg.detail)
      messages.value.push(msg.detail)
      delete(streamData.value[msg.chatid])
      setNextMsg(streamData.value[msg.chatid])
      break;
    }
    case 'stream': {
      errmsg.value = ''
      if (!task.getActive) {
        chatid.value = msg.chatid
        task.setActive(msg.chatid)
      }
      if (!streamData.value[msg.chatid]) {
        streamData.value[msg.chatid] = {}
      }
      const {idx, str} =  msg.detail as any
      streamData.value[msg.chatid][idx] = str
      setNextMsg(streamData.value[msg.chatid])
      break;
    }
    case 'errors': {
      return handleErrors(msg.detail)
    }
    case 'change': {
      handleFileChange(msg.detail)
      break;
    }
  }
}
const handleErrors = (detail: string) => {
  if (!detail || !detail.split) {
    return;
  }
  var error = detail.split(':').shift()
  switch (error) {
    case errors.EMPTY_LLM_RESPONSE:
    case errors.NO_RESULT_PRESENT: {
      break
    }
    case errors.EXCEEDED_MAXIMUM_TURNS:
    case errors.TASK_TERMINATED_BY_USER: {
      nextMsg.value = null
      running.value = false
      errmsg.value = detail
      break
    }
    default: {
      nextMsg.value = null
      running.value = false
      errmsg.value = detail
    }
  }
}

// 处理文件变动消息
const handleFileChange = (detail: any) => {
  if (detail && detail.path) {
    app.setContent(true)
    app.setAction('browser')
    view.setChange(detail)
  }
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
watch(() => app.getBotList, () => {
  currBot.value = queryBot('')
})
watch(() => app.getActive, () => {
  currBot.value = queryBot('')
})
onMounted(() => {
  socket.useHandle(
    'message', onMessage
  )
  currBot.value = queryBot('')
  // Load task data when component mounts
  if (task.getActive) {
    loadTaskMsgs(task.getActive)
    loadTaskInfo(task.getActive)
  }
})
</script>

<template>
  <ChatHeader @new-chat="() => task.setActive('')" />
  <div class="chat-container">
    <div class="list-container">
      <ChatMsgList
        :errmsg="errmsg"
        :currbot="currBot"
        :loading="nextMsg"
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
      <ChatInput :running="running"
        v-model:content="inputMsg.content"
        placeholder="输入消息内容，按下回车键发送"
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
            :tools="currBot?.tools" :enable="!task.getActive">
            <button class="btn-icon btn-tools"/>
          </McpToolList>
          <ShowSelect v-if="app.multi" :items="getProviders()"
            :active="currBot?.provider" @select="onProviderChange">
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
