<script setup>
import { ref, onMounted, onBeforeUnmount, watch } from 'vue';

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  disabled: {
    type: Boolean,
    default: false
  }
})

const isDark = () => {
  const support = window.matchMedia('(prefers-color-scheme)').media
  const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  return support != 'not all' && isDark
}

const emit = defineEmits(['update:modelValue'])

const editorContainer = ref(null)
let editorView = null
let cmModulesReady = false
let cm, State, View, keymapMod, commandsMod, mdLang, themeDark, themeLight

// 创建编辑器状态
const createEditorState = (content, disabled = false) => {
  const extensions = []
  if (mdLang) extensions.push(mdLang())
  if (themeDark && themeLight) extensions.push(isDark() ? themeDark : themeLight)
  if (View) {
    extensions.push(View.lineWrapping)
    extensions.push(View.updateListener.of((update) => {
      if (update.docChanged && !disabled) {
        emit('update:modelValue', update.state.doc.toString())
      }
    }))
  }
  if (disabled && State) {
    extensions.push(State.readOnly.of(true))
  } else if (keymapMod && commandsMod) {
    extensions.push(keymapMod.of(commandsMod))
  }
  return State.create({
    doc: content,
    extensions
  })
}

// 初始化编辑器
onMounted(async () => {
  const [codemirror, state, view, commands, md, dark, light] = await Promise.all([
    import('codemirror'),
    import('@codemirror/state'),
    import('@codemirror/view'),
    import('@codemirror/commands'),
    import('@codemirror/lang-markdown'),
    import('@fsegurai/codemirror-theme-github-dark'),
    import('@fsegurai/codemirror-theme-github-light')
  ])
  cm = codemirror
  State = state.EditorState
  View = view.EditorView
  keymapMod = view.keymap
  commandsMod = commands.defaultKeymap
  mdLang = md.markdown
  themeDark = dark.githubDark
  themeLight = light.githubLight
  cmModulesReady = true
  editorView = new View({
    state: createEditorState(props.modelValue, props.disabled),
    parent: editorContainer.value
  })
})

// 清理编辑器
onBeforeUnmount(() => {
  if (editorView) {
    editorView.destroy()
  }
})

// 监听 modelValue 变化
watch(() => props.modelValue, (newValue) => {
  if (editorView && cmModulesReady && newValue !== editorView.state.doc.toString()) {
    editorView.setState(createEditorState(newValue, props.disabled))
  }
})

// 监听 disabled 变化
watch(() => props.disabled, (disabled) => {
  if (editorView && cmModulesReady) {
    editorView.setState(createEditorState(editorView.state.doc.toString(), disabled))
  }
})
</script>

<template>
  <div class="codemirror-container" :class="{ 'disabled': disabled }">
    <div ref="editorContainer"></div>
    <slot v-if="disabled" name="disabled-overlay"/>
  </div>
</template>

<style>
/* 确保编辑器容器有合适的高度 */
.cm-editor {
  height: 100%;
  outline: 1px dotted #dadada;
}

.codemirror-container.disabled {
  width: 100%;
  height: 100%;
  opacity: 0.6;
  position: relative;
  cursor: not-allowed;
  min-height: inherit;
}

.codemirror-container > div:first-child {
  height: 100%;
  width: 100%;
  min-height: inherit;
}

</style>
