<script setup lang="ts">
import { ref, onMounted, nextTick, watch, computed } from 'vue'
import { api, chat, watchSession } from '../api'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { useLayoutStore } from '../stores/layout'
import { DEFAULT_AGENT_KEY } from '../constants/defaults'
import { renderMarkdown } from '../lib/markdown'
import LocalSvgIcon from '../components/LocalSvgIcon.vue'
import UploadFileBar from '../components/UploadFileBar.vue'
import ToolCallBlock from '../components/ToolCallBlock.vue'
import ThinkingBlock from '../components/ThinkingBlock.vue'
import SubagentDrawer from '../components/SubagentDrawer.vue'
import { cancelClarify } from '../lib/windowBridge'
import { useClarifyStore } from '../stores/clarify'
import { handleUiRequest } from '../lib/windowBridge'
import { submitClarifyAnswer } from '../lib/windowBridge'
import { composeMessageWithAttachments } from '../lib/workspacePath'
import { displayMessageBody, fromAtPath } from '../lib/workspacePath'
import type { ChatEvent, Message, Session } from '../types'

const props = withDefaults(
  defineProps<{
    expanded?: boolean
    /** Bound session for a maximized chat tab; omit for the sidebar panel. */
    sessionKey?: string
  }>(),
  { expanded: false },
)

const auth = useAuthStore()
const chatStore = useChatStore()
const layout = useLayoutStore()
const clarifyStore = useClarifyStore()

const clarifyAnswer = ref('')
const clarifyPending = computed(() => {
  const key = currentKey.value
  return key ? clarifyStore.bySession[key] || null : null
})

async function answerClarify(text?: string) {
  const key = currentKey.value
  if (!key) return
  const ans = (text ?? clarifyAnswer.value).trim()
  if (!ans) return
  clarifyAnswer.value = ''
  try {
    await submitClarifyAnswer(key, ans)
  } catch (e: any) {
    error.value = e.message
  }
}

async function dismissClarify() {
  const key = currentKey.value
  if (!key) return
  await cancelClarify(key)
}

const pendingAttachments = computed(() => {
  const key = currentKey.value
  return key ? chatStore.pendingBySession[key] || [] : []
})

function removePending(atPath: string) {
  const key = currentKey.value
  if (!key) return
  chatStore.removePending(key, atPath)
}

function openAttached(atPath: string) {
  const rel = fromAtPath(atPath)
  if (rel) layout.openFile(rel)
}

function messageDisplayBody(content: string) {
  return displayMessageBody(content)
}

function shortFileLabel(at: string): string {
  const name = fromAtPath(at).split('/').pop() || at
  return name.length > 22 ? name.slice(0, 10) + '…' + name.slice(-8) : name
}

const isTabMode = computed(() => !!props.sessionKey)
const localTitle = ref('')

function maximizeChat() {
  let key = chatStore.currentKey
  if (!key) {
    key = 'sess-' + Math.random().toString(36).slice(2, 10)
    chatStore.setSession(key, '')
  }
  layout.openChatTab(key, chatStore.currentTitle || 'New Chat')
}

function restoreChatSidebar() {
  if (props.sessionKey) {
    chatStore.setSession(props.sessionKey, localTitle.value || '')
    layout.exitChatTab(props.sessionKey)
  } else {
    layout.exitChatTab()
  }
}

interface Msg {
  role: string
  content: string
  thinking?: string
  tool_name?: string
  id?: string
  arguments?: Record<string, unknown>
  isError?: boolean
  streaming?: boolean
  progress?: string
  childSession?: string
  startedAt?: number
  endedAt?: number
}

const sessions = ref<Session[]>([])
const currentKey = computed(() => (isTabMode.value ? props.sessionKey || '' : chatStore.currentKey))
const showHistory = ref(false)
const subagentKey = ref<string | null>(null)

function openSubagent(key: string) {
  if (key) subagentKey.value = key
}
const messages = ref<Msg[]>([])
const input = ref('')
const streaming = ref(false)
const error = ref('')
const scrollEl = ref<HTMLElement | null>(null)
let watchAbort: AbortController | null = null
let bootstrapped = false
/** Key whose messages are currently loaded in this panel. */
let loadedKey = ''
/** Skip auto-scroll when the user has scrolled up to read history. */
let stickToBottom = true
let scrollRaf = 0

const headerTitle = computed(() => {
  if (showHistory.value) return 'History'
  if (!currentKey.value) return 'New Chat'
  if (isTabMode.value) return localTitle.value || currentKey.value
  return chatStore.currentTitle || currentKey.value
})

function setSessionMeta(key: string, title: string) {
  if (isTabMode.value) {
    localTitle.value = title
    if (title) layout.renameChatTab(key, title)
  } else {
    chatStore.setSession(key, title)
  }
}

function openHistory() {
  showHistory.value = true
  loadSessions()
}

function closeHistory() {
  showHistory.value = false
}

async function pickSession(key: string) {
  if (isTabMode.value) {
    // Another (or same) session opens as its own tab; same key reuses that tab.
    const known = sessions.value.find((s) => s.id === key)
    layout.openChatTab(key, known?.title || '')
    showHistory.value = false
    return
  }
  await selectSession(key)
  showHistory.value = false
}

async function loadSessions() {
  try {
    const r = await api.listSessions()
    sessions.value = r.sessions || []
    if (!currentKey.value) return
    const s = sessions.value.find((x) => x.id === currentKey.value)
    if (s?.title) setSessionMeta(s.id, s.title)
  } catch {}
}

function handleChatEvent(ev: ChatEvent, getCur: () => Msg | null, setCur: (m: Msg | null) => void) {
  if (handleUiRequest(ev, currentKey.value)) return
  if (ev.type === 'queued') {
    messages.value.push({
      role: 'assistant',
      content: `Queued (position ${ev.position ?? '?'}). Will run after the current turn.`,
    })
    scrollBottom()
    return
  }
  if (ev.type === 'user') {
    const last = messages.value[messages.value.length - 1]
    if (last?.role === 'user' && last.content === (ev.content || '')) {
      streaming.value = true
      const cur = pushAssistant()
      setCur(cur)
      scrollBottom()
      return
    }
    messages.value.push({ role: 'user', content: ev.content || '' })
    streaming.value = true
    let cur = pushAssistant()
    setCur(cur)
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
      startedAt: Date.now(),
    })
  } else if (ev.type === 'tool_progress') {
    const t = messages.value.find((m) => m.role === 'tool' && m.id === ev.id)
    if (t) {
      if (ev.child) t.childSession = ev.child
      if (ev.content) t.progress = ev.content
    }
  } else if (ev.type === 'tool_result') {
    const t = messages.value.find((m) => m.role === 'tool' && m.id === ev.id)
    if (t) {
      t.content = ev.result || ''
      t.isError = !!ev.is_error || looksLikeToolError(ev.result)
      t.endedAt = Date.now()
    }
  } else if (ev.type === 'error') {
    error.value = ev.error || 'error'
  } else if (ev.type === 'done') {
    if (cur) {
      cur.streaming = false
      setCur(null)
    }
    streaming.value = false
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
  const toolByID = new Map<string, Msg>()

  for (const m of raw) {
    if (m.role === 'assistant' && m.tool_calls?.length) {
      const tcs = m.tool_calls
      if (m.content || m.thinking) {
        out.push({ role: 'assistant', content: m.content, thinking: m.thinking })
      }
      for (const tc of tcs) {
        const entry: Msg = {
          role: 'tool',
          id: tc.id,
          tool_name: tc.name,
          arguments: tc.arguments,
          content: '',
          isError: false,
        }
        out.push(entry)
        if (tc.id) toolByID.set(tc.id, entry)
      }
      continue
    }
    if (m.role === 'tool') {
      const isErr = looksLikeToolError(m.content)
      const existing = m.tool_call_id ? toolByID.get(m.tool_call_id) : undefined
      if (existing) {
        existing.content = m.content || ''
        existing.isError = isErr
        if (m.tool_name) existing.tool_name = m.tool_name
      } else {
        out.push({
          role: 'tool',
          id: m.tool_call_id,
          tool_name: m.tool_name,
          content: m.content || '',
          isError: isErr,
        })
      }
      continue
    }
    out.push({ role: m.role, content: m.content, thinking: m.thinking })
  }
  return out
}

function looksLikeToolError(content: string | undefined): boolean {
  if (!content) return false
  return /^error:/i.test(content.trim())
}

async function selectSession(key: string) {
  if (!key) return
  stopWatch()
  const known = sessions.value.find((s) => s.id === key)
  const fallbackTitle = isTabMode.value
    ? localTitle.value
    : key === chatStore.currentKey
      ? chatStore.currentTitle
      : ''
  setSessionMeta(key, known?.title || fallbackTitle || '')
  loadedKey = key
  messages.value = []
  error.value = ''
  try {
    const r = await api.getSession(key)
    messages.value = mapStoredMessages(r.messages || [])
    const title = r.session?.title || known?.title || ''
    if (title) setSessionMeta(key, title)
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    // Draft key not yet on server — keep empty thread with this id.
    if (!/not found|404/i.test(msg)) {
      error.value = msg
    }
  }
  startWatch(key)
  await nextTick()
  scrollBottom(true)
}

async function restoreLastSession() {
  await loadSessions()
  const key = chatStore.currentKey
  if (key) {
    await selectSession(key)
    return
  }
  if (sessions.value.length) {
    await selectSession(sessions.value[0].id)
  }
}

async function bootstrap() {
  if (!auth.isAuthed || bootstrapped) return
  bootstrapped = true
  if (props.sessionKey) {
    await loadSessions()
    await selectSession(props.sessionKey)
    return
  }
  await restoreLastSession()
}

onMounted(() => {
  bootstrap()
})

watch(() => auth.isAuthed, (v) => {
  if (v) {
    bootstrapped = false
    bootstrap()
  } else {
    stopWatch()
    messages.value = []
    loadedKey = ''
    bootstrapped = false
  }
})

// Sidebar: external session switches (e.g. Welcome when not opening a tab)
watch(
  () => chatStore.currentKey,
  (key) => {
    if (isTabMode.value) return
    if (!bootstrapped || !auth.isAuthed) return
    if (!key || key === loadedKey) return
    selectSession(key)
  },
)

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
        // Allow queue-continued runs via hub; drop only mid-stream duplicates except user/done.
        if (streaming.value && ev.type !== 'user' && ev.type !== 'done' && ev.type !== 'error') return
        handleChatEvent(ev, () => null, (_m) => { /* no-op */ })
      }, ac.signal)
    } catch (e: unknown) {
      if (e instanceof Error && e.name === 'AbortError') return
    }
    if (ac.signal.aborted || currentKey.value !== key) return
    await new Promise((r) => setTimeout(r, 1000))
  }
}

function newSession() {
  const key = 'sess-' + Math.random().toString(36).slice(2, 10)
  if (isTabMode.value) {
    layout.openChatTab(key, 'New Chat')
    showHistory.value = false
    return
  }
  stopWatch()
  chatStore.setSession(key, '')
  loadedKey = key
  messages.value = []
  error.value = ''
  showHistory.value = false
  startWatch(key)
}

async function send() {
  const pending = currentKey.value
    ? [...(chatStore.pendingBySession[currentKey.value] || [])]
    : []
  if (!input.value.trim() && !pending.length) return
  if (!currentKey.value) {
    if (isTabMode.value) return
    newSession()
  }
  const key = currentKey.value
  if (!key) return
  const text = composeMessageWithAttachments(
    input.value,
    pending.map((p) => p.atPath),
  )
  input.value = ''
  chatStore.clearPending(key)
  const wasStreaming = streaming.value
  messages.value.push({ role: 'user', content: text })
  error.value = ''
  await nextTick()
  scrollBottom(true)

  if (wasStreaming) {
    try {
      await chat(key, text, DEFAULT_AGENT_KEY, (ev) => {
        handleChatEvent(ev, () => null, () => {})
      })
    } catch (e: any) {
      error.value = e.message
    }
    return
  }

  let cur: Msg | null = pushAssistant()
  streaming.value = true
  try {
    await chat(key, text, DEFAULT_AGENT_KEY, (ev) => {
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
    messages.value.push({
      role: 'assistant',
      content: 'Aborted. Queued messages (if any) will continue after this turn ends.',
    })
  } catch {}
}

function onMessagesScroll() {
  const el = scrollEl.value
  if (!el) return
  const gap = el.scrollHeight - el.scrollTop - el.clientHeight
  stickToBottom = gap < 80
}

function scrollBottom(force = false) {
  if (!force && !stickToBottom) return
  if (scrollRaf) cancelAnimationFrame(scrollRaf)
  scrollRaf = requestAnimationFrame(() => {
    scrollRaf = 0
    const el = scrollEl.value
    if (!el) return
    el.scrollTop = el.scrollHeight
  })
}

function render(content: string) {
  return renderMarkdown(content)
}

function gapClass(m: Msg, i: number): string {
  if (i === 0) return ''
  if (m.role === 'tool') return 'mt-1'
  if (m.role === 'assistant' && m.thinking) return 'mt-1'
  return 'mt-4'
}
</script>

<template>
  <div class="h-full flex flex-col min-w-0">
    <!-- Header -->
    <div
      class="shrink-0 w-full"
      :class="props.expanded ? 'max-w-[960px] mx-auto px-4' : 'border-b border-neutral-200'"
    >
      <div
        class="h-9 flex items-center justify-between gap-2"
        :class="[
          props.expanded ? 'border-b border-neutral-200' : '',
          showHistory ? 'pl-1.5 pr-2' : (props.expanded ? '' : 'px-3'),
        ]"
      >
        <div class="flex items-center gap-1 min-w-0">
          <button
            v-if="showHistory"
            type="button"
            class="shrink-0 w-7 h-7 flex items-center justify-center rounded hover:bg-neutral-100 text-neutral-600"
            title="Back"
            @click="closeHistory"
          >
            <LocalSvgIcon name="back" :size="16" />
          </button>
          <span class="text-sm font-medium truncate leading-none">{{ headerTitle }}</span>
        </div>
        <div class="shrink-0 flex items-center gap-0.5">
          <button
            v-if="streaming && !showHistory"
            type="button"
            class="w-7 h-7 flex items-center justify-center rounded hover:bg-neutral-100 text-neutral-500"
            title="Abort"
            @click="abortRun"
          >
            <LocalSvgIcon name="stop" :size="13" />
          </button>
          <button
            v-if="!showHistory && !props.expanded"
            type="button"
            class="w-7 h-7 flex items-center justify-center rounded hover:bg-neutral-100 text-neutral-600"
            title="Maximize chat"
            @click="maximizeChat"
          >
            <LocalSvgIcon name="maximize" :size="16" />
          </button>
          <button
            v-if="!showHistory && props.expanded"
            type="button"
            class="w-7 h-7 flex items-center justify-center rounded hover:bg-neutral-100 text-neutral-600"
            title="Restore sidebar"
            @click="restoreChatSidebar"
          >
            <LocalSvgIcon name="minimize" :size="16" />
          </button>
          <button
            v-if="!showHistory"
            type="button"
            class="w-7 h-7 flex items-center justify-center rounded hover:bg-neutral-100 text-neutral-600"
            title="History"
            @click="openHistory"
          >
            <LocalSvgIcon name="history" :size="16" />
          </button>
          <button
            v-else
            type="button"
            class="h-7 px-2.5 text-xs rounded bg-neutral-800 text-white hover:bg-neutral-700"
            @click="newSession"
          >New Chat</button>
        </div>
      </div>
    </div>

    <!-- History list -->
    <div v-if="showHistory" class="flex-1 overflow-y-auto">
      <div class="w-full" :class="props.expanded ? 'max-w-[960px] mx-auto' : ''">
        <div v-if="!sessions.length" class="p-6 text-base text-neutral-400 text-center">No sessions yet</div>
        <div v-else class="py-1">
          <button
            v-for="s in sessions"
            :key="s.id"
            type="button"
            class="w-full text-left pl-3 pr-4 py-2.5 text-base hover:bg-neutral-50 border-b border-neutral-100 flex items-center gap-2"
            :class="s.id === currentKey ? 'bg-neutral-50 font-medium' : ''"
            @click="pickSession(s.id)"
          >
            <span class="truncate flex-1">{{ s.title || s.id }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Messages -->
    <div v-else ref="scrollEl" class="flex-1 overflow-y-auto" @scroll.passive="onMessagesScroll">
      <div class="w-full p-4" :class="props.expanded ? 'max-w-[960px] mx-auto' : ''">
        <div v-if="!auth.isAuthed" class="text-neutral-500">Authenticate to start chatting.</div>
        <template v-else>
          <div v-for="(m, i) in messages" :key="i" :class="gapClass(m, i)">
            <div v-if="m.role === 'user'" class="flex justify-end">
              <div class="max-w-[80%] flex flex-col items-end gap-1.5">
                <UploadFileBar :content="m.content" @open="openAttached" />
                <div
                  v-if="messageDisplayBody(m.content)"
                  class="bg-neutral-800 text-white text-[15px] leading-relaxed rounded-lg px-3 py-2 whitespace-pre-wrap"
                >{{ messageDisplayBody(m.content) }}</div>
              </div>
            </div>
            <div v-else-if="m.role === 'assistant'">
              <ThinkingBlock v-if="m.thinking" :content="m.thinking" />
              <div v-if="m.content" class="prose-swiflow" v-html="render(m.content)"></div>
              <span v-else-if="m.streaming" class="typing-dots">
                <span></span><span></span><span></span>
              </span>
            </div>
            <div v-else-if="m.role === 'tool'">
              <ToolCallBlock
                :name="m.tool_name || ''"
                :args="m.arguments"
                :content="m.content"
                :is-error="m.isError"
                :progress="m.progress"
                :child-session="m.childSession"
                :started-at="m.startedAt"
                :ended-at="m.endedAt"
                @view-child="(key) => openSubagent(key)"
              />
            </div>
          </div>
          <div v-if="error" class="text-red-600 text-sm">{{ error }}</div>
        </template>
      </div>
    </div>

    <!-- Clarify prompt (agent → user) -->
    <div
      v-if="clarifyPending && !showHistory"
      class="shrink-0 w-full"
      :class="props.expanded ? 'max-w-[960px] mx-auto px-4 pb-2' : 'px-3 pb-2'"
    >
      <div class="border border-amber-200 bg-amber-50/80 rounded-lg p-3 space-y-2">
        <div class="text-sm font-medium text-neutral-800">{{ clarifyPending.question }}</div>
        <div v-if="clarifyPending.options.length" class="flex flex-wrap gap-2">
          <button
            v-for="opt in clarifyPending.options"
            :key="opt"
            type="button"
            class="px-2.5 py-1 text-sm rounded border border-neutral-300 bg-white hover:bg-neutral-100"
            @click="answerClarify(opt)"
          >{{ opt }}</button>
        </div>
        <div v-if="clarifyPending.allowFreeText" class="flex gap-2">
          <input
            v-model="clarifyAnswer"
            type="text"
            class="flex-1 text-sm border border-neutral-200 rounded px-2 py-1.5 bg-white focus:outline-none focus:border-neutral-400"
            placeholder="Your answer…"
            @keydown.enter.prevent="answerClarify()"
          />
          <button
            type="button"
            class="px-3 py-1.5 text-sm rounded bg-neutral-800 text-white hover:bg-neutral-700 disabled:opacity-40"
            :disabled="!clarifyAnswer.trim()"
            @click="answerClarify()"
          >Send</button>
        </div>
        <button type="button" class="text-xs text-neutral-500 hover:underline" @click="dismissClarify">Cancel</button>
      </div>
    </div>

    <!-- Input: same column width as messages (border lives on the column, not full page) -->
    <div
      v-if="!showHistory"
      class="shrink-0 w-full"
      :class="props.expanded ? 'max-w-[960px] mx-auto px-4 pb-4' : ''"
    >
      <div
        class="relative"
        :class="props.expanded
          ? 'border border-neutral-200 rounded-xl p-3'
          : 'border-t border-neutral-200 p-3'"
      >
        <div
          v-if="pendingAttachments.length"
          class="flex items-center gap-1.5 mb-2 pb-2 border-b border-neutral-100 min-w-0 overflow-hidden"
        >
          <span
            v-for="f in pendingAttachments.slice(0, 2)"
            :key="f.atPath"
            class="inline-flex items-center gap-1 min-w-0 max-w-[42%] pl-2 pr-1 py-0.5 rounded-md text-xs bg-neutral-100 text-neutral-700"
            :title="f.atPath"
          >
            <button
              type="button"
              class="font-mono truncate hover:underline text-left min-w-0"
              @click="openAttached(f.atPath)"
            >{{ shortFileLabel(f.atPath) }}</button>
            <button
              type="button"
              class="shrink-0 w-5 h-5 flex items-center justify-center rounded hover:bg-neutral-200 text-neutral-500"
              title="Remove"
              @click="removePending(f.atPath)"
            >×</button>
          </span>
          <span
            v-if="pendingAttachments.length > 2"
            class="shrink-0 text-xs text-neutral-500 tabular-nums"
            :title="pendingAttachments.map(f => f.atPath).join('\n')"
          >+{{ pendingAttachments.length - 2 }} · {{ pendingAttachments.length }} files</span>
          <span
            v-else-if="pendingAttachments.length > 1"
            class="shrink-0 text-xs text-neutral-500 tabular-nums"
          >{{ pendingAttachments.length }} files</span>
        </div>
        <textarea
          v-model="input"
          @keydown.enter.exact.prevent="send"
          rows="3"
          class="w-full border-0 bg-transparent px-0 py-0 pr-10 pb-9 text-[15px] leading-relaxed resize-none focus:outline-none"
          :placeholder="streaming ? 'Message (queued while running)…' : 'Message…'"
        ></textarea>
        <button
          type="button"
          class="absolute right-3 bottom-3 w-8 h-8 flex items-center justify-center rounded-md bg-neutral-800 text-white hover:bg-neutral-700 disabled:opacity-35 disabled:hover:bg-neutral-800"
          :disabled="!input.trim() && !pendingAttachments.length"
          :title="streaming ? 'Queue message' : 'Send'"
          @click="send"
        >
          <LocalSvgIcon name="send" :size="16" />
        </button>
      </div>
    </div>

    <SubagentDrawer v-if="subagentKey" :session-key="subagentKey" @close="subagentKey = null" />
  </div>
</template>
