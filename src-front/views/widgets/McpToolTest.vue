<script setup lang="ts">
import { ref, unref, computed } from 'vue'
import { watch, PropType, onMounted } from 'vue'
import { toast } from 'vue3-toastify'
import { throttle } from 'lodash-es'
import { errors } from '@/support/index'
import { parser, request } from '@/support'
import { useWebSocket } from '@/hooks/index'
import { autoScrollToEnd } from '@/logics/chat'
import { showUseToolModal } from '@/logics/popup'
import McpToolList from './McpToolList.vue'
import ChatInput from '../chatbox/ChatInput.vue'
import ChatMsgList from '../chatbox/ChatMsgList.vue'

const errmsg = ref("")
const content = ref("")
const running = ref(false)
const socket = useWebSocket()
const currmcp = ref<BotEntity>()
const nextMsg = ref<ActionMsg|null>()
const messages = ref<ActionMsg[]>([])


// 新增props参数，接收MCP工具信息
const props = defineProps({
  config: {
    type: Object as PropType<McpServer>,
    default: () => null
  }
})
const server = ref(props.config)
const status = computed(() => {
  return props.config?.status
})
socket.useHandle('message', (data) => {
  onMessage(data)
})
onMounted(() => {
  handleLoadMsgs(props.config?.uuid)
})
watch(() => props.config, (data) => {
  if (unref(server).uuid != data.uuid) {
    server.value = data
    handleLoadMsgs(data.uuid)
  }
})

const getChatId = () => {
  return `#debug#${unref(server).uuid}`
}

const handleSend = async() => {
  if (!content.value) {
    return
  }
  const { status } = props.config || {}
  if (status.enable !== true) {
    return toast.info('请先启用此 MCP Server')
  }
  const msg: SocketMsg = {
    method: "message", 
    action: "user-input",
    chatid: getChatId(),
    detail: {
      content: content.value
    }
  }
  const conn = socket.getConnect()
  conn!.send(JSON.stringify(msg))
  content.value = ""
}

const handleStop = async() => {
  if (!running.value) {
    return
  }
  try {
    const uuid = unref(server).uuid
    const url = `/mcp?act=stop&uuid=${uuid}`
    const resp = await request.post(url) as any
    if (resp?.errmsg) {
      console.log("cancel error", resp)
      return toast.error(resp.errmsg)
    }
    nextMsg.value = null
    running.value = false
    delete(streamData.value[getChatId()])
  } catch (err) {
    console.error('use bot:', err)
  }
}

const handleClearMsg = async () => {
  const uuid = unref(server).uuid
  const url = `/mcp?act=clear&uuid=${uuid}`
  const resp = await request.get(url) 
  if (!resp || (resp as any).errmsg) {
    return
  }
  messages.value = [] as ActionMsg[]
}

const handleLoadMsgs = async (uuid: string) => {
  const url = `/mcp?act=msgs&uuid=${uuid}`
  const resp = await request.get(url) 
  if (!resp || (resp as any).errmsg) {
    messages.value = [] as ActionMsg[]
    return
  }

  const msgs = resp as ActionMsg[]
  messages.value = msgs as ActionMsg[]
  setTimeout(() => {
    autoScrollToEnd(true)
  }, 240)
}

const handleDisplay = (act: MsgAct) => {
  showUseToolModal(act)
}

const handleCheck = (val: string) => {
  if (content.value.includes(val)) {
    return
  }
  content.value += "\n" + val
  content.value = content.value.trim()
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
  setTimeout(() => autoScroll(false), 150)
}, 180)

const streamData = ref<any>({})
const onMessage = (msg: SocketMsg) => {
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
      const finished = [
        'completed', 'canceled', 'failed',
      ]
      if (finished.includes(msg.detail)) {
        nextMsg.value = null
        running.value = false
        delete(streamData.value[msg.chatid])
      }
      break;
    }
    case 'running': {
      errmsg.value = ''
      if (!streamData.value[msg.chatid]) {
        streamData.value[msg.chatid] = {}
      }
      const {idx, str} =  msg.detail as any
      streamData.value[msg.chatid][idx] = str
      setNextMsg(streamData.value[msg.chatid])
      break;
    }
    case 'respond': {
      errmsg.value = ''
      nextMsg.value = {
        actions: [] as MsgAct[]
      } as unknown as ActionMsg
      messages.value.push(msg.detail)
      delete(streamData.value[msg.chatid])
      break;
    }
    case 'errors': {
      return handleErrors(msg.detail)
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
</script>

<template>
  <div class="chat-container">
    <div class="list-container">
      <ChatMsgList
        :errmsg="errmsg"
        :currbot="currmcp"
        :loading="nextMsg"
        :messages="messages"
        @check="handleCheck"
        @display="handleDisplay"
      >
        <McpToolList :status="status" :server="server"/>
      </ChatMsgList>
    </div>
    <div class="input-container">
      <ChatInput :running="running"
        v-model:content="content" 
        @send="handleSend" @stop="handleStop"
      >
        <template #tools>
          <button class="btn-icon btn-remove" @click="handleClearMsg"/>
        </template>
      </ChatInput>
    </div>
  </div>
</template>

<style>
@import "../chatbox/index.css";
</style>
