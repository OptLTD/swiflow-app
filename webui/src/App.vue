<script setup lang="ts">
import { watch, onMounted, defineAsyncComponent } from 'vue'
import { useAuthStore } from './stores/auth'
import { useLayoutStore } from './stores/layout'
import { useSetupStore } from './stores/setup'
import ToastHost from './components/ToastHost.vue'
import HeadTabBar from './components/HeadTabBar.vue'
import SetupWizard from './components/SetupWizard.vue'
import ResizeHandle from './components/ResizeHandle.vue'
import FileDropZone from './components/FileDropZone.vue'
import WelcomeView from './views/WelcomeView.vue'
import LoginView from './views/LoginView.vue'

// Heavy views load on demand — keeps the Windows WebView2 first paint small.
const ChatPanel = defineAsyncComponent(() => import('./components/ChatPanel.vue'))
const FilePreview = defineAsyncComponent(() => import('./views/FilePreview.vue'))
const ExploreView = defineAsyncComponent(() => import('./views/ExploreView.vue'))
const SettingsView = defineAsyncComponent(() => import('./views/SettingsView.vue'))
const LightAppsView = defineAsyncComponent(() => import('./views/LightAppsView.vue'))

const auth = useAuthStore()
const layout = useLayoutStore()
const setup = useSetupStore()

async function maybeCheckSetup() {
  if (!auth.isAuthed) return
  if (setup.checked) return
  try {
    await setup.check()
  } catch {
    setup.checked = true
  }
}

onMounted(maybeCheckSetup)
watch(() => auth.isAuthed, maybeCheckSetup)

async function onSetupDone() {
  await setup.complete()
}
</script>

<template>
  <LoginView v-if="auth.needsLogin" />
  <FileDropZone v-else class="app-shell">
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
          <LightAppsView v-else-if="tab.type === 'light-apps'" />
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

    <SetupWizard
      v-if="setup.showWizard"
      @done="onSetupDone"
    />

    <ToastHost />
  </FileDropZone>
</template>
