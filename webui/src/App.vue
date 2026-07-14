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
      <!-- Left: non-chat content tabs -->
      <div
        v-show="!layout.isChatTabActive"
        class="flex-1 min-w-0 overflow-hidden relative"
      >
        <div
          v-for="tab in layout.tabs"
          :key="tab.id"
          v-show="tab.id === layout.activeTabId && tab.type !== 'chat'"
          class="absolute inset-0 overflow-hidden"
        >
          <WelcomeView v-if="tab.type === 'home'" />
          <FilePreview v-else-if="tab.type === 'file'" :path="tab.path || ''" />
          <ExploreView v-else-if="tab.type === 'explore'" :path="tab.path" />
          <SettingsView v-else-if="tab.type === 'settings'" />
        </div>
      </div>

      <!-- Maximized chat tabs (one panel per session; same session = one tab) -->
      <div
        v-show="layout.isChatTabActive"
        class="flex-1 min-w-0 bg-white overflow-hidden relative"
      >
        <div
          v-for="tab in layout.chatTabs"
          :key="tab.id"
          v-show="tab.id === layout.activeTabId"
          class="absolute inset-0 overflow-hidden"
        >
          <ChatPanel expanded :session-key="tab.path!" />
        </div>
      </div>

      <!-- Sidebar chat (unmounted while a chat tab is focused to avoid dual SSE) -->
      <ResizeHandle v-if="layout.showChatSidebar" />
      <div
        v-if="layout.showChatSidebar"
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
