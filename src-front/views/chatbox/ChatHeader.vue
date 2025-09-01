<script setup lang="ts">
import { computed } from 'vue'
import { Tippy } from 'vue-tippy'
import { md } from '@/support/index'
import { useAppStore } from '@/stores/app'
import HistoryTasks from './HistoryTasks.vue'

const app = useAppStore()
const showWelcome = () => {
  app.setShowEpigraph(true)
}
const toggleHistory = () => {
  app.toggleHistory()
}

const hasNewVer = computed(() => {
  if (!app.getRelease) {
    return false
  }
  return app.getRelease['url']
})
const newFeature = computed(() => {
  if (!app.getRelease) {
    return ''
  }
  return app.getRelease['body']
})
const downloadUrl = computed(() => {
  if (!app.getRelease) {
    return ''
  }
  return app.getRelease['url']
})
const showEpigraph = computed(() => {
  if (hasNewVer.value) {
    return false
  }
  return !!app.getDisplay.epigraphText
})
const goDownload = () => {
  if (!app.getRelease) {
    return
  }
  const url = app.getRelease['url'];
  if (!url) {
    return
  }
  const down = document.getElementById('downloadUrl')
  return down && down.click && down.click()
}
</script>
<template>
  <div id="chat-header">
    <button class="btn-icon icon-large btn-menubar" @click="toggleHistory" />
    <button class="btn-icon icon-large btn-newchat" @click="$emit('new-chat')"/>
    <h3 class="main-title">SWIFLOW</h3>
  </div>
  <div id="new-version" v-if="hasNewVer">
    <tippy placement="bottom-end" trigger="mouseenter click" :theme="app.getTheme">
      <a :href="downloadUrl" target="_blank" id="downloadUrl"/>
      <button class="btn-icon btn-restart" @click="goDownload">
        {{ $t('tips.newVersion') }}
      </button>
      <template #content>
        <div id="new-feature" class="rich-text" 
          v-html="md.render(newFeature)"
        />
      </template>
    </tippy>
  </div>
  <div id="show-welcome" v-if="showEpigraph">
    <button class="btn-icon btn-welcome" @click="showWelcome">
      🎉
    </button>
  </div>
  <HistoryTasks v-if="app.getHistory"/>
</template>
<style scoped>
#chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.main-title {
  font-weight: bold;
}
.icon-large{
  margin-top: 3px;
  display: block!important;
}
#new-version{
  right: 15px;
  z-index: 200;
  position: absolute;
  top: calc(8px - var(--nav-height));
}
#new-feature{
  min-width: 15rem;
  margin: 1rem 1rem;
}

/* 欢迎动画按钮样式 */
#show-welcome {
  right: 15px; /* 与new-version位置相同 */
  z-index: 200;
  position: absolute;
  top: calc(8px - var(--nav-height));
}
.btn-welcome {
  font-size: 1.2rem;
  padding: 0rem 0.5rem;
  background-color: transparent;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.3s ease;
}
.btn-welcome:hover {
  background-color: rgba(var(--color-primary-rgb), 0.1);
}
</style>