import { defineStore } from 'pinia'
import { throttle } from 'lodash-es'
import { useAppStore } from './app'
import { useTaskStore } from './task'
import { errors, parser } from '@/support'

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
    console.log(`Emitting event: ${event} with args:`, ...args)
    this.listeners[event].forEach(callback => callback(...args))
  }
}

export const eventEmitter = new MsgEventEmitter()

export const useMsgStore = defineStore('msg', {
  state: () => ({
    taskid: '' as string,
    errmsg: '' as string,
    running: false as boolean,
    subtasks: [] as string[],
    nextMsg: null as ActionMsg | null,
    streams: {} as Record<string, any>,
  }),

  actions: {
    // Handle user input message
    handleUserInput(msg: SocketMsg) {
      this.nextMsg = {
        actions: [] as MsgAct[]
      } as unknown as ActionMsg
      this.streams[msg.taskid] = {}
      
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
        delete this.streams[msg.taskid]
        
        // Clear nextMsg after delay if still not running
        setTimeout(() => {
          if (!this.running) {
            this.nextMsg = null
          }
        }, 500)
      }
      
      // Update task state in history
      const current = task.getHistory.find(t => {
        return t.uuid === (msg.taskid || task.getActive)
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
        task.setActive(msg.taskid)
      }
      
      delete this.streams[msg.taskid]
      
      // Emit UI events for adding message and trigger processNextMsg
      eventEmitter.emit('respond', msg)
      this.processNextMsg(this.streams[msg.taskid])
    },

    // Handle stream message
    handleStream(msg: SocketMsg) {
      this.errmsg = ''
      if (!this.streams[msg.taskid]) {
        this.streams[msg.taskid] = {}
      }
      
      const {idx, str} = msg.detail as any
      this.streams[msg.taskid][idx] = str
      
      // Process next message with priority handling
      this.processNextMsg()
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

    setTaskId(taskid: string) {
      this.taskid = taskid
    },

    setErrMsg(errmsg: string) {
      this.errmsg = errmsg
    },

    setNextMsg(nextMsg: ActionMsg | null) {
      this.nextMsg = nextMsg
    },

    clearStream(taskid: string) {
      delete this.streams[taskid]
    },

    updateStream(taskid: string, idx: number, str: string) {
      if (!this.streams[taskid]) {
        this.streams[taskid] = {}
      }
      this.streams[taskid][idx] = str
    },

    setSubtasks(subtasks: string[]) {
      this.subtasks = subtasks
    },
    
    clearSubtasks() {
      this.subtasks = []
    },

    // Process stream data and emit next-msg event with priority handling for subtasks
    processNextMsg: throttle(function(this: any, stream: any = {}) {
      if (!this.running) {
        return
      }
      
      // Determine which taskid to process based on subtasks priority
      let active = ''
      if (this.subtasks.length > 0) {
        // Find the first subtask that has stream data
        for (const subtaskId of this.subtasks) {
          if (this.streams[subtaskId] && Object.keys(this.streams[subtaskId]).length > 0) {
            active = subtaskId
            break
          }
        }
      }
      
      active = active || this.taskid
      const target = this.streams[active] || stream
      var data = '', worker = '', msgid = ''
      for (var i = 1; i < 50000; i++) {
        if (target.hasOwnProperty(i)) {
          data += target[i]
        } else {
          break
        }
      }
      
      if (target[0] && target[0].includes('data:')) {
        [, worker, msgid] = target[0].split(':')
        console.log('stream data', worker, msgid)
      }
      
      const next = parser.Parse(data, worker, msgid)
      this.setNextMsg(next as any as ActionMsg)
      
      // Emit next-msg event for UI components to handle
      eventEmitter.emit('next-msg', {
        taskid: active, nextMsg: next,
      })
    }, 180),
  },

  getters: {
    // Get current running state
    isRunning: (state) => state.running,
    
    // Get next message
    getNextMsg: (state) => state.nextMsg,

    // Get subtasks array
    getSubtasks: (state) => state.subtasks,
    
    // Get error message
    getErrMsg: (state) => state.errmsg,

    // Get current chat ID
    getTaskId: (state) => state.taskid,

    // Get stream data
    getStream: (state) => state.streams,

    // Check if stream data exists for taskid
    hasStream: (state) => (taskid: string) => {
      return !!state.streams[taskid] && Object.keys(state.streams[taskid]).length > 0
    }
  }
})