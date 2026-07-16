<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { EditorView, lineNumbers, highlightActiveLineGutter, highlightSpecialChars, drawSelection, dropCursor, rectangularSelection, crosshairCursor, highlightActiveLine, keymap } from '@codemirror/view'
import { EditorState, type Extension } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { syntaxHighlighting, defaultHighlightStyle, bracketMatching, indentOnInput } from '@codemirror/language'
import { codemirrorLanguage } from '../../lib/filePreview'

const props = defineProps<{ content: string; path: string }>()

const host = ref<HTMLElement | null>(null)
let view: EditorView | null = null
let mountSeq = 0

async function buildExtensions(): Promise<Extension[]> {
  const lang = await codemirrorLanguage(props.path)
  return [
    lineNumbers(),
    highlightActiveLineGutter(),
    highlightSpecialChars(),
    history(),
    drawSelection(),
    dropCursor(),
    indentOnInput(),
    bracketMatching(),
    rectangularSelection(),
    crosshairCursor(),
    highlightActiveLine(),
    syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
    keymap.of([...defaultKeymap, ...historyKeymap]),
    EditorView.editable.of(false),
    EditorView.lineWrapping,
    ...(lang ? [lang] : []),
  ]
}

async function mountEditor() {
  if (!host.value) return
  const seq = ++mountSeq
  const extensions = await buildExtensions()
  if (seq !== mountSeq || !host.value) return
  view?.destroy()
  view = new EditorView({
    state: EditorState.create({
      doc: props.content,
      extensions,
    }),
    parent: host.value,
  })
}

onMounted(() => { void mountEditor() })
onBeforeUnmount(() => {
  mountSeq++
  view?.destroy()
})
watch(() => [props.content, props.path], () => { void mountEditor() })
</script>

<template>
  <div ref="host" class="h-full overflow-auto text-sm" />
</template>
