<script setup lang="ts">
import { ref, onMounted, nextTick, watch } from 'vue'
import { api, chat } from '../api'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import MarkdownIt from 'markdown-it'
import ToolCallBlock from '../components/ToolCallBlock.vue'
import ThinkingBlock from '../components/ThinkingBlock.vue'

const md = new MarkdownIt({ html: false, linkify: true, breaks: true })
const auth = useAuthStore()
const chatStore = useChatStore()

interface Msg {
  role: string
  content: string
  thinking?: string
  tool_name?: string
  id?: string
  arguments?: any
  isError?: boolean
  streaming?: boolean
}

const sessions = ref<any[]>([])
const currentKey = ref('')
const messages = ref<Msg[]>([])
const input = ref('')
const streaming = ref(false)
const error = ref('')
const scrollEl = ref<HTMLElement | null>(null)

// Create a fresh assistant bubble for the current LLM round and return it.
function pushAssistant(): Msg {
  const a: Msg = { role: 'assistant', content: '', streaming: true }
  messages.value.push(a)
  return a
}

onMounted(() => {
  if (auth.isAuthed) loadSessions()
})
watch(() => auth.isAuthed, (v) => {
  if (v) loadSessions()
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
  chatStore.closeDrawer()
  currentKey.value = key
  messages.value = []
  try {
    const r = await api.getSession(key)
    messages.value = (r.messages || []).map((m: any) => ({
      role: m.role,
      content: m.content,
      thinking: m.thinking,
      tool_name: m.tool_name,
      isError: false,
    }))
    chatStore.setSession(key, r.session?.title || '')
    await nextTick()
    scrollBottom()
  } catch (e: any) {
    error.value = e.message
  }
}

function newSession() {
  currentKey.value = 'sess-' + Math.random().toString(36).slice(2, 10)
  messages.value = []
  chatStore.setSession(currentKey.value, '')
  chatStore.closeDrawer()
}

async function send() {
  if (!input.value.trim() || streaming.value) return
  if (!currentKey.value) newSession()
  const text = input.value
  input.value = ''
  messages.value.push({ role: 'user', content: text })
  // One assistant bubble per LLM round. The first round's bubble is created
  // up front so the "…" indicator shows; later rounds create one on the next
  // delta.
  let cur: Msg | null = pushAssistant()
  streaming.value = true
  error.value = ''
  await nextTick()
  scrollBottom()
  try {
    await chat(currentKey.value, text, '', (ev) => {
      if (ev.type === 'delta') {
        if (!cur) cur = pushAssistant()
        cur.content += ev.content
      } else if (ev.type === 'thinking') {
        if (!cur) cur = pushAssistant()
        cur.thinking = (cur.thinking || '') + ev.content
      } else if (ev.type === 'tool_call') {
        // Close the current round's assistant bubble (the "intent" text before
        // the call). Drop it if it produced no text so we don't show empties.
        if (cur) {
          cur.streaming = false
          if (!cur.content && !cur.thinking) {
            messages.value.splice(messages.value.length - 1, 1)
          }
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
          t.content = ev.result
          t.isError = ev.is_error
        }
      } else if (ev.type === 'error') {
        error.value = ev.error
      } else if (ev.type === 'done') {
        if (cur) {
          cur.streaming = false
          cur = null
        }
        if (ev.title) {
          chatStore.currentTitle = ev.title
          const s = sessions.value.find((s) => s.key === currentKey.value)
          if (s) s.title = ev.title
          else loadSessions()
        }
      }
      scrollBottom()
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
  return md.render(content || '')
}

// Per-message top gap: tool rows and thinking-bearing assistant bubbles are
// tight (2px); user messages and final replies keep the normal 16px gap.
function gapClass(m: Msg, i: number): string {
  if (i === 0) return ''
  if (m.role === 'tool') return 'mt-1'
  if (m.role === 'assistant' && m.thinking) return 'mt-1'
  return 'mt-4'
}
</script>

<template>
  <div class="h-full">
    <!-- sessions drawer -->
    <div
      v-if="chatStore.drawerOpen"
      class="fixed inset-0 bg-black/20 z-30"
      @click="chatStore.closeDrawer()"
    ></div>
    <div
      v-if="chatStore.drawerOpen"
      class="absolute left-0 top-0 bottom-0 w-64 bg-white border-r border-neutral-200 z-40 flex flex-col"
    >
      <button class="px-4 py-2 bg-neutral-800 text-white text-sm" @click="newSession">+ New session</button>
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

    <!-- chat column -->
    <div class="h-full flex flex-col min-w-0">
      <div ref="scrollEl" class="flex-1 overflow-y-auto">
        <div class="max-w-[960px] mx-auto px-4 py-6">
          <div v-if="!auth.isAuthed" class="text-neutral-500">Authenticate to start chatting.</div>
          <template v-else>
            <div
              v-for="(m, i) in messages"
              :key="i" :class="gapClass(m, i)"
            >
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
        <div class="max-w-[960px] mx-auto p-3 flex gap-2">
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
</template>
