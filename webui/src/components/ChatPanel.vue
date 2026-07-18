<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, watch, computed } from 'vue'
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
import HarnessPanel from '../components/HarnessPanel.vue'
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
  durationMs?: number
}

const sessions = ref<Session[]>([])
const currentKey = computed(() => (isTabMode.value ? props.sessionKey || '' : chatStore.currentKey))
const showHistory = ref(false)
const subagentKey = ref<string | null>(null)
const showHarness = ref(false)
const harnessWarns = ref<{ code: string; message: string; child?: string }[]>([])

function openSubagent(key: string) {
  if (key) subagentKey.value = key
}

function toolActivityLabel(m: Msg): string {
  const name = m.tool_name || 'tool'
  const a = m.arguments || {}
  const pick = (k: string) => {
    const v = a[k]
    return v == null ? '' : String(v)
  }
  const trim = (s: string, n: number) => (s.length > n ? s.slice(0, n) + '…' : s)
  switch (name) {
    case 'fs_read':
      return '读取 ' + trim(pick('path'), 40)
    case 'fs_write':
      return '写入 ' + trim(pick('path'), 40)
    case 'fs_edit':
      return '编辑 ' + trim(pick('path'), 40)
    case 'content_extract':
      return '内容抽取 ' + trim(pick('path'), 40)
    case 'web_fetch':
      return '抓取 ' + trim(pick('url'), 40)
    case 'web_search':
      return '搜索 ' + trim(pick('query'), 40)
    case 'browser':
      return '浏览器 ' + (pick('action') || '')
    case 'delegate_task':
      return '委派 ' + trim(pick('goal'), 40)
    default:
      return name
  }
}

const messages = ref<Msg[]>([])
const input = ref('')
const streaming = ref(false)
const error = ref('')
const scrollEl = ref<HTMLElement | null>(null)
let watchAbort: AbortController | null = null
let bootstrapped = false
/** True only while the POST /chat SSE body is being consumed (not watch). */
let chatStreamActive = false
/** Key whose messages are currently loaded in this panel. */
let loadedKey = ''
/** Skip auto-scroll when the user has scrolled up to read history. */
let stickToBottom = true
let scrollRaf = 0
let resyncTimer = 0

/** Live line above the input: current tool / thinking / subagent / idle / harness warn. */
const statusBar = computed(() => {
  const latestWarn = harnessWarns.value.length
    ? harnessWarns.value[harnessWarns.value.length - 1]
    : null

  if (streaming.value) {
    for (let i = messages.value.length - 1; i >= 0; i--) {
      const m = messages.value[i]
      if (m.role !== 'tool') continue
      if (m.content || m.isError) continue
      if (m.tool_name === 'delegate_task') {
        return {
          kind: 'sub' as const,
          text: m.progress || toolActivityLabel(m) || '子任务运行中…',
          warn: !!latestWarn,
        }
      }
      return {
        kind: 'tool' as const,
        text: toolActivityLabel(m),
        warn: !!latestWarn,
      }
    }
    for (let i = messages.value.length - 1; i >= 0; i--) {
      const m = messages.value[i]
      if (m.role !== 'assistant') continue
      if (m.streaming || (!m.content && m.thinking)) {
        if (m.thinking && !m.content) {
          return { kind: 'think' as const, text: '思考中…', warn: !!latestWarn }
        }
        if (m.streaming && m.content) {
          return { kind: 'reply' as const, text: '生成回复中…', warn: !!latestWarn }
        }
        return { kind: 'think' as const, text: '思考中…', warn: !!latestWarn }
      }
      break
    }
    return { kind: 'run' as const, text: '运行中…', warn: !!latestWarn }
  }

  if (latestWarn) {
    return {
      kind: 'warn' as const,
      text: latestWarn.message || latestWarn.code,
      warn: true,
    }
  }
  if (error.value) {
    return { kind: 'error' as const, text: error.value, warn: false }
  }
  // Keep the bar after the run so status + harness entry stay visible.
  const hasTurn = messages.value.some((m) => m.role === 'assistant' || m.role === 'tool')
  return {
    kind: 'idle' as const,
    text: hasTurn ? '已完成' : '就绪',
    warn: false,
  }
})

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

/** In-app confirm — window.confirm is unavailable / broken under Wails. */
const deleteTarget = ref<Session | null>(null)
const deleting = ref(false)

const deleteTargetLabel = computed(() => {
  const s = deleteTarget.value
  if (!s) return ''
  return (s.title || s.id).trim() || s.id
})

function askDeleteSession(s: Session) {
  deleteTarget.value = s
}

function cancelDeleteSession() {
  if (deleting.value) return
  deleteTarget.value = null
}

async function confirmDeleteSession() {
  const s = deleteTarget.value
  if (!s) return
  deleting.value = true
  try {
    await api.deleteSession(s.id)
    sessions.value = sessions.value.filter((x) => x.id !== s.id)
    chatStore.clearPending(s.id)
    layout.closeTab('chat:' + s.id)
    deleteTarget.value = null
    if (currentKey.value !== s.id) return
    watchAbort?.abort()
    watchAbort = null
    messages.value = []
    streaming.value = false
    error.value = ''
    chatStore.clearSession()
    if (sessions.value.length) {
      await pickSession(sessions.value[0].id)
    }
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'delete failed'
  } finally {
    deleting.value = false
  }
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
      if (typeof ev.duration_ms === 'number') t.durationMs = ev.duration_ms
    }
  } else if (ev.type === 'harness_warn') {
    const code = ev.name || 'drift'
    const message = ev.content || code
    harnessWarns.value.push({ code, message, child: ev.child })
    if (harnessWarns.value.length > 20) harnessWarns.value.shift()
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

function mapStoredMessages(raw: Message[], opts?: { runActive?: boolean }): Msg[] {
  const out: Msg[] = []
  const toolByID = new Map<string, Msg>()
  const runActive = !!opts?.runActive

  for (const m of raw) {
    if (m.role === 'assistant' && m.tool_calls?.length) {
      const tcs = m.tool_calls
      if (m.content || m.thinking) {
        out.push({ role: 'assistant', content: m.content, thinking: m.thinking })
      }
      const startedAt = parseTs(m.created_at)
      for (const tc of tcs) {
        const entry: Msg = {
          role: 'tool',
          id: tc.id,
          tool_name: tc.name,
          arguments: tc.arguments,
          content: '',
          isError: false,
          startedAt,
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
        existing.endedAt = parseTs(m.created_at)
        if (m.tool_name) existing.tool_name = m.tool_name
      } else {
        out.push({
          role: 'tool',
          id: m.tool_call_id,
          tool_name: m.tool_name,
          content: m.content || '',
          isError: isErr,
          endedAt: parseTs(m.created_at),
        })
      }
      continue
    }
    out.push({ role: m.role, content: m.content, thinking: m.thinking })
  }
  // Orphan tool_calls with no persisted tool result must not look like still-running
  // tools — unless a run is still active (refresh mid-flight).
  if (!runActive) {
    for (const m of out) {
      if (m.role === 'tool' && !m.content && m.endedAt == null) {
        m.content = 'error: 工具结果未保存'
        m.isError = true
        m.endedAt = m.startedAt
      }
    }
  }
  return out
}

async function sessionRunActive(sessionKey: string): Promise<boolean> {
  try {
    const r = await api.listRuns()
    return (r.runs || []).some(
      (run) =>
        run.session_id === sessionKey &&
        (run.status === 'running' || run.status === 'queued'),
    )
  } catch {
    return false
  }
}

/** Last in-progress assistant bubble (for watch/hub events after chat SSE dies). */
function watchGetCur(): Msg | null {
  for (let i = messages.value.length - 1; i >= 0; i--) {
    const m = messages.value[i]
    if (m.role === 'assistant' && m.streaming) return m
  }
  return null
}

function watchSetCur(m: Msg | null) {
  if (m) return
  for (const msg of messages.value) {
    if (msg.role === 'assistant' && msg.streaming) msg.streaming = false
  }
}

async function reloadMessages(key: string, opts?: { quiet?: boolean }) {
  try {
    const runActive = await sessionRunActive(key)
    const r = await api.getSession(key)
    if (currentKey.value !== key) return
    messages.value = mapStoredMessages(r.messages || [], { runActive })
    streaming.value = runActive
    chatStreamActive = false
    const title = r.session?.title || ''
    if (title) setSessionMeta(key, title)
  } catch (e: unknown) {
    if (opts?.quiet) return
    const msg = e instanceof Error ? e.message : String(e)
    if (!/not found|404/i.test(msg)) error.value = msg
  }
}

function scheduleResync(key: string) {
  if (resyncTimer) window.clearTimeout(resyncTimer)
  resyncTimer = window.setTimeout(() => {
    resyncTimer = 0
    if (currentKey.value === key) {
      void reloadMessages(key, { quiet: true })
    }
  }, 200)
}

function onVisibilityChange() {
  if (document.visibilityState !== 'visible') return
  const key = currentKey.value
  if (!key || !auth.isAuthed) return
  void reloadMessages(key, { quiet: true })
  if (!watchAbort) startWatch(key)
}

function looksLikeToolError(content: string | undefined): boolean {
  if (!content) return false
  return /^error:/i.test(content.trim())
}

function parseTs(s?: string): number | undefined {
  if (!s) return undefined
  const t = Date.parse(s)
  return Number.isNaN(t) ? undefined : t
}

async function selectSession(key: string) {
  if (!key) return
  stopWatch()
  chatStreamActive = false
  streaming.value = false
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
  harnessWarns.value = []
  try {
    const runActive = await sessionRunActive(key)
    const r = await api.getSession(key)
    messages.value = mapStoredMessages(r.messages || [], { runActive })
    streaming.value = runActive
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
  document.addEventListener('visibilitychange', onVisibilityChange)
  window.addEventListener('online', onVisibilityChange)
})

onUnmounted(() => {
  stopWatch()
  chatStreamActive = false
  if (resyncTimer) window.clearTimeout(resyncTimer)
  document.removeEventListener('visibilitychange', onVisibilityChange)
  window.removeEventListener('online', onVisibilityChange)
})

watch(() => auth.isAuthed, (v) => {
  if (v) {
    bootstrapped = false
    bootstrap()
  } else {
    stopWatch()
    loadedKey = ''
    messages.value = []
    bootstrapped = false
    streaming.value = false
    chatStreamActive = false
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
        // Drop duplicates only while POST /chat SSE is alive — not merely while
        // streaming UI is true (that flag sticks after a stalled chat stream).
        if (
          chatStreamActive &&
          ev.type !== 'user' &&
          ev.type !== 'done' &&
          ev.type !== 'error' &&
          ev.type !== 'harness_warn'
        ) return
        handleChatEvent(ev, watchGetCur, watchSetCur)
      }, ac.signal)
    } catch (e: unknown) {
      if (e instanceof Error && e.name === 'AbortError') return
    }
    if (ac.signal.aborted || currentKey.value !== key) return
    // SSE dropped (stutter / sleep / proxy) — reload history then reconnect.
    scheduleResync(key)
    await new Promise((r) => setTimeout(r, 800))
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
        handleChatEvent(ev, watchGetCur, watchSetCur)
      })
    } catch (e: any) {
      error.value = e.message
    }
    return
  }

  let cur: Msg | null = pushAssistant()
  streaming.value = true
  chatStreamActive = true
  try {
    await chat(key, text, DEFAULT_AGENT_KEY, (ev) => {
      handleChatEvent(ev, () => cur, (m) => { cur = m })
    })
  } catch (e: any) {
    error.value = e.message
  } finally {
    chatStreamActive = false
    if (cur) cur.streaming = false
    // If hub/queue continues the run, keep UI "running" until done arrives on watch.
    const stillBusy = await sessionRunActive(key)
    streaming.value = stillBusy
    if (!stillBusy) {
      for (const m of messages.value) {
        if (m.role === 'assistant' && m.streaming) m.streaming = false
      }
    }
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

async function refreshSession() {
  const key = currentKey.value
  if (!key) return
  chatStreamActive = false
  await selectSession(key)
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
            v-if="!showHistory && currentKey"
            type="button"
            class="w-7 h-7 flex items-center justify-center rounded hover:bg-neutral-100 text-neutral-500"
            title="Refresh"
            @click="refreshSession"
          >
            <LocalSvgIcon name="refresh" :size="14" />
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
          <div
            v-for="s in sessions"
            :key="s.id"
            class="w-full pl-3 pr-1.5 py-1 text-sm hover:bg-neutral-50 border-b border-neutral-100 flex items-center gap-0.5"
            :class="s.id === currentKey ? 'bg-neutral-50 font-medium' : ''"
          >
            <button
              type="button"
              class="min-w-0 flex-1 text-left truncate leading-5"
              @click="pickSession(s.id)"
            >{{ s.title || s.id }}</button>
            <button
              type="button"
              class="shrink-0 w-6 h-6 inline-flex items-center justify-center rounded text-neutral-400 hover:text-red-600 hover:bg-red-50"
              title="删除会话"
              @click="askDeleteSession(s)"
            >
              <LocalSvgIcon name="trash" :size="13" />
            </button>
          </div>
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
                  class="bg-neutral-100 text-neutral-700 text-[15px] leading-relaxed rounded-lg px-3 py-2 whitespace-pre-wrap"
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
                :duration-ms="m.durationMs"
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

    <!-- Run status (above input): live tool / thinking / subagent / harness warn -->
    <div
      v-if="!showHistory && statusBar"
      class="shrink-0 w-full"
      :class="props.expanded ? 'max-w-[960px] mx-auto' : ''"
    >
      <div
        class="flex items-center gap-2 px-3 py-1.5 text-xs"
        :class="[
          props.expanded
            ? 'mx-[8px] border rounded-tl-lg rounded-tr-lg'
            : 'border-t',
          statusBar.warn
            ? 'border-amber-200 bg-amber-50 text-amber-900'
            : statusBar.kind === 'error'
              ? 'border-red-200 bg-red-50 text-red-800'
              : 'border-neutral-200 bg-neutral-50 text-neutral-600',
        ]"
      >
        <span v-if="streaming" class="swiflow-spin shrink-0"></span>
        <span class="truncate flex-1 min-w-0" :title="statusBar.text">{{ statusBar.text }}</span>
        <button
          type="button"
          class="shrink-0 text-neutral-700 hover:underline"
          :class="statusBar.warn ? 'text-amber-800' : ''"
          title="Runtime harness"
          @click="showHarness = true"
        >明细</button>
      </div>
    </div>

    <!-- Input: same column width as messages (border lives on the column, not full page) -->
    <div
      v-if="!showHistory"
      class="shrink-0 w-full"
      :class="props.expanded ? 'max-w-[960px] mx-auto pb-4' : ''"
    >
      <div
        class="relative"
        :class="props.expanded
          ? 'border border-neutral-200 rounded-xl p-3'
          : (statusBar ? 'p-3' : 'border-t border-neutral-200 p-3')"
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
          v-if="streaming"
          type="button"
          class="absolute right-3 bottom-3 w-8 h-8 flex items-center justify-center rounded-md bg-neutral-800 text-white hover:bg-neutral-700"
          title="Abort"
          @click="abortRun"
        >
          <LocalSvgIcon name="stop" :size="22" />
        </button>
        <button
          v-else
          type="button"
          class="absolute right-3 bottom-3 w-8 h-8 flex items-center justify-center rounded-md bg-neutral-800 text-white hover:bg-neutral-700 disabled:opacity-35 disabled:hover:bg-neutral-800"
          :disabled="!input.trim() && !pendingAttachments.length"
          title="Send"
          @click="send"
        >
          <LocalSvgIcon name="send" :size="22" />
        </button>
      </div>
    </div>

    <SubagentDrawer v-if="subagentKey" :session-key="subagentKey" @close="subagentKey = null" />
    <HarnessPanel
      :open="showHarness"
      :focus-session="currentKey || null"
      @close="showHarness = false"
    />

    <!-- Confirm delete (Wails-safe; no window.confirm) -->
    <div
      v-if="deleteTarget"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      @click="cancelDeleteSession"
    >
      <div
        class="bg-white rounded-lg shadow-xl w-full max-w-sm p-4 space-y-3"
        @click.stop
      >
        <div class="text-sm font-medium text-neutral-900">删除会话</div>
        <p class="text-sm text-neutral-600 leading-relaxed">
          确定删除会话「{{ deleteTargetLabel }}」？删除后不可恢复。
        </p>
        <div class="flex justify-end gap-2 pt-1">
          <button
            type="button"
            class="h-8 px-3 text-sm rounded border border-neutral-200 bg-white hover:bg-neutral-50 disabled:opacity-50"
            :disabled="deleting"
            @click="cancelDeleteSession"
          >取消</button>
          <button
            type="button"
            class="h-8 px-3 text-sm rounded bg-red-600 text-white hover:bg-red-700 disabled:opacity-50"
            :disabled="deleting"
            @click="confirmDeleteSession"
          >{{ deleting ? '删除中…' : '删除' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>
