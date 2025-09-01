<script setup lang="ts">
import { PropType, ref } from 'vue'

const props = defineProps({
  active: {
    type: String as PropType<string>,
    default: () => ''
  },
  items: {
    type: Array as PropType<MenuMeta[]>,
    default: () => []
  }
})
const active = ref<string>(props.active)

const emit = defineEmits(['click'])
const clazz = (item: MenuMeta) => {
  return active.value == item.value ? 'active': ''
}
const onClick = (item: MenuMeta) => {
  active.value = item.value
  emit('click', item)
}
</script>

<template>
  <div> nothing here </div>
  <ul class="menu-list">
    <template v-for="item in props.items">
    <li :class="clazz(item)" @click="onClick(item)">
      <slot name="default" :item="item">
        <span class="item-label">
          {{ item.label }}
        </span>
      </slot>
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
  height: 24px;
  margin: 2px 0;
  padding: 0.5em 1.2em;
  line-height: 24px;
  border-radius: 5px;

  display: flex;
  cursor: pointer;
  align-items: center;
  justify-content: space-between;
}
.menu-list>li {
  font-weight: bold;
}
</style>
