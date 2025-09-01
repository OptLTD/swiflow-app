<script setup lang="ts">
import { computed, ref } from 'vue'
import { Tippy } from 'vue-tippy'
import { usePreferredDark } from '@vueuse/core';

const theTheme = computed(() => {
  const isDark = usePreferredDark().value
  return isDark ? 'dark' : 'light'
})

const tippyRef = ref()
const emit = defineEmits(['set', 'open'])

function handleSet() {
  setTimeout(() => {
    tippyRef.value?.hide?.()
  }, 300)
  emit('set')
}

function handleOpen() {
  setTimeout(() => {
    tippyRef.value?.hide?.()
  }, 300)
  emit('open')
}
</script>

<template>
  <tippy ref="tippyRef" interactive :theme="theTheme" arrow
    placement="top-start" trigger="click">
    <slot name="default" v-if="$slots['default']"/>
    <button class="btn-icon btn-folder"  v-else/>
    <template #content>
      <div class="opt-folder" @click="handleSet">
        <span>{{ $t('common.setHomePath') }}</span>
      </div>
      <div class="opt-folder" @click="handleOpen">
        <span>{{ $t('common.openHomePath') }}</span>
      </div>
    </template>
  </tippy>
</template>

<style scoped>
.opt-folder{
  height: 1rem;
  cursor: pointer;
  margin: 5px 5px;
  padding-bottom: 5px;
  border-width: 0;
  border-style: dotted;
  border-bottom-width: 1px;
  border-color: #d5d5d5;
  min-width: 6rem;
  display: flex;
  align-items: center;
  width: max-content;
}
.opt-folder:hover{
  box-shadow: inset 1px;
}
.opt-folder:last-of-type{
  border-bottom: 0;
  padding-bottom: 0px;
}
</style>