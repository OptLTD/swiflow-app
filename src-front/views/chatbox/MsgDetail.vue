<script setup lang="ts">
import { md } from '@/support';
import { computed, PropType } from 'vue'
import { useAppStore } from '@/stores/app'
import { useViewStore } from '@/stores/view'
import { getActDesc } from '@/logics/chat'
import { showDisplayAct } from '@/logics/chat'

const emit = defineEmits(["check", 'display'])
const app = useAppStore()
const view = useViewStore()
const props = defineProps({
  loading: {
    type: Boolean as PropType<boolean>,
    default: () => false
  },
  isLast: {
    type: Boolean as PropType<boolean>,
    default: () => false
  },
  detail: {
    type: Object as PropType<ActionMsg | null>,
    default: null
  },
})

const thinking = computed(() => {
  const { thinking } = props.detail || {}
  return thinking || ''
})

const actions = computed(() => {
  const { actions } = props.detail || {}
  if (!actions || !actions.length) {
    return []
  }
  return actions
})

const isInput = computed(() => {
  const { actions } = props.detail || {}
  if (!actions || !actions.length) {
    return ''
  }
  const { type } = actions[0]
  return type == 'user-input'
})

const render = (item: MsgAct) => {
  const act = (item as DefaultAction)
  if (!act.content) {
    return ''
  }
  return md.render(act.content)
}

const getUpload = (item: MsgAct) => {
  const act = (item as UserInput)
  if (!act.uploads) {
    return ''
  }
  let text = act.uploads.join(',').trim()
  if (text.startsWith('[[') && text.endsWith(')]')) {
    text = text.replace('[[', '[').replace(')]', ')')
  }
  return md.render(text)
}

const handleFileClick = (e: MouseEvent) => {
  const ele = e.target as HTMLLinkElement
  const href = ele.getAttribute('href')
  if (href) {
    // Extract file path from markdown link format [filename](path)
    // Decode the file path to prevent double encoding in Browser.vue
    const filePath = decodeURIComponent(href)
    const detail = { path: filePath }
    app.setContent(true)
    app.setAction('browser')
    view.setChange(detail)
  }
}

const handleOptionCheck = (act: MakeAsk, m: number) => {
  if (act.options[m]) {
    act.checked = m
  } else {
    return
  }
  if (act.options[m].includes('other')) {
    return
  }
  if (act.options[m].includes('其他')) {
    return
  }
  emit('check', act.options[m], act)
}

const handleDisplayWithArgs = (act: MsgAct) => {
  // @ts-ignore
  act.more = true
  emit('display', act)
}

const handleDisplayDirect = (act: MsgAct) => {
  // @ts-ignore
  act.more = false
  emit('display', act)
}

const toolsName = [
  'execute-command',
  'path-list-files',
  'file-put-content',
  'file-get-content',
  'file-replace-text',
  'use-mcp-tool',
  'use-self-tool',
  'start-async-cmd',
  'query-async-cmd',
  'abort-async-cmd',
]
const hasMoreArgs = [
  'use-mcp-tool',
  'use-self-tool',
]
</script>

<template>
  <div class="msg-wrap" :class="{
    'pull-right': isInput, loading
  }">
    <slot name="header" v-if="!isInput" />
    <div class="msg-act" v-if="loading && thinking">
      <div class="rich-text" v-html="md.render(thinking)" />
    </div>
    <template v-for="(act, j) in actions" :key="j">
      <div class="msg-act" v-if="act.type == 'make-ask'">
        <dl class="request">
          <dt>{{ (act as MakeAsk).question }}</dt>
          <dd v-for="(option, m) in (act as MakeAsk).options">
            <label>
              <input type="checkbox" :value="option" @click="handleOptionCheck(act as MakeAsk, m)"
                :disabled="isLast == false" :checked="m == act.checked" />
              {{ option }}
            </label>
          </dd>
        </dl>
      </div>
      <div class="msg-act" v-else-if="act.type == 'user-input'">
        <div class="user-input" v-html="render(act)" />
        <template v-if="(act as UserInput).uploads?.length">
          <div class="user-files" v-html="getUpload(act)" @click.prevent="handleFileClick" />
        </template>
      </div>
      <div class="msg-act" v-else-if="act.type == 'tool-result'">
        <div class="rich-text" v-html="render(act)" />
      </div>
      <div class="msg-act" v-else-if="act.type == 'complete'">
        <template v-if="!showDisplayAct(act)">
          <div class="rich-text" v-html="render(act)" />
        </template>
        <div v-else class="act-card" @click="$emit('display', act)">
          {{ getActDesc(act) }}
        </div>
      </div>
      <div class="msg-act" v-else-if="toolsName.includes(act.type)">
        <div class="act-card" @click.stop="handleDisplayDirect(act)">
          {{ getActDesc(act) }}
          <img v-if="$props.loading" class="loading-image" src="/assets/loading.svg" />
          <template v-if="!$props.loading && hasMoreArgs.includes(act.type)">
            <button class="btn-more" @click.stop="handleDisplayWithArgs(act)">
              <img class="icon" src="/assets/weather.svg" />
            </button>
          </template>
        </div>
      </div>
      <div class="msg-act" v-else-if="!act.hide">
        <div class="rich-text" v-html="render(act)" />
      </div>
    </template>
    <slot name="footer" v-if="!isInput" />
    <!-- loading state -->
    <img v-if="$props.loading" class="loading-image" src="/assets/loading.svg" />
  </div>
</template>

<style scoped>
.msg-act {
  /* overflow: auto; */
  margin: 12px 12px;
  word-break: break-word;
}

.thinking,
.user-input,
.present,
.request {
  font-size: 1rem;
}

.user-input {
  display: block;
  padding: 8px 12px;
  border-radius: 5px;
  background: #f1f1f1;
}

.user-files {
  margin-top: -5px;
  max-height: 120px;
  overflow-y: auto;
  padding: 5px 0;
}

.user-files :deep(p) {
  margin: 2px 0;
  line-height: 1.3;
}

.user-files :deep(a) {
  display: inline-block;
  text-decoration: none;
  cursor: pointer;
  padding: 1px 1px;
  border-radius: 3px;
  max-width: 8rem;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  vertical-align: middle;
  transition: background-color 0.2s ease;
}

.user-files :deep(a:hover) {
  text-decoration: underline;
  background-color: rgba(0, 0, 0, 0.05);
}

.user-input:deep(:last-child),
.user-input:deep(:first-child) {
  margin-block-end: 0em;
  margin-block-start: 0em;
}

.request {
  padding-left: 0;
}

.request>dt {
  font-weight: 500;
  margin-top: -1em;
  margin-bottom: 0.5em;
}

.request>dd {
  margin-left: 1rem;
}

.loading .act-card {
  cursor: wait;
}

.loading-image {
  height: 20px;
  margin-top: 8px;
  margin-left: 12px;
  align-self: baseline;
  /* margin-top: 1rem; */
}

.act-card .loading-image {
  top: 0px;
  right: 10px;
  position: absolute;
}

.btn-more {
  top: 2px;
  right: 2px;
  outline: none;
  cursor: pointer;
  font-size: 14px;
  display: flex;
  position: absolute;
  border: 0px solid;
  border-radius: 6px;
  padding: 0.6em 0.6em;
  align-items: center;
  transition: all 0.2s ease;
  background-color: var(--bg-light);
}

.btn-more:hover {
  background-color: var(--bg-menu);
}

.btn-more>.icon {
  width: 16px;
  height: 16px;
  fill: currentColor;
}
</style>
