<script setup lang="ts">
import { Tippy } from 'vue-tippy'
import { ref, PropType } from 'vue'
import { useAppStore } from '@/stores/app'

const app = useAppStore()
const props = defineProps({
  items: {
    type: Array as PropType<OptMeta[]>,
    default: () => []
  },
  active: {
    type: String as PropType<String>,
    default: () => ''
  },
  enable: {
    type: Boolean as PropType<Boolean>,
    default: () => true
  },
})

const tippyRef = ref()
const emit = defineEmits(['select'])
const handleClick = (item: OptMeta) => {
  if (!props.enable) {
    return;
  }
  setTimeout(() => {
    tippyRef.value?.hide?.()
  }, 300)
  emit('select', item)
}
</script>

<template>
  <tippy ref="tippyRef" :theme="app.getTheme" arrow
    interactive placement="top-start" trigger="click">
    <slot name="default" v-if="$slots['default']"/>
    <button class="btn-icon btn-folder"  v-else/>
    <template #content>
      <template v-for="item in items" :key="item.value">
      <div class="opt-item" @click="handleClick(item)">
        <span :class="{
          active: active==item.value,
          disabled: !props.enable
        }">
          {{ item.label }}
        </span>
      </div>
      </template>
    </template>
  </tippy>
</template>

<style scoped>
.opt-item{
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
.opt-item:hover{
  box-shadow: inset 1px;
}
.opt-item:last-of-type{
  border-bottom: 0;
  padding-bottom: 0px;
}
.opt-item .active{
  font-weight: bold;
}
.opt-item .disabled{
  cursor: not-allowed;
  color: var(--color-tertiary);
}
</style>