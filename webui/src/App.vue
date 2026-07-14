<script setup lang="ts">
import { useAuthStore } from './stores/auth'
import { useLayoutStore } from './stores/layout'
import ToastHost from './components/ToastHost.vue'
import ChatPanel from './components/ChatPanel.vue'
import HeadTabBar from './components/HeadTabBar.vue'
import LoginDialog from './components/LoginDialog.vue'
import ResizeHandle from './components/ResizeHandle.vue'
import FileDropZone from './components/FileDropZone.vue'
import WelcomeView from './views/WelcomeView.vue'
import FilePreview from './views/FilePreview.vue'
import ExploreView from './views/ExploreView.vue'
import SettingsView from './views/SettingsView.vue'

const auth = useAuthStore()
const layout = useLayoutStore()
</script>

<template>
  <FileDropZone class="app-shell">
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

    <!-- Login dialog (server mode only) -->
    <LoginDialog v-if="auth.needsLogin" />

    <ToastHost />
  </FileDropZone>
</template>
