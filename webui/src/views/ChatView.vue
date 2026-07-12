<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { api, chat, watchSession } from '../api'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { useAgentsStore } from '../stores/agents'
import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js/lib/core'
import jsonLang from 'highlight.js/lib/languages/json'
import python from 'highlight.js/lib/languages/python'
import bash from 'highlight.js/lib/languages/bash'
import 'highlight.js/styles/github.min.css'
import ToolCallBlock from '../components/ToolCallBlock.vue'
import ThinkingBlock from '../components/ThinkingBlock.vue'
import type { ChatEvent, Message, Session } from '../types'

hljs.registerLanguage('json', jsonLang)
hljs.registerLanguage('python', python)
hljs.registerLanguage('bash', bash)

const md = new MarkdownIt({ html: false, linkify: true, breaks: true })
const auth = useAuthStore()
const chatStore = useChatStore()
const agentsStore = useAgentsStore()

interface Msg {
  role: string
  content: string
  thinking?: string
  tool_name?: string
  id?: string
  arguments?: Record<string, unknown>
  isError?: boolean
  streaming?: boolean
}

const sessions = ref<Session[]>([])
const currentKey = ref('')
const sessionAgentKey = ref('')
const selectedAgentKey = ref('default')
const messages = ref<Msg[]>([])
const input = ref('')
const streaming = ref(false)
const error = ref('')
const scrollEl = ref<HTMLElement | null>(null)
let watchAbort: AbortController | null = null
let bgCur: Msg | null = null

function stopWatch() {
  watchAbort?.abort()
  watchAbort = null
}

async function startWatch(key: string) {
  stopWatch()
  if (!key) return
  const ac = new AbortController()
  watchAbort = ac
  while (!ac.signal.aborted && currentKey.value === key) {
    try {
      await watchSession(key, (ev) => {
        if (streaming.value) return
        handleChatEvent(ev, () => bgCur, (m) => { bgCur = m })
      }, ac.signal)
    } catch (e: unknown) {
      if (e instanceof Error && e.name === 'AbortError') return
    }
    if (ac.signal.aborted || currentKey.value !== key) return
    await new Promise((r) => setTimeout(r, 1000))
  }
}

function handleChatEvent(
  ev: ChatEvent,
  getCur: () => Msg | null,
  setCur: (m: Msg | null) => void,
) {
  if (ev.type === 'user') {
    messages.value.push({ role: 'user', content: ev.content || '' })
    streaming.value = true
    setCur(pushAssistant())
    scrollBottom()
    return
  }
  let cur = getCur()
  if (ev.type === 'delta') {
    if (!cur) { streaming.value = true; cur = pushAssistant(); setCur(cur) }
    cur.content += ev.content || ''
  } else if (ev.type === 'thinking') {
    if (!cur) { streaming.value = true; cur = pushAssistant(); setCur(cur) }
    cur.thinking = (cur.thinking || '') + (ev.content || '')
  } else if (ev.type === 'tool_call') {
    if (cur) {
      cur.streaming = false
      if (!cur.content && !cur.thinking) {
        messages.value.splice(messages.value.length - 1, 1)
      }
      setCur(null)
      cur = null
    }
    messages.value.push({
      role: 'tool',
      id: ev.id,
      tool_name: ev.name,
      arguments: ev.arguments,
      content: '',
      isError: false,
    })
  } else if (ev.type === 'tool_result') {
    const t = messages.value.find((m) => m.role === 'tool' && m.id === ev.id)
    if (t) {
      t.content = ev.result || ''
      t.isError = ev.is_error
    }
  } else if (ev.type === 'error') {
    error.value = ev.error || 'error'
  } else if (ev.type === 'done') {
    if (cur) {
      cur.streaming = false
      setCur(null)
    }
    streaming.value = false
    if (ev.title) {
      chatStore.currentTitle = ev.title
      const s = sessions.value.find((s) => s.key === currentKey.value)
      if (s) s.title = ev.title
      else loadSessions()
    }
  }
  scrollBottom()
}

function pushAssistant(): Msg {
  const a: Msg = { role: 'assistant', content: '', streaming: true }
  messages.value.push(a)
  return a
}

function mapStoredMessages(raw: Message[]): Msg[] {
  const out: Msg[] = []
  for (const m of raw) {
    if (m.role === 'assistant' && m.tool_calls_json) {
      let tcs: { id: string; name: string; arguments?: Record<string, unknown> }[] = []
      try {
        tcs = JSON.parse(m.tool_calls_json)
      } catch {}
      if (m.content || m.thinking) {
        out.push({ role: 'assistant', content: m.content, thinking: m.thinking })
      }
      for (const tc of tcs) {
        out.push({
          role: 'tool',
          id: tc.id,
          tool_name: tc.name,
          arguments: tc.arguments,
          content: '',
        })
      }
      continue
    }
    if (m.role === 'tool') {
      out.push({
        role: 'tool',
        id: m.tool_call_id,
        tool_name: m.tool_name,
        content: m.content,
        isError: false,
      })
      continue
    }
    out.push({ role: m.role, content: m.content, thinking: m.thinking })
  }
  return out
}

onMounted(() => {
  if (auth.isAuthed) {
    loadSessions()
    agentsStore.load().catch((e: Error) => { error.value = e.message })
  }
})
onUnmounted(() => stopWatch())
watch(() => auth.isAuthed, (v) => {
  if (v) {
    loadSessions()
    agentsStore.load().catch((e: Error) => { error.value = e.message })
  }
})

async function loadSessions() {
  try {
    const r = await api.listSessions()
    sessions.value = r.sessions || []
  } catch (e: any) {
    error.value = e.message
  }
}

async function selectSession(key: string) {
  stopWatch()
  chatStore.closeDrawer()
  currentKey.value = key
  messages.value = []
  try {
    const r = await api.getSession(key)
    messages.value = mapStoredMessages(r.messages || [])
    sessionAgentKey.value = r.session?.agent_key || ''
    selectedAgentKey.value = sessionAgentKey.value || selectedAgentKey.value
    chatStore.setSession(key, r.session?.title || '')
    await nextTick()
    scrollBottom()
    startWatch(key)
  } catch (e: any) {
    error.value = e.message
  }
}

function newSession() {
  stopWatch()
  currentKey.value = 'sess-' + Math.random().toString(36).slice(2, 10)
  sessionAgentKey.value = ''
  messages.value = []
  chatStore.setSession(currentKey.value, '')
  chatStore.closeDrawer()
  startWatch(currentKey.value)
}

function chatAgentKey(): string {
  return sessionAgentKey.value || selectedAgentKey.value || 'default'
}

async function send() {
  if (!input.value.trim() || streaming.value) return
  if (!currentKey.value) newSession()
  const text = input.value
  input.value = ''
  messages.value.push({ role: 'user', content: text })
  let cur: Msg | null = pushAssistant()
  streaming.value = true
  error.value = ''
  await nextTick()
  scrollBottom()
  try {
    await chat(currentKey.value, text, chatAgentKey(), (ev) => {
      handleChatEvent(ev, () => cur, (m) => { cur = m })
    })
  } catch (e: any) {
    error.value = e.message
  } finally {
    if (cur) cur.streaming = false
    streaming.value = false
    loadSessions()
  }
}

async function abortRun() {
  if (!currentKey.value) return
  try {
    await api.abortSession(currentKey.value)
  } catch {}
}

function scrollBottom() {
  if (scrollEl.value) scrollEl.value.scrollTop = scrollEl.value.scrollHeight
}

function render(content: string) {
  const html = md.render(content || '')
  const el = document.createElement('div')
  el.innerHTML = html
  el.querySelectorAll('pre code').forEach((node) => {
    hljs.highlightElement(node as HTMLElement)
  })
  el.querySelectorAll('table').forEach((table) => {
    const wrap = document.createElement('div')
    wrap.className = 'prose-table-wrap'
    table.parentNode?.insertBefore(wrap, table)
    wrap.appendChild(table)
  })
  return el.innerHTML
}

function gapClass(m: Msg, i: number): string {
  if (i === 0) return ''
  if (m.role === 'tool') return 'mt-1'
  if (m.role === 'assistant' && m.thinking) return 'mt-1'
  return 'mt-4'
}
</script>

<template>
  <div class="h-full">
    <div
      v-if="chatStore.drawerOpen"
      class="fixed inset-0 bg-black/20 z-30"
      @click="chatStore.closeDrawer()"
    ></div>
    <div
      v-if="chatStore.drawerOpen"
      class="absolute left-0 top-0 bottom-0 w-64 bg-white border-r border-neutral-200 z-40 flex flex-col"
    >
      <button class="px-4 py-3 bg-neutral-800 text-white text-sm border-y border-neutral-800" @click="newSession">
        + New session
      </button>
      <div class="flex-1 overflow-y-auto">
        <div
          v-for="s in sessions"
          :key="s.key"
          class="px-4 py-2 cursor-pointer hover:bg-neutral-100 truncate text-sm border-b border-neutral-100"
          :class="{ 'bg-neutral-100': s.key === currentKey }"
          @click="selectSession(s.key)"
        >
          <div class="truncate">{{ s.title || s.key }}</div>
          <div class="text-xs text-neutral-400">{{ s.agent_key }}</div>
        </div>
      </div>
    </div>

    <div class="h-full flex flex-col min-w-0">
      <div ref="scrollEl" class="flex-1 overflow-y-auto">
        <div class="max-w-[960px] mx-auto px-4 py-6">
          <div v-if="!auth.isAuthed" class="text-neutral-500">Authenticate to start chatting.</div>
          <template v-else>
            <div v-for="(m, i) in messages" :key="i" :class="gapClass(m, i)">
              <div v-if="m.role === 'user'" class="flex justify-end">
                <div class="bg-neutral-800 text-white rounded-lg px-3 py-2 max-w-[80%] whitespace-pre-wrap">{{ m.content }}</div>
              </div>
              <div v-else-if="m.role === 'assistant'">
                <ThinkingBlock v-if="m.thinking" :content="m.thinking" />
                <div v-if="m.content" class="prose-mira" v-html="render(m.content)"></div>
                <span v-else-if="m.streaming" class="typing-dots">
                  <span></span><span></span><span></span>
                </span>
              </div>
              <div v-else-if="m.role === 'tool'">
                <ToolCallBlock :name="m.tool_name || ''" :args="m.arguments" :content="m.content" :is-error="m.isError" />
              </div>
            </div>
            <div v-if="error" class="text-red-600 text-sm">{{ error }}</div>
          </template>
        </div>
      </div>

      <div>
        <div class="max-w-[960px] mx-auto p-3 flex flex-col gap-2">
          <div v-if="!sessionAgentKey" class="flex items-center gap-2 text-sm">
            <label>Agent</label>
            <select v-model="selectedAgentKey" class="border rounded px-2 py-1">
              <option v-for="a in agentsStore.agents" :key="a.key" :value="a.key">{{ a.key }}</option>
            </select>
          </div>
          <div class="flex gap-2">
            <textarea
              v-model="input"
              @keydown.enter.exact.prevent="send"
              rows="1"
              class="flex-1 border border-neutral-300 rounded px-3 py-2 resize-none focus:outline-none focus:border-neutral-500"
              placeholder="Message…"
            ></textarea>
            <button v-if="!streaming" class="px-4 py-2 bg-neutral-800 text-white rounded" @click="send">Send</button>
            <button v-else class="px-4 py-2 bg-red-600 text-white rounded" @click="abortRun">Abort</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
