<script setup lang="ts">
withDefaults(
  defineProps<{
    name:
    | 'file'
    | 'chat'
    | 'stop'
    | 'send'
    | 'home'
    | 'back'
    | 'edit'
    | 'lock'
    | 'close'
    | 'folder'
    | 'chat-off'
    | 'history'
    | 'refresh'
    | 'copy'
    | 'provider'
    | 'maximize'
    | 'minimize'
    | 'settings'
    | 'folder-open'
    size?: number
  }>(),
  { size: 16 },
)
</script>

<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 16 16"
    fill="none"
    stroke="currentColor"
    stroke-width="1.35"
    stroke-linecap="round"
    stroke-linejoin="round"
    class="shrink-0 block"
    aria-hidden="true"
  >
    <!-- Cog / gear -->
    <template v-if="name === 'settings'">
      <path
        d="M6.7 1.6h2.6l.35 1.55c.42.14.8.35 1.15.61l1.5-.55 1.3 1.3-.55 1.5c.26.35.47.73.61 1.15L14.4 6.7v2.6l-1.55.35c-.14.42-.35.8-.61 1.15l.55 1.5-1.3 1.3-1.5-.55c-.35.26-.73.47-1.15.61L9.3 14.4H6.7l-.35-1.55a4.7 4.7 0 0 1-1.15-.61l-1.5.55-1.3-1.3.55-1.5a4.7 4.7 0 0 1-.61-1.15L1.6 9.3V6.7l1.55-.35c.14-.42.35-.8.61-1.15l-.55-1.5 1.3-1.3 1.5.55c.35-.26.73-.47 1.15-.61L6.7 1.6z"
      />
      <circle cx="8" cy="8" r="2.1" />
    </template>
    <template v-else-if="name === 'folder'">
      <path d="M2 4.5A1 1 0 0 1 3 3.5h3l1 1.5h6a1 1 0 0 1 1 1v6.5a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V4.5z" />
    </template>
    <template v-else-if="name === 'folder-open'">
      <path d="M2 5.5V12a1 1 0 0 0 1 1h10a1 1 0 0 0 1-1V6.5a1 1 0 0 0-1-1H8.5L7 4H3a1 1 0 0 0-1 1v.5z" />
      <path d="M2 6.5h12" />
    </template>
    <template v-else-if="name === 'file'">
      <path d="M4.5 2h4.6L13 5.9V13a1 1 0 0 1-1 1H4.5a1 1 0 0 1-1-1V3a1 1 0 0 1 1-1z" />
      <path d="M9 2v4h4" />
    </template>
    <template v-else-if="name === 'chat'">
      <path d="M2.5 3.5h11a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H9l-2.5 2v-2h-4a1 1 0 0 1-1-1v-6a1 1 0 0 1 1-1z" />
    </template>
    <template v-else-if="name === 'chat-off'">
      <path d="M2.5 3.5h11a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H9l-2.5 2v-2h-4a1 1 0 0 1-1-1v-6a1 1 0 0 1 1-1z" opacity="0.45" />
      <path d="M3 13L13 3" />
    </template>
    <template v-else-if="name === 'close'">
      <path d="M4 4l8 8M12 4l-8 8" />
    </template>
    <template v-else-if="name === 'stop'">
      <rect x="4.5" y="4.5" width="7" height="7" rx="1" fill="currentColor" stroke="none" />
    </template>
    <template v-else-if="name === 'send'">
      <path d="M8 12.25V3.75M8 3.75L4.25 7.5M8 3.75l3.75 3.75" stroke-width="1.75" />
    </template>
    <template v-else-if="name === 'home'">
      <path d="M2.5 7.5L8 2.5l5.5 5" />
      <path d="M4 6.75V13h3.25V9.5h1.5V13H12V6.75" />
    </template>
    <template v-else-if="name === 'provider'">
      <rect x="2.5" y="3" width="11" height="4" rx="0.75" />
      <path d="M5 7v2.5M11 7v2.5" />
      <rect x="4" y="9.5" width="8" height="3.5" rx="0.75" />
    </template>
    <template v-else-if="name === 'back'">
      <path d="M9 3L4 8l5 5M4 8h9" />
    </template>
    <template v-else-if="name === 'history'">
      <path d="M3 4.5h10M3 8h7M3 11.5h4" />
    </template>
    <template v-else-if="name === 'refresh'">
      <!-- open arc (gap at top-right) + arrowhead -->
      <path d="M12.8 5.2A4.8 4.8 0 1 0 13.5 9.5" />
      <path d="M12.8 2.4v3.2h-3.2" />
    </template>
    <template v-else-if="name === 'copy'">
      <rect x="5.5" y="5.5" width="7.5" height="8" rx="1" />
      <path d="M3.5 10.5V3.5a1 1 0 0 1 1-1H10" />
    </template>
    <template v-else-if="name === 'edit'">
      <path d="M3.5 12.5l1.2-4.2L11 2.5l2.5 2.5-6.3 6.3L3.5 12.5z" />
      <path d="M9.2 4.3l2.5 2.5" />
    </template>
    <template v-else-if="name === 'lock'">
      <rect x="3.5" y="7" width="9" height="6.5" rx="1" />
      <path d="M5.5 7V5.2a2.5 2.5 0 0 1 5 0V7" />
    </template>
    <template v-else-if="name === 'maximize'">
      <!-- frame + large outbound arrows (readable at ~14px) -->
      <rect x="1.75" y="1.75" width="12.5" height="12.5" rx="1.5" />
      <path d="M8.5 8.5L3.75 3.75" />
      <path d="M3.75 6.75V3.75H6.75" />
      <path d="M7.5 7.5L12.25 12.25" />
      <path d="M12.25 9.25V12.25H9.25" />
    </template>
    <template v-else-if="name === 'minimize'">
      <!-- square frame + inset right bar (~1/3 width, gap from border) -->
      <rect x="2" y="2" width="12" height="12" rx="1.5" />
      <rect x="8.5" y="3.5" width="4" height="9" rx="0.75" fill="currentColor" stroke="none" />
    </template>
  </svg>
</template>
