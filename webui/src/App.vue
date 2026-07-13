<script setup lang="ts">
import { useLayoutStore } from './stores/layout'
import { useAuthStore } from './stores/auth'
import ChatPanel from './components/ChatPanel.vue'
import HeadTabBar from './components/HeadTabBar.vue'
import ResizeHandle from './components/ResizeHandle.vue'
import WelcomeView from './views/WelcomeView.vue'
import FilePreview from './views/FilePreview.vue'
import ExploreView from './views/ExploreView.vue'
import SettingsView from './views/SettingsView.vue'
import LoginDialog from './components/LoginDialog.vue'

const layout = useLayoutStore()
const auth = useAuthStore()
</script>

<template>
  <div class="app-shell h-full flex flex-col overflow-hidden">
    <HeadTabBar />

    <!-- Main content area -->
    <div class="flex-1 flex min-h-0 overflow-hidden">
      <!-- Left: keep each tab mounted while switching -->
      <div class="flex-1 min-w-0 overflow-hidden relative">
        <div
          v-for="tab in layout.tabs"
          :key="tab.id"
          v-show="tab.id === layout.activeTabId"
          class="absolute inset-0 overflow-hidden"
        >
          <WelcomeView v-if="tab.type === 'welcome'" />
          <FilePreview v-else-if="tab.type === 'file'" :path="tab.path || ''" />
          <ExploreView v-else-if="tab.type === 'explore'" :path="tab.path" />
          <SettingsView v-else-if="tab.type === 'settings'" />
        </div>
      </div>

      <!-- Resize handle -->
      <ResizeHandle v-if="layout.chatPanelOpen" />

      <!-- Right: Chat panel -->
      <div
        v-if="layout.chatPanelOpen"
        class="shrink-0 bg-white overflow-hidden"
        :style="{ width: layout.chatPanelWidth + 'px' }"
      >
        <ChatPanel />
      </div>
    </div>

    <!-- Login dialog (server mode) -->
    <LoginDialog v-if="!auth.isAuthed" />
  </div>
</template>
