<script setup lang="ts">
import { useLayoutStore } from '../stores/layout'
import { isDesktop, toggleMaximize } from '../lib/desktop'
import SvgIcon from './SvgIcon.vue'

const layout = useLayoutStore()

function onTitlebarDblClick(e: MouseEvent) {
  if (!isDesktop()) return
  const el = e.target as HTMLElement
  if (el.closest('button, a, input, select, textarea')) return
  toggleMaximize()
}
</script>

<template>
  <div
    class="header-tab-bar h-9 shrink-0 flex items-center bg-neutral-100 border-b border-neutral-200 select-none"
    @dblclick="onTitlebarDblClick"
  >
    <!-- macOS traffic light spacer -->
    <div class="traffic-light-spacer" aria-hidden="true" />

    <!-- Tabs -->
    <div class="flex-1 flex items-stretch overflow-x-auto min-w-0 h-full">
      <button
        v-for="tab in layout.tabs"
        :key="tab.id"
        class="px-3 text-sm flex items-center justify-center gap-1.5 shrink-0 border-r border-neutral-200 transition-colors leading-none"
        :class="[
          tab.id === layout.activeTabId ? 'bg-white text-neutral-900' : 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200',
          tab.type === 'welcome' ? 'w-9 px-0' : '',
        ]"
        :title="tab.type === 'welcome' ? 'Home' : tab.title"
        @click="layout.activateTab(tab.id)"
      >
        <SvgIcon v-if="tab.type === 'welcome'" name="home" :size="15" />
        <template v-else>
          <span class="truncate max-w-[120px] leading-none">{{ tab.title }}</span>
          <span
            v-if="tab.closable"
            class="w-4 h-4 flex items-center justify-center rounded hover:bg-neutral-300 text-neutral-500"
            @click.stop="layout.closeTab(tab.id)"
          >
            <SvgIcon name="close" :size="12" />
          </span>
        </template>
      </button>
    </div>

    <!-- Right actions -->
    <div class="shrink-0 flex items-center gap-0.5 px-1.5 border-l border-neutral-200 h-full">
      <button
        class="w-8 h-8 flex items-center justify-center rounded hover:bg-neutral-200 text-neutral-600"
        title="Settings"
        @click="layout.openSettings()"
      >
        <SvgIcon name="settings" :size="16" />
      </button>
      <button
        class="w-8 h-8 flex items-center justify-center rounded transition-colors"
        :class="layout.chatPanelOpen
          ? 'bg-neutral-200 text-neutral-900'
          : 'text-neutral-500 hover:bg-neutral-200 hover:text-neutral-700'"
        :title="layout.chatPanelOpen ? 'Hide Chat' : 'Show Chat'"
        @click="layout.toggleChatPanel()"
      >
        <SvgIcon :name="layout.chatPanelOpen ? 'chat' : 'chat-off'" :size="16" />
      </button>
    </div>
  </div>
</template>
