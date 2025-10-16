<script setup lang="ts">
import { computed } from 'vue'
import { Tippy } from 'vue-tippy'
import { md } from '@/support/index'
import { useAppStore } from '@/stores/app'
import HistoryTasks from './HistoryTasks.vue'

const app = useAppStore()
defineEmits(['new-chat'])
const showWelcome = () => {
  app.setShowEpigraph(true)
}
const toggleHistory = () => {
  app.toggleHistory()
}

const hasNewVer = computed(() => {
  if (!app.getRelease || app.getInDocker) {
    return false
  }
  return app.getRelease['url']
})
const newFeature = computed(() => {
  if (!app.getRelease || app.getInDocker) {
    return ''
  }
  return app.getRelease['body']
})
const downloadUrl = computed(() => {
  if (!app.getRelease || app.getInDocker) {
    return ''
  }
  return app.getRelease['url']
})
const showEpigraph = computed(() => {
  if (hasNewVer.value || app.getInDocker) {
    return false
  }
  return !!app.getDisplay.epigraphText
})
const goDownload = () => {
  if (!app.getRelease || app.getInDocker) {
    return
  }
  const url = app.getRelease['url'];
  if (!url) {
    return
  }
  const down = document.getElementById('downloadUrl')
  return down && down.click() || window.open(url, '_blank')
}
</script>
<template>
  <div id="chat-header">
     <Icon icon="icon-list" 
      @click="() => toggleHistory()" 
    />
    <Icon icon="icon-plus" 
      @click="() => $emit('new-chat')" 
    />
    <h3 class="main-title">SWIFLOW</h3>
  </div>
  <div id="new-version" v-if="hasNewVer">
    <tippy  trigger="mouseenter click"
      placement="bottom-end" :theme="app.getTheme">
      <a :href="downloadUrl" target="_blank" id="downloadUrl"/>
      <Icon icon="arrow-sync" @click="goDownload"
        size="small" :text="$t('tips.newVersion')"
      />
      <template #content>
        <div id="new-feature" class="rich-text" 
          v-html="md.render(newFeature)"
        />
      </template>
    </tippy>
  </div>
  <div id="show-welcome" v-if="showEpigraph">
    <button class="btn-welcome" @click="showWelcome">
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
  padding: 4px 0.5rem;
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