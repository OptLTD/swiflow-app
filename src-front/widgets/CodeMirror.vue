<script setup>
import { basicSetup } from "codemirror";
import { EditorState } from '@codemirror/state';
import { EditorView, keymap } from '@codemirror/view';
import { defaultKeymap } from '@codemirror/commands';
import { markdown } from '@codemirror/lang-markdown';
import { ref, onMounted, onBeforeUnmount, watch } from 'vue';
import { githubDark } from '@fsegurai/codemirror-theme-github-dark';
import { githubLight } from '@fsegurai/codemirror-theme-github-light';

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

// 创建编辑器状态
const createEditorState = (content, disabled = false) => {
  const extensions = [
    markdown(),
    isDark() ? githubDark : githubLight,
    EditorView.lineWrapping,
    EditorView.updateListener.of((update) => {
      if (update.docChanged && !disabled) {
        emit('update:modelValue', update.state.doc.toString())
      }
    })
  ]
  
  // 如果禁用，添加只读扩展并移除键盘映射
  if (disabled) {
    extensions.push(EditorState.readOnly.of(true))
    // 不添加键盘映射，这样就不会响应任何键盘输入
  } else {
    extensions.push(keymap.of(defaultKeymap))
  }
  
  return EditorState.create({
    doc: content,
    extensions
  })
}

// 初始化编辑器
onMounted(() => {
  editorView = new EditorView({
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
  if (editorView && newValue !== editorView.state.doc.toString()) {
    editorView.setState(createEditorState(newValue, props.disabled))
  }
})

// 监听 disabled 变化
watch(() => props.disabled, (disabled) => {
  if (editorView) {
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
