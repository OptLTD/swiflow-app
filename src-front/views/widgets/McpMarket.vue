<script setup lang="ts">
import { request } from '@/support'
import { ref, computed, onMounted } from 'vue'
import McpInstallModal from '@/modals/McpInstallModal.vue'

const current = ref<string|undefined>('')
const servers = ref<(McpServer | any)[]>([])
const showImportModal = ref(false)
const selectedServer = ref<any>(null)

onMounted(async () => {
  await loadMcpList()
})

const loadMcpList = async () => {
  try {
    const url = `/mcp?act=servers`
    const resp = await request.post(url)
    if ((resp as any)?.errmsg) {
      throw (resp as any)?.errmsg
    }
    if (resp && (resp as any).length) {
      servers.value = resp as McpServer[]
    }
  } catch (err) {
    alert(err as string)
  }
}

const allTags = computed(() => {
  const tags = new Set<string>()
  servers.value.forEach(s => {
    s.tags = s.tags || [];
    s.tags.forEach((x: string) => {
      return tags.add(x)
    })
  })
  return Array.from(tags)
})

const filteredServers = computed(() => {
  if (!current.value) return servers.value
  return servers.value.filter(s => (s.tags || []).includes(current.value))
})

function badgeClass(cmd: string) {
  if (!cmd) return ''
  if (cmd === 'python') return 'badge-python'
  if (cmd === 'npx') return 'badge-npx'
  if (cmd === 'uvx') return 'badge-uvx'
  return 'badge-other'
}
function extractDomain(url: string) {
  try {
    return new URL(url).hostname.replace('www.', '')
  } catch {
    return url
  }
}
function truncate(str: string, len: number) {
  if (!str) return ''
  return str.length > len ? str.slice(0, len) + '…' : str
}
function openImportModal(server: any) {
  console.log('server', server)
  selectedServer.value = server
  showImportModal.value = true
}
</script>


<template>
  <div class="tags-bar">
    <span
      class="tag-btn"
      :class="{active: current === ''}"
      @click="current = ''"
    >全部</span>
    <span
      v-for="tag in allTags"
      :key="tag"
      class="tag-btn"
      :class="{active: current === tag}"
      @click="current = tag"
    >{{ tag }}</span>
  </div>
  <div class="mcp-servers-grid">
    <div
      v-for="server in filteredServers"
      :key="server.name"
      class="mcp-card border"
    >
      <div class="mcp-card-title">{{ server.name }}</div>
      <div class="mcp-card-desc">{{ truncate(server.description, 100) }}</div>
      <div class="mcp-card-row">
        <span class="mcp-badge" :class="badgeClass(server.command)">{{ server.command }}</span>
        <a v-if="server.homepage" :href="server.homepage" class="mcp-card-link" target="_blank">
          {{ extractDomain(server.homepage) }}
        </a>
        <button class="install-btn" @click="openImportModal(server)">Install</button>
      </div>
    </div>
  </div>
  <McpInstallModal
    v-if="showImportModal && selectedServer"
    :show="showImportModal" :mcpConfig="selectedServer"
    @close="showImportModal=false;selectedServer=null"
    @submit="showImportModal=false;selectedServer=null"
  />
</template>


<style scoped>
@import url('@/styles/mcp.css');
.mcp-servers-grid {
  width: 100%;
  display: grid;
  gap: 1.0rem;
  overflow-x: auto;
  overflow-y: scroll;
  margin: 0px -15px;
  padding: 15px 15px;
  height: calc(100vh - 140px);
  grid-template-columns: repeat(4, minmax(0, 1fr));
}
@media (max-width: 1360px) {
  .mcp-servers-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}
@media (max-width: 960px) {
  .mcp-servers-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
@media (max-width: 600px) {
  .mcp-servers-grid {
    grid-template-columns: 1fr;
    padding-left: 0.5rem;
    padding-right: 0.5rem;
  }
}
</style>