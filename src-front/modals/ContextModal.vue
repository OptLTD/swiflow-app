<script setup lang="ts">
import { computed } from 'vue'
import { md } from '@/support/index';
import { VueFinalModal } from 'vue-final-modal'


const emit = defineEmits(['submit', 'cancel'])
const props = defineProps({
  context: {
    type: Object,
    default: {}
  },
})

const html = computed(() => {
  return md.render(props.context.context)
})
</script>

<template>
  <VueFinalModal
    modalId="theContextModal"
    class="swiflow-modal-wrapper"
    content-class="modal-content"
    overlay-transition="vfm-fade"
    content-transition="vfm-fade"
  >
    <h2> {{ props.context?.subject }} </h2>
    <div v-html="html" class="rich-text"/>
    <div class="actions">
      <button class="btn-cancel" @click="emit('cancel')">
        CLOSE
      </button>
    </div>
  </VueFinalModal>
</template>

<style scoped>
@import url('@/styles/modal.css');
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
.rich-text {
  padding: 0 0.5rem;
  overflow-y: auto;
}
:global(.context-modal){
  height: 75% !important;
  width: 55% !important;
}
</style>