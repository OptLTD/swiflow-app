<script setup lang="ts">
import { PropType, computed } from 'vue'

// 临时补充类型定义，实际项目可移至 models.ts
interface McpTool {
  name: string;
  description: string;
}
interface McpStatus {
  active: boolean;
  enable: boolean;
  tools?: McpTool[];
  checked?: string[];
}

// 展示mcp tools 信息
const props = defineProps({
  status: {
    type: Object as PropType<McpStatus>,
    default: () => {}
  },
  server: {
    type: Object as PropType<McpServer>,
    default: () => {}
  }
})
const isEmpty = computed(() => {
  if (!props.status) {
    return true
  }
  const tools = props.status?.tools
  return tools?.length == 0
})
</script>

<template>
  <div v-if="!isEmpty" class="mcp-tool-list">
    <h2 class="mcp-title">{{ $t('common.mcpList') }} 
      <span v-if="server.name">({{ server.name }})</span>
    </h2>
    <ul>
      <li v-for="tool in status.tools" :key="tool.name" class="tool-item">
        <div class="tool-header">
          <span class="tool-name">{{ tool.name }}</span>
          <span class="tool-status" :class="{ 
            enabled: status.checked!.includes(tool.name), 
            disabled: !status.checked!.includes(tool.name) 
          }">
            {{ status.checked!.includes(tool.name) ? '可用' : '禁用' }}
          </span>
        </div>
        <div class="tool-desc">{{ tool.description }}</div>
      </li>
    </ul>
  </div>
  <div v-else class="mcp-tool-empty">{{ $t('common.noTools') }}</div>
</template>

<style scoped>
.mcp-tool-list {
  background: var(--bg-main);
  border-radius: 10px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
  margin: 0 auto;

  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  overflow: hidden;
  max-width: 950px;
  margin-top: 20px;
  height: calc(100vh - 200px);
}
[data-theme="dark"] .mcp-tool-list {
  box-shadow: 0 4px 16px rgba(0,0,0,0.32);
  border: 1px solid var(--color-tertiary);
}
.mcp-title {
  color: var(--color-primary);
  font-size: 1.3rem;
  font-weight: bold;
  margin-bottom: 12px;
  letter-spacing: 1px;
}
ul {
  margin: 0 0px;
  padding: 15px 25px;
  list-style: none;
  overflow-y:scroll;
  width: -webkit-fill-available;
}
.tool-item {
  padding: 10px 0 10px 0;
  transition: background 0.2s;
  border-bottom: 1px solid var(--color-divider);
}
.tool-item:last-child {
  border-bottom: none;
}
.tool-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.tool-name {
  font-size: 1.1rem;
  font-weight: 500;
  color: var(--color-primary);
}
.tool-status {
  font-size: 0.95rem;
  padding: 2px 10px;
  border-radius: 12px;
  margin-left: 10px;
  background: var(--bg-menu);
  color: var(--color-secondary);
}
.tool-status.enabled {
  background: var(--bg-menu);
  color: #2e7d32;
}
.tool-status.disabled {
  background: var(--bg-menu);
  color: #c62828;
}
.tool-desc {
  color: var(--color-tertiary);
  font-size: 0.98rem;
  line-height: 1.5;
}
[data-theme="dark"] .tool-desc {
  color: var(--color-secondary);
}
.mcp-tool-empty {
  text-align: center;
  color: var(--color-tertiary);
  font-size: 1.1rem;
  padding: 40px 0;
}
</style>
