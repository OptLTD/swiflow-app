<script setup lang="ts">
import { ref, computed } from 'vue'

const props = defineProps<{ name: string; args?: any; content: string; isError?: boolean }>()
const open = ref(false) // collapsed by default

function pick(a: any, k: string): string {
  const v = a?.[k]
  return v == null ? '' : String(v)
}
function trim(s: string, n: number): string {
  return s.length > n ? s.slice(0, n) + '…' : s
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

// Running = call emitted but no result yet (empty content, not an error).
const running = computed(() => !props.content && !props.isError)
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
      <span class="shrink-0 flex items-center gap-1" :class="isError ? 'text-red-600' : 'text-green-600'">
        <template v-if="running"><span class="swiflow-spin"></span><span class="text-neutral-500">running</span></template>
        <template v-else>{{ isError ? 'error' : 'ok' }}</template>
      </span>
    </button>
    <pre v-show="open" class="p-2 whitespace-pre-wrap max-h-64 overflow-y-auto bg-neutral-50">{{ body }}</pre>
  </div>
</template>
