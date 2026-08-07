<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api'
import { openExternalURL } from '../lib/openExternal'
import { searchProviderPageURL } from '../lib/searchURL'
import { fromAtPath } from '../lib/workspacePath'
import { useLayoutStore } from '../stores/layout'

const props = defineProps<{
  name: string
  args?: any
  content: string
  isError?: boolean
  progress?: string
  childSession?: string
  childActive?: boolean
  startedAt?: number
  endedAt?: number
  durationMs?: number
}>()
const emit = defineEmits<{ viewChild: [key: string] }>()
const layout = useLayoutStore()
const { t } = useI18n()
const open = ref(false) // collapsed by default

// Live clock so the elapsed time keeps ticking while the tool is running.
const now = ref(Date.now())
let timer: number | undefined
onMounted(() => {
  timer = window.setInterval(() => {
    if (running.value) now.value = Date.now()
  }, 1000)
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
})

function fmtDur(ms: number): string {
  if (ms < 0) ms = 0
  const s = Math.floor(ms / 1000)
  if (s < 60) return `${s}s`
  const m = Math.floor(s / 60)
  const rem = s % 60
  return rem ? `${m}m${rem}s` : `${m}m`
}

const elapsed = computed(() => {
  // Prefer the server-measured execution time (excludes queue wait); fall back
  // to the wall-clock between dispatch and result when unavailable (e.g. history).
  if (props.durationMs != null) return fmtDur(props.durationMs)
  if (!props.startedAt) return ''
  const end = props.endedAt ?? now.value
  return fmtDur(end - props.startedAt)
})

function pick(a: any, k: string): string {
  const v = a?.[k]
  return v == null ? '' : String(v)
}
function trim(s: string, n: number): string {
  return s.length > n ? s.slice(0, n) + '…' : s
}

// Prefer the live child-session key from progress events; fall back to args / spawn result JSON.
const childSession = computed(() => {
  if (props.childSession) return props.childSession
  const fromArgs = pick(props.args, 'child_session').trim()
  if (props.name === 'subagent_status' || props.name === 'subagent_wait') return fromArgs
  if (props.name !== 'subagent_spawn') return ''
  if (!props.content) return fromArgs
  try {
    const j = JSON.parse(props.content)
    return typeof j.child_session === 'string' ? j.child_session : fromArgs
  } catch {
    return fromArgs
  }
})

const canViewChild = computed(() => {
  return !!childSession.value && (
    props.name === 'subagent_spawn' ||
    props.name === 'subagent_status' ||
    props.name === 'subagent_wait'
  )
})

/** Workspace-relative file path for file tools (preview in a new app tab). */
const previewPath = computed(() => {
  const a = props.args || {}
  switch (props.name) {
    case 'fs_read':
    case 'fs_write':
    case 'fs_edit':
    case 'content_extract': {
      const p = fromAtPath(pick(a, 'path'))
      return p || ''
    }
    case 'python_run':
    case 'node_run': {
      const p = fromAtPath(pick(a, 'file'))
      return p || ''
    }
    case 'browser': {
      if (pick(a, 'action') !== 'screenshot') return ''
      const name = pick(a, 'filename').trim()
      if (name) {
        const base = name.replace(/^.*[/\\]/, '')
        return fromAtPath(base.toLowerCase().endsWith('.png') ? `browser/${base}` : `browser/${base}.png`)
      }
      const m = props.content?.match(/screenshot saved to\s+(\S+)/i)
      return m ? fromAtPath(m[1]) : ''
    }
    default:
      return ''
  }
})

/** External http(s) URL for web tools (open in the system default browser). */
const visitURL = computed(() => {
  const a = props.args || {}
  switch (props.name) {
    case 'web_fetch': {
      const u = pick(a, 'url').trim()
      return /^https?:\/\//i.test(u) ? u : ''
    }
    case 'browser': {
      if (pick(a, 'action') !== 'navigate') return ''
      const u = pick(a, 'url').trim()
      return /^https?:\/\//i.test(u) ? u : ''
    }
    default:
      return ''
  }
})

/** web_search uses the configured provider's human-facing results page. */
const canVisit = computed(() => {
  if (visitURL.value) return true
  return props.name === 'web_search' && !!pick(props.args, 'query').trim()
})

async function resolveVisitURL(): Promise<string> {
  if (visitURL.value) return visitURL.value
  if (props.name !== 'web_search') return ''
  const query = pick(props.args, 'query').trim()
  if (!query) return ''
  try {
    const r = await api.getSearchSettings()
    return searchProviderPageURL(r.provider || 'bing', r.base_url || '', query)
  } catch {
    return searchProviderPageURL('bing', '', query)
  }
}

// Human-readable intent derived from the tool name + arguments, so the header
// reads like an action rather than a raw function call.
const intent = computed(() => {
  const a = props.args || {}
  switch (props.name) {
    case 'fs_read':
      return t('toolCall.fsRead', { path: trim(pick(a, 'path'), 60) })
    case 'fs_write':
      return t('toolCall.fsWrite', { path: trim(pick(a, 'path'), 60) })
    case 'fs_list': {
      const p = pick(a, 'path') || '.'
      return p === '.'
        ? t('toolCall.fsListRoot')
        : t('toolCall.fsList', { path: trim(p, 40) })
    }
    case 'fs_edit':
      return t('toolCall.fsEdit', { path: trim(pick(a, 'path'), 60) })
    case 'content_extract':
      return t('toolCall.contentExtract', { path: trim(pick(a, 'path'), 60) })
    case 'web_fetch':
      return t('toolCall.webFetch', { url: trim(pick(a, 'url'), 60) })
    case 'web_search':
      return t('toolCall.webSearch', { query: trim(pick(a, 'query'), 60) })
    case 'browser': {
      const act = pick(a, 'action')
      if (act === 'navigate') return t('toolCall.browserNavigate', { url: trim(pick(a, 'url'), 50) })
      if (act === 'screenshot') return t('toolCall.browserScreenshot')
      if (act === 'click') {
        const tip = pick(a, 'text') || pick(a, 'selector')
        return t('toolCall.browserClick', { target: trim(tip, 40) })
      }
      if (act === 'type') return t('toolCall.browserType', { selector: trim(pick(a, 'selector'), 40) })
      return t('toolCall.browser', { action: act })
    }
    case 'exec':
    case 'exec_run':
    case 'cmd_run':
      return t('toolCall.exec', { command: trim(pick(a, 'command'), 60) })
    case 'python_run':
      if (pick(a, 'file')) return t('toolCall.pythonFile', { file: trim(pick(a, 'file'), 50) })
      return t('toolCall.pythonCode')
    case 'node_run':
      if (pick(a, 'file')) return t('toolCall.nodeFile', { file: trim(pick(a, 'file'), 50) })
      return t('toolCall.nodeCode')
    case 'skill_use':
      return t('toolCall.skillUse', { slug: trim(pick(a, 'slug'), 40) })
    case 'skill_search':
      return t('toolCall.skillSearch', { query: trim(pick(a, 'query'), 40) })
    case 'skill_manage': {
      const act = pick(a, 'action')
      const slug = trim(pick(a, 'slug'), 40)
      if (act === 'create') return t('toolCall.skillCreate', { slug })
      if (act === 'patch') return t('toolCall.skillPatch', { slug })
      return t('toolCall.skillManage', { slug })
    }
    case 'skill_draft':
      return t('toolCall.skillDraft', { slug: trim(pick(a, 'slug'), 40) })
    case 'subagent_spawn':
      return t('toolCall.subagentSpawn', { goal: trim(pick(a, 'goal'), 50) })
    case 'subagent_status':
      return t('toolCall.subagentStatus', { session: trim(pick(a, 'child_session'), 40) })
    case 'subagent_wait':
      return t('toolCall.subagentWait', { session: trim(pick(a, 'child_session'), 40) })
    case 'experience_write':
      return t('toolCall.experienceWrite', { summary: trim(pick(a, 'summary'), 50) })
    case 'experience_list':
      return t('toolCall.experienceList')
    case 'experience_use':
      return t('toolCall.experienceUse')
    case 'todo_write':
      return t('toolCall.todoWrite')
    case 'todo_read':
      return t('toolCall.todoRead')
    case 'schedule_run':
      return t('toolCall.scheduleRun', {
        seconds: pick(a, 'delay_seconds') || '?',
        message: trim(pick(a, 'message'), 50),
      })
    case 'schedule_create':
      return t('toolCall.scheduleCreate', {
        name: trim(pick(a, 'name'), 40),
        schedule: trim(pick(a, 'schedule'), 30),
      })
    case 'light_app_create':
      return t('toolCall.lightAppCreate', { name: trim(pick(a, 'name'), 40) })
    case 'light_app_list':
      return t('toolCall.lightAppList')
    case 'light_app_launch':
      return t('toolCall.lightAppLaunch', { id: trim(pick(a, 'id'), 40) })
    case 'light_app_write':
      return t('toolCall.lightAppWrite', { path: trim(pick(a, 'path'), 60) })
    case 'light_app_read':
      return t('toolCall.lightAppRead', { path: trim(pick(a, 'path'), 60) })
    case 'light_app_ls':
      return t('toolCall.lightAppLs', { path: trim(pick(a, 'path'), 40) })
    case 'light_app_open':
      return t('toolCall.lightAppOpen', { url: trim(pick(a, 'url'), 50) })
    default:
      return props.name
  }
})

const body = computed(() => {
  let out = ''
  if (props.args && Object.keys(props.args).length) {
    out += 'args:\n' + JSON.stringify(props.args, null, 2) + '\n\n'
  }
  out += props.content
  return out
})

// Running while we have no final result yet (tool_result not received).
const running = computed(() => {
  if (props.isError) return false
  if (props.endedAt != null || props.durationMs != null) return false
  return !props.content
})

const showChildProgress = computed(() => {
  return props.name === 'subagent_spawn' && !!childSession.value &&
    (props.childActive || running.value || !!props.progress)
})

/** URL returned by light_app_launch — shown as an "打开" action button. */
const launchURL = computed(() => {
  if (props.name !== 'light_app_launch' || !props.content || props.isError) return ''
  try {
    const j = JSON.parse(props.content)
    return typeof j.url === 'string' ? j.url : ''
  } catch {
    return ''
  }
})

async function openLaunch(e: Event) {
  e.stopPropagation()
  if (!launchURL.value) return
  const { openLightApp } = await import('../lib/openLightApp')
  let title = 'Light App'
  try {
    const j = JSON.parse(props.content || '')
    if (typeof j.name === 'string' && j.name) title = j.name
  } catch { /* ignore */ }
  await openLightApp(launchURL.value, title)
}

function viewChild(e: Event) {
  e.stopPropagation()
  if (childSession.value) emit('viewChild', childSession.value)
}

function openPreview(e: Event) {
  e.stopPropagation()
  if (previewPath.value) layout.openFile(previewPath.value)
}

async function openVisit(e: Event) {
  e.stopPropagation()
  const u = await resolveVisitURL()
  if (!u) return
  await openExternalURL(u)
}
</script>

<template>
  <div class="border border-neutral-200 rounded text-xs">
    <button
      class="w-full px-2 py-1 bg-neutral-100 flex justify-between items-center hover:bg-neutral-200 gap-2"
      @click="open = !open"
    >
      <span class="flex items-center gap-1 truncate">
        <span class="text-neutral-400">{{ open ? '▼' : '▶' }}</span>
        <span class="truncate">{{ intent }}</span>
        <span class="text-neutral-400 font-mono shrink-0">{{ name }}</span>
      </span>
      <span class="shrink-0 flex items-center gap-1.5 leading-none">
        <span v-if="elapsed" class="text-neutral-400 tabular-nums">{{ elapsed }}</span>
        <span
          v-if="previewPath"
          role="button"
          tabindex="0"
          class="text-neutral-700 hover:underline cursor-pointer"
          @click="openPreview"
          @keydown.enter.prevent="openPreview"
        >{{ t('toolCall.view') }}</span>
        <span
          v-if="canVisit"
          role="button"
          tabindex="0"
          class="text-neutral-700 hover:underline cursor-pointer"
          @click="openVisit"
          @keydown.enter.prevent="openVisit"
        >{{ t('toolCall.visit') }}</span>
        <span
          v-if="launchURL"
          role="button"
          tabindex="0"
          class="text-neutral-700 hover:underline cursor-pointer"
          @click="openLaunch"
          @keydown.enter.prevent="openLaunch"
        >{{ t('toolCall.open') }}</span>
        <span
          v-if="canViewChild"
          role="button"
          tabindex="0"
          class="text-neutral-700 hover:underline cursor-pointer"
          @click="viewChild"
          @keydown.enter.prevent="viewChild"
        >{{ t('toolCall.view') }}</span>
        <span
          class="inline-flex items-center gap-1"
          :class="isError ? 'text-red-600' : running ? 'text-neutral-500' : 'text-green-600'"
        >
          <template v-if="running"><span class="swiflow-spin"></span><span>{{ t('toolCall.running') }}</span></template>
          <template v-else>{{ isError ? t('toolCall.failed') : t('toolCall.success') }}</template>
        </span>
      </span>
    </button>
    <div
      v-if="showChildProgress"
      class="px-2 py-1 bg-neutral-50 border-t border-neutral-100 flex items-center gap-1.5"
    >
      <span class="swiflow-spin shrink-0"></span>
      <span class="truncate text-neutral-500 flex-1">{{ progress || t('toolCall.subagentRunning') }}</span>
    </div>
    <pre v-show="open" class="p-2 whitespace-pre-wrap max-h-64 overflow-y-auto bg-neutral-50 border-t border-neutral-100">{{ body }}</pre>
  </div>
</template>
