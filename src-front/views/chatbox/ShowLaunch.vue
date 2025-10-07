<script setup lang="ts">
import { PropType } from 'vue'
import { Tippy } from 'vue-tippy'
import { useAppStore } from '@/stores/app'

const app = useAppStore()
const props = defineProps({
  launch: {
    type: Array as PropType<string[]>,
    default: () => []
  },
})

const names = {
  open: '在 Finder 中查看',
  code: '在 VS Code 中查看',
  subl: '在 Sublime 中查看',
} as Record<string, string>

</script>

<template>
  <tippy interactive :theme="app.getTheme" arrow
    placement="top-start" trigger="mouseenter click">
    <slot name="default" v-if="$slots['default']"/>
    <icon v-else icon="bin-launch"/>
    <template #content>
      <template v-for="item in props.launch">
      <div class="opt-launch" @click="$emit('select', item)">
        <span>{{ names[item] as string }}</span>
      </div> 
      </template>
    </template>
  </tippy>
</template>

<style scoped>
.opt-launch{
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
.opt-launch:hover{
  box-shadow: inset 1px;
}
.opt-launch:last-of-type{
  border-bottom: 0;
  padding-bottom: 0px;
}
</style>