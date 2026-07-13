<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { api } from '../api'
import hljs from 'highlight.js/lib/core'
import jsonLang from 'highlight.js/lib/languages/json'
import python from 'highlight.js/lib/languages/python'
import bash from 'highlight.js/lib/languages/bash'
import javascript from 'highlight.js/lib/languages/javascript'
import typescript from 'highlight.js/lib/languages/typescript'
import go from 'highlight.js/lib/languages/go'
import markdown from 'highlight.js/lib/languages/markdown'
import yaml from 'highlight.js/lib/languages/yaml'
import xml from 'highlight.js/lib/languages/xml'
import css from 'highlight.js/lib/languages/css'
import 'highlight.js/styles/github.min.css'

hljs.registerLanguage('json', jsonLang)
hljs.registerLanguage('python', python)
hljs.registerLanguage('bash', bash)
hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('typescript', typescript)
hljs.registerLanguage('go', go)
hljs.registerLanguage('markdown', markdown)
hljs.registerLanguage('yaml', yaml)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('css', css)

const props = defineProps<{ path: string }>()
const content = ref('')
const loading = ref(true)
const error = ref('')

const extLangMap: Record<string, string> = {
  go: 'go', ts: 'typescript', tsx: 'typescript', js: 'javascript', jsx: 'javascript',
  py: 'python', json: 'json', yml: 'yaml', yaml: 'yaml', xml: 'xml', html: 'xml',
  css: 'css', md: 'markdown', sh: 'bash', bash: 'bash', zsh: 'bash',
}

function langFromPath(path: string): string {
  const ext = path.split('.').pop()?.toLowerCase() || ''
  return extLangMap[ext] || ''
}

async function loadFile() {
  loading.value = true
  error.value = ''
  try {
    const r = await api.readWorkspaceFile(props.path)
    content.value = r.content
    if (r.truncated) {
      content.value += '\n\n...[truncated]'
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

onMounted(loadFile)
watch(() => props.path, loadFile)

function highlighted(): string {
  if (!content.value) return ''
  const lang = langFromPath(props.path)
  if (lang) {
    try {
      return hljs.highlight(content.value, { language: lang }).value
    } catch {}
  }
  return hljs.highlightAuto(content.value).value
}

function lines(): string[] {
  return content.value.split('\n')
}
</script>

<template>
  <div class="h-full overflow-y-auto bg-neutral-50">
    <div v-if="loading" class="p-6 text-neutral-400">Loading…</div>
    <div v-else-if="error" class="p-6 text-red-600">{{ error }}</div>
    <div v-else class="font-mono text-xs leading-relaxed">
      <table class="w-full border-collapse">
        <tbody>
          <tr v-for="(line, i) in lines()" :key="i" class="hover:bg-blue-50">
            <td class="text-right pr-4 pl-4 text-neutral-400 select-none w-[1%] whitespace-nowrap">{{ i + 1 }}</td>
            <td class="pr-4 whitespace-pre" v-html="hljs.highlightAuto(line).value"></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
