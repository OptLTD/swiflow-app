<script setup lang="ts">
import { ref, watch, computed } from 'vue'
const props = defineProps({
  content: String,
  running: Boolean,
  placeholder: String,
})
const emit = defineEmits([
  'update:content',
  'send', 'stop',
])
const content = ref(props.content)
watch(content, v => emit('update:content', v))
watch(() => props.content, v => content.value = v)
const canStop = computed(() => {
  return props.running && !content.value?.trim()
})

// 迁移输入法相关逻辑
const isIMEActive = ref(false)
let justEndedComposition = false
const handleCompositionStart = () => {
  isIMEActive.value = true
}
const handleCompositionEnd = () => {
  isIMEActive.value = false
  justEndedComposition = true
  setTimeout(() => {
    justEndedComposition = false
  }, 10) // 10ms 足够
}

const handleSend = () => {
  emit('send', content.value)
}

// 处理回车逻辑，回车时 emit send
const handleEnter = (e: KeyboardEvent) => {
  if (e.key === 'Enter' && e.shiftKey) {
    return
  }
  // 输入法激活状态的选词
  if (e.key === 'Enter' && (isIMEActive.value || justEndedComposition)) {
    return
  }
  if (e.key === 'Enter' && content.value) {
    e.preventDefault();
    emit('send', content.value)
  }
}
</script>
<template>
  <div class="input-inner">
    <div class="header-action">
      <slot name="header"/>
    </div>
    <div class="btn-tool-group">
      <slot name="tools"/>
    </div>
    <div class="btn-main-group">
     
      <Icon v-if="canStop" @click="$emit('stop')" 
        icon="record-stop" class="input-btn" size="large"
      />
      <Icon v-if="!canStop" @click="handleSend"  
        icon="record-send" class="input-btn" size="large"
      />
    </div>
    <textarea v-model="content" class="input-control"
      @keydown="handleEnter" :placeholder="placeholder" 
      @compositionstart="handleCompositionStart"
      @compositionend="handleCompositionEnd"
    />
  </div>
</template>

<style>
@import "./index.css";
</style> 