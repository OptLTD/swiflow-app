<script setup lang="ts">
import { computed } from 'vue'
import { getActHtml } from '@/logics/chat'
import { getDisplayActDesc } from '@/logics/chat'
import { VueFinalModal } from 'vue-final-modal'

const emit = defineEmits(['submit', 'cancel'])
const props = defineProps({
  tool: {
    type: Object,
    default: {}
  },
})
const html = computed(() => {
  return getActHtml(props.tool as MsgAct)
})

const title = computed(() => {
  return getDisplayActDesc(props.tool as MsgAct)
})
</script>

<template>
  <VueFinalModal
    modalId="theUseToolModal"
    class="swiflow-modal-wrapper"
    content-class="modal-content"
    overlay-transition="vfm-fade"
    content-transition="vfm-fade"
  >
    <h2 class="modal-title">{{ title }}</h2>
    <div class="context-html" v-html="html"/>
    <div class="actions">
      <button class="btn-cancel" @click="emit('cancel')">
        CLOSE
      </button>
    </div>
  </VueFinalModal>
</template>

<style scoped>
@import url('@/styles/modal.css');
.modal-title{
  text-align: left;
}
.context-tabs{
  gap: 12px;
  padding: 0;
  display: flex;
  list-style: none;
  margin: 0.5rem;
}
.context-tabs>li{
  font-size: 1.1rem;
  cursor: pointer;
  text-decoration: underline;
}
.context-tabs>li.active{
  cursor: initial;
  font-weight: bold;
  text-decoration: none;
}
.context-html{
  margin: 0 0.5rem;
  overflow: scroll;
}
.context-html :deep(h2){
  margin: 1rem 0rem;
  font-size: 1.35rem;
}
:global(.context-modal){
  height: 75% !important;
  width: 55% !important;
}
</style>