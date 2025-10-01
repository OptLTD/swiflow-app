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
import { useTaskStore } from '@/stores/task'
import { useViewStore } from '@/stores/view'
import ChatInput from './chatbox/ChatInput.vue'
import ChatHeader from './chatbox/ChatHeader.vue'
import ChatMsgList from './chatbox/ChatMsgList.vue'
import ShowSelect from './chatbox/ShowSelect.vue'
import McpToolList from './chatbox/McpToolList.vue'
import UploadFiles from './chatbox/UploadFiles.vue'
import { setBotTools } from '@/logics/chat'
import { autoScrollToEnd } from '@/logics/chat'
import { shouldAutoDisplay } from '@/logics/chat'
import { canShowDisplayAct } from '@/logics/chat'

const app = useAppStore()
const msg = useMsgStore()
const task = useTaskStore()
const view = useViewStore()

const taskInfo = ref<TaskEntity>()
const inputMsg = ref({} as InputMsg)
const messages = ref<ActionMsg[]>([])
const currWorker = ref<BotEntity>()
const refMcpTool = ref<typeof McpToolList>()
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

  if (task.getActive == '') {
    const newTaskId = nanoid(12)
    inputMsg.value.startNew = 'yes'
    inputMsg.value.taskUUID = newTaskId
    msg.setTaskId(newTaskId)
    task.setActive(newTaskId)
  } else {
    inputMsg.value.taskUUID = task.getActive
  }
  // select worker to debug
  const { uuid } = currWorker.value || {}
  if (uuid !== app.getActive?.uuid) {
    inputMsg.value.workerId = uuid
  }

  if (app.getUploads.length > 0) {
    inputMsg.value.uploads = app.getUploads
    app.setUploads([])
  }

  // http method: post
  request.post('/start', inputMsg.value).then((resp: any) => {
    if (resp?.errmsg) {
      return toast.error(resp.errmsg)
    }
    // Add user input message to local messages
    const action = { type: 'user-input', ...inputMsg.value }
    messages.value.push({ actions: [action] } as ActionMsg)
    setTimeout(() => autoScroll(true), 150)
    inputMsg.value = {} as InputMsg
  })
}

const handleStop = async() => {
  if (!msg.isRunning) {
    return
  }
  try {
    const url = `/execute?act=stop&uuid=${msg.getTaskId}`
    const resp = await request.post(url) as any
    if (resp?.errmsg) {
      console.log("cancel error", resp)
      return toast.error(resp.errmsg)
    }
    msg.setNextMsg(null)
    msg.setRunning(false)
    msg.clearStream(msg.getTaskId)
  } catch (err) {
    console.error('use bot:', err)
  }
}

const onWorkerChange = (item: OptMeta) => {
  const select = app.getBotList.find(x => {
    return x.uuid == item.value
  })
  currWorker.value = select || currWorker.value 
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
  const shouldDisplay = shouldAutoDisplay(act)
  const canShowComplete = canShowDisplayAct(act)
  if (shouldDisplay && !canShowComplete) {
    return
  }
  const skip = ['complete', 'start-subtask']
  if (!act.result && !skip.includes(act.type)) {
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
  if (tid != msg.getTaskId) {
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
    return canShowDisplayAct(x)
  })
  if (!find && force && app.getContent) {
    view.setAction(null)
    app.setContent(false)
    return;
  }
  find && handleDisplay(find)
}

const handleToolsChange = (tools: string[]) => {
  currWorker.value!.tools = tools
  const uuid = unref(currWorker)?.uuid
  setBotTools(uuid as string, tools.join())
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

const handleRemoveUpload = (index: number) => {
  const uploads = [...app.getUploads]
  uploads.splice(index, 1)
  app.setUploads(uploads)
}

watch(() => task.getActive, (uuid) => {
  handleSwitch(uuid)
})
onMounted(() => {
  currWorker.value = app.getActive
  eventEmitter.on('respond', (socketMsg: SocketMsg) => {
    if (socketMsg.taskid != task.getActive) {
      console.log('Respond message not match:', socketMsg)
      return
    }
    console.log('Respond message:', socketMsg)
    // Add response message to local messages
    messages.value.push(socketMsg.detail)
    startPlayAction(socketMsg.detail)
  })
  
  eventEmitter.on('next-msg', (data: any) => {
    if (msg.getSubtasks.includes(data.taskid)) {
      console.log('Received next-msg message:', data)
      startPlayAction(data.nextMsg)
      setTimeout(() => autoScroll(false), 150)
    }
  })

  if (task.getActive) {
    loadTaskMsgs(task.getActive)
    loadTaskInfo(task.getActive)
  }
})

onUnmounted(() => {
  // Clean up event listeners
  eventEmitter.off('respond', () => {})
  eventEmitter.off('next-msg', () => {})
})

// Method to set message content from external components
const setMsgContent = (content: string) => {
  if (content && content.trim()) {
    inputMsg.value.content = content.trim()
  }
}
const getWorkers = (): OptMeta[] => {
  if (!app.useSubAgent) {
    return []
  }
  const leader = app.getActive
  const result = [] as OptMeta[]
  result.push({
    label: leader.name,
    value: leader.uuid,
    disabled: false,
  } as OptMeta)
  const workers = app.getBotList.filter((worker) => {
    return worker.leader == leader.uuid
  })
  workers.forEach((worker) => {
    result.push({
      label: worker.name,
      value: worker.uuid,
      disabled: false,
    } as OptMeta)
  })
  return result;
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
          <ShowSelect 
            v-if="app.useSubAgent" :active="currWorker?.uuid"
            @select="onWorkerChange" :items="getWorkers()">
            <button class="btn-icon btn-switch"/>
          </ShowSelect>
          <McpToolList v-if="app.getLoaded"
            @change="handleToolsChange" ref="refMcpTool"
            :tools="currWorker?.tools" :enable="!task.getActive">
            <button class="btn-icon btn-tools"/>
          </McpToolList>
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
