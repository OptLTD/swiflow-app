<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
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
  startedAt?: number
  endedAt?: number
  durationMs?: number
}>()
const emit = defineEmits<{ viewChild: [key: string] }>()
const layout = useLayoutStore()
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

// Prefer the live child-session key from progress events; fall back to the final
// delegate_task result JSON once the call completes.
const childSession = computed(() => {
  if (props.name !== 'delegate_task') return ''
  if (props.childSession) return props.childSession
  if (!props.content) return ''
  try {
    const j = JSON.parse(props.content)
    return typeof j.child_session === 'string' ? j.child_session : ''
  } catch {
    return ''
  }
})

/** Workspace-relative file path for file tools (preview in a new app tab). */
const previewPath = computed(() => {
  const a = props.args || {}
  switch (props.name) {
    case 'fs_read':
    case 'fs_write':
    case 'fs_edit':
    case 'document_extract': {
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
      return '读取文件 ' + trim(pick(a, 'path'), 60)
    case 'fs_write':
      return '写入文件 ' + trim(pick(a, 'path'), 60)
    case 'fs_list': {
      const p = pick(a, 'path') || '.'
      return p === '.' ? '列出目录内容' : '列出目录 ' + trim(p, 40) + ' 内容'
    }
    case 'fs_edit':
      return '编辑文件 ' + trim(pick(a, 'path'), 60)
    case 'document_extract':
      return '抽取文档 ' + trim(pick(a, 'path'), 60)
    case 'web_fetch':
      return '抓取网页 ' + trim(pick(a, 'url'), 60)
    case 'web_search':
      return '搜索关键字 ' + trim(pick(a, 'query'), 60)
    case 'browser': {
      const act = pick(a, 'action')
      if (act === 'navigate') return '浏览器打开 ' + trim(pick(a, 'url'), 50)
      if (act === 'screenshot') return '浏览器截图'
      if (act === 'click') {
        const tip = pick(a, 'text') || pick(a, 'selector')
        return '浏览器点击 ' + trim(tip, 40)
      }
      if (act === 'type') return '浏览器输入 ' + trim(pick(a, 'selector'), 40)
      return '浏览器 ' + act
    }
    case 'exec':
    case 'exec_run':
    case 'cmd_run':
      return '执行命令: ' + trim(pick(a, 'command'), 60)
    case 'python_run':
      if (pick(a, 'file')) return '运行 Python 脚本 ' + trim(pick(a, 'file'), 50)
      return '运行 Python 代码'
    case 'node_run':
      if (pick(a, 'file')) return '运行 Node 脚本 ' + trim(pick(a, 'file'), 50)
      return '运行 Node 代码'
    case 'skill_use':
      return '使用技能 ' + trim(pick(a, 'slug'), 40)
    case 'skill_search':
      return '搜索技能 ' + trim(pick(a, 'query'), 40)
    case 'skill_manage': {
      const act = pick(a, 'action')
      const slug = trim(pick(a, 'slug'), 40)
      if (act === 'create') return '创建技能 ' + slug
      if (act === 'patch') return '更新技能 ' + slug
      return '管理技能 ' + slug
    }
    case 'skill_draft':
      return '技能草案 ' + trim(pick(a, 'slug'), 40)
    case 'delegate_task':
      return '委派子任务 ' + trim(pick(a, 'goal'), 50)
    case 'todo_write':
      return '更新任务清单'
    case 'todo_read':
      return '读取任务清单'
    case 'schedule_run':
      return `将在 ${pick(a, 'delay_seconds') || '?'} 秒后执行任务: ` + trim(pick(a, 'message'), 50)
    case 'schedule_create':
      return '创建定时任务 ' + trim(pick(a, 'name'), 40) + ' (' + trim(pick(a, 'schedule'), 30) + ')'
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
        >查看</span>
        <span
          v-if="canVisit"
          role="button"
          tabindex="0"
          class="text-neutral-700 hover:underline cursor-pointer"
          @click="openVisit"
          @keydown.enter.prevent="openVisit"
        >访问</span>
        <span
          class="inline-flex items-center gap-1"
          :class="isError ? 'text-red-600' : running ? 'text-neutral-500' : 'text-green-600'"
        >
          <template v-if="running"><span class="swiflow-spin"></span><span>运行中</span></template>
          <template v-else>{{ isError ? '失败' : '成功' }}</template>
        </span>
      </span>
    </button>
    <div
      v-if="name === 'delegate_task' && running && (progress || childSession)"
      class="px-2 py-1 bg-neutral-50 border-t border-neutral-100 flex items-center gap-1.5"
    >
      <span class="swiflow-spin shrink-0"></span>
      <span class="truncate text-neutral-500 flex-1">{{ progress || '子任务运行中…' }}</span>
      <button
        v-if="childSession"
        type="button"
        class="shrink-0 text-neutral-700 hover:underline"
        @click="viewChild"
      >查看</button>
    </div>
    <div
      v-if="name === 'delegate_task' && !running && !isError && childSession"
      class="px-2 py-1 bg-neutral-50 border-t border-neutral-100"
    >
      <button
        type="button"
        class="text-neutral-700 hover:underline"
        @click="viewChild"
      >查看子任务过程</button>
    </div>
    <pre v-show="open" class="p-2 whitespace-pre-wrap max-h-64 overflow-y-auto bg-neutral-50 border-t border-neutral-100">{{ body }}</pre>
  </div>
</template>
