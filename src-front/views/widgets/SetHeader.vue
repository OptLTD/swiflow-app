<script setup lang="ts">
import { PropType } from 'vue'
import { useAppStore } from '@/stores/app'
import { useTaskStore } from '@/stores/task'

defineProps({
  title: {
    type: String as PropType<string>,
    default: () => ''
  },
})

const app = useAppStore()
const task = useTaskStore()
const onGoBack = () => {
  app.setChatBar(true)
  app.setContent(false)
  app.setAction('default')
  document.title = task.getCurrent?.name || 'Swiflow'
}
</script>

<template>
  <div id="set-header">
    <button class="btn-back" @click="onGoBack">
      <svg class="icon" viewBox="0 0 24 24">
        <path d="M15.41 7.41L14 6l-6 6 6 6 1.41-1.41L10.83 12z"/>
      </svg>
    </button>
    <h3 class="main-title">{{ title }}</h3>
    <button @click="app.toggleMenuBar" 
      class="btn-icon icon-large btn-menubar" 
    />
  </div>
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
