<script setup lang="ts">
import { PropType, ref, watch } from 'vue'

const props = defineProps({
  active: {
    type: String as PropType<string>,
    default: () => ''
  },
  keyby: {
    type: String as PropType<string>,
    default: () => 'key'
  },
  items: {
    type: Array as PropType<any[]>,
    default: () => []
  }
})
const active = ref<string>(props.active)

const emit = defineEmits(['click'])
const clazz = (item: any) => {
  try {
    const value = item[props.keyby || 'key']
    return active.value == value ? 'active': ''
  } catch (error) {
    console.error('Error in BasicMenu clazz:', error)
    return ''
  }
}
const onClick = (item: any) => {
  try {
    const value = item[props.keyby || 'key']
    active.value = value as string
    emit('click', item)
  } catch (error) {
    console.error('Error in BasicMenu onClick:', error)
  }
}
watch(() => props.active, (val, old) => {
  try {
    if (!active.value || val != old) {
      active.value = props.active
    }
  } catch (error) {
    console.error('Error in BasicMenu watch:', error)
  }
})
</script>

<template>
  <ul class="menu-list">
    <template v-for="item in (props.items || [])">
    <li :class="clazz(item)" @click="onClick(item)">
      <slot name="default" :item="item">
        <span class="item-label">
          {{ item?.label || '' }}
        </span>
      </slot>
    </li>
    </template>
    <template v-if="!props.items || !props.items.length">
      <li class="empty-result">
        {{ $t('common.empty') }}
      </li>
    </template>
  </ul>
</template>

<style scoped>
.menu-list{
  margin: 0;
  padding: 0;
  list-style: none;
}
.menu-list>li{
  margin: 2px 0;
  padding: 0.5em 8px;
  border-radius: 5px;

  display: flex;
  cursor: pointer;
  align-items: center;
  justify-content: space-between;
}
</style>
