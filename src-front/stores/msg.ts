import { defineStore } from 'pinia'
import { useTaskStore } from './task'
import { useAppStore } from './app'
import { errors } from '@/support/index'

// Event emitter for UI-specific actions
class MsgEventEmitter {
  private listeners: Record<string, Function[]> = {}

  on(event: string, callback: Function) {
    if (!this.listeners[event]) {
      this.listeners[event] = []
    }
    this.listeners[event].push(callback)
  }

  off(event: string, callback: Function) {
    if (!this.listeners[event]) return
    const index = this.listeners[event].indexOf(callback)
    if (index > -1) {
      this.listeners[event].splice(index, 1)
    }
  }

  emit(event: string, ...args: any[]) {
    if (!this.listeners[event]) return
    this.listeners[event].forEach(callback => callback(...args))
  }
}

export const eventEmitter = new MsgEventEmitter()

export const useMsgStore = defineStore('msg', {
  state: () => ({
    chatid: '' as string,
    errmsg: '' as string,
    running: false,
    nextMsg: null as ActionMsg | null,
    streams: {} as Record<string, any>,
  }),

  actions: {
    // Handle user input message
    handleUserInput(msg: SocketMsg) {
      this.nextMsg = {
        actions: [] as MsgAct[]
      } as unknown as ActionMsg
      this.streams[msg.chatid] = {}
      
      // Emit UI event for adding message and auto-scroll
      eventEmitter.emit('user-input', msg)
    },

    // Handle control message
    handleControl(msg: SocketMsg) {
      const task = useTaskStore()
      
      if (msg.detail === "running") {
        this.running = true
      }
      
      if (msg.detail !== "running") {
        this.nextMsg = null
        this.running = false
        delete this.streams[msg.chatid]
        
        // Clear nextMsg after delay if still not running
        setTimeout(() => {
          if (!this.running) {
            this.nextMsg = null
          }
        }, 500)
      }
      
      // Update task state in history
      const current = task.getHistory.find(t => {
        return t.uuid === (msg.chatid || task.getActive)
      })
      if (current && current.state !== msg.detail) {
        current.state = msg.detail
      }
    },

    // Handle respond message
    handleRespond(msg: SocketMsg) {
      const task = useTaskStore()
      
      this.errmsg = ''
      this.nextMsg = {
        actions: [] as MsgAct[]
      } as unknown as ActionMsg
      
      if (!task.getActive) {
        task.setActive(msg.chatid)
      }
      
      delete this.streams[msg.chatid]
      
      // Emit UI events for adding message
      eventEmitter.emit('respond', msg)
    },

    // Handle stream message
    handleStream(msg: SocketMsg) {
      const task = useTaskStore()
      
      this.errmsg = ''
      if (!task.getActive) {
        this.chatid = msg.chatid
        task.setActive(msg.chatid)
      }
      
      if (!this.streams[msg.chatid]) {
        this.streams[msg.chatid] = {}
      }
      
      const {idx, str} = msg.detail as any
      this.streams[msg.chatid][idx] = str
      
      // Emit UI event for stream processing
      eventEmitter.emit('stream', msg)
    },

    // Handle error messages
    handleErrors(detail: string) {
      if (!detail || !detail.split) {
        return
      }
      
      const error = detail.split(':').shift()
      
      switch (error) {
        case errors.EMPTY_LLM_RESPONSE:
        case errors.NO_RESULT_PRESENT: {
          break
        }
        case errors.EXCEEDED_MAXIMUM_TURNS:
        case errors.TASK_TERMINATED_BY_USER: {
          this.nextMsg = null
          this.running = false
          this.errmsg = detail
          break
        }
        default: {
          this.nextMsg = null
          this.running = false
          this.errmsg = detail
        }
      }
    },

    // Handle file change messages
    handleFileChange(detail: any) {
      const app = useAppStore()
      
      if (detail && detail.path) {
        app.setContent(true)
        app.setAction('browser')
      }
    },

    // Main message processor - replaces the original onMessage function
    processMessage(msg: SocketMsg) {
      switch (msg.action) {
        case "user-input":
          this.handleUserInput(msg)
          break
        case 'control':
          this.handleControl(msg)
          break
        case 'respond':
          this.handleRespond(msg)
          break
        case 'stream':
          this.handleStream(msg)
          break
        case 'errors':
          this.handleErrors(msg.detail)
          break
        case 'change':
          this.handleFileChange(msg.detail)
          break
      }
    },

    // Utility actions
    clearMessages() {
      this.nextMsg = null
      this.streams = {}
      this.errmsg = ''
    },

    setRunning(running: boolean) {
      this.running = running
    },

    setChatId(chatid: string) {
      this.chatid = chatid
    },

    setNextMsg(nextMsg: ActionMsg | null) {
      this.nextMsg = nextMsg
    },

    setErrorMsg(errmsg: string) {
      this.errmsg = errmsg
    },

    clearStream(chatid: string) {
      delete this.streams[chatid]
    },

    updateStream(chatid: string, idx: number, str: string) {
      if (!this.streams[chatid]) {
        this.streams[chatid] = {}
      }
      this.streams[chatid][idx] = str
    },
  },

  getters: {
    // Get current running state
    isRunning: (state) => state.running,
    
    // Get next message
    getNextMsg: (state) => state.nextMsg,
    
    // Get error message
    getErrMsg: (state) => state.errmsg,

    // Get current chat ID
    getChatId: (state) => state.chatid,

    // Get stream data
    getStream: (state) => state.streams,

    // Check if stream data exists for chatid
    hasStream: (state) => (chatid: string) => {
      return !!state.streams[chatid] && Object.keys(state.streams[chatid]).length > 0
    }
  }
})