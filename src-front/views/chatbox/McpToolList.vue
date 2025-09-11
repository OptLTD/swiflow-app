<script setup lang="ts">
import { Tippy } from 'vue-tippy'
import { ref, watch } from 'vue'
import { onMounted, PropType } from 'vue'
import { useAppStore } from '@/stores/app'
import { request, alert } from '@/support/index';
import { showSetupEnvModal } from '@/logics/popup'
import SwitchInput from '@/widgets/SwitchInput.vue'

const app = useAppStore()
const props = defineProps({
  tools: {
    type: Array as PropType<string[]>,
    default: () => []
  },
  enable: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['change'])
const lossCmd = ref<string[]>([])
const servers = ref<McpServer[]>([])
const checked = ref(props.tools || [])
const groupChecked = ref<Record<string, boolean>>({})
onMounted(async () => {
  // Load MCP list when component mounts
  await loadMcpList()
  await checkMcpEnv()
})
watch(() => props.tools, (val) => {
  checked.value = val || [] as string[]
  groupChecked.value = getGroupChecked()
})
watch(() => app.getMcpEnv, () => {
  checkMcpEnv()
})
const doActiveMcp = async (server: McpServer) => {
  try {
    server.loading = true
    const url = `/mcp?act=active&uuid=${server.uuid}`
    const resp = await request.post(url) as McpStatus
    if ((resp as any)?.errmsg) {
      throw (resp as any)?.errmsg
    }
    if (resp) {
      resp.enable = !!resp.enable
      server.status = resp as McpStatus
    }
    return resp
  } catch (err) {
    throw err
  } finally {
    server.loading = false
  }
}

const loadMcpList = async () => {
  try {
    const url = `/mcp?act=get-mcp`
    const resp = await request.post(url)
    if ((resp as any)?.errmsg) {
      throw (resp as any)?.errmsg
    }
    app.setMcpList(resp as McpServer[])
    servers.value = resp as McpServer[]

  } catch (err) {
    alert(err as string)
  }
}

const checkMcpEnv = async () => {
  const loss = [] as string[]
  for (var item of servers.value) {
    if (!item.status?.enable) {
      continue
    }
    // @ts-ignore
    if (!app.getMcpEnv[item.command]) {
      loss.push(item.command)
      continue
    }
    const status = await doActiveMcp(item)
    item.status = status || {} as McpStatus
  }
  groupChecked.value = getGroupChecked()
  lossCmd.value = [...new Set(loss)] // Remove duplicates from loss array
}

const getGroupChecked = () => {
  const result = {} as Record<string, boolean>
  servers.value.forEach((server) => {
    const tools = server.status.tools || []
    const items = tools.map((x: any) => x.name)
    const values = checked.value.filter(x => {
      return items.includes(x)
    })
    result[server.name] = values.length === items.length
  })
  return result
}

const toggleCheck = (item: any, server: McpServer) => {
  if (!props.enable || !server.status.enable) return
  const groupKey = `${server.name}:*`
  const tools = server.status.tools || []
  const toolNames = tools.map((x: any) => x.name)
  let newChecked = [...checked.value]

  // 如果 groupKey 存在，先拆成所有 tool name
  if (newChecked.includes(groupKey)) {
    newChecked = newChecked.filter(x => x !== groupKey)
    newChecked.push(...toolNames.filter(name => !newChecked.includes(name)))
  }

  // toggle 当前 tool
  const idx = newChecked.indexOf(item.name)
  if (idx === -1 && !item.enable) {
    newChecked.push(item.name)
  } else if (idx !== -1) {
    newChecked.splice(idx, 1)
  }

  // 如果所有 tool 都被选中，则只保留 groupKey
  if (toolNames.every(name => newChecked.includes(name))) {
    newChecked = newChecked.filter(x => !toolNames.includes(x))
    newChecked.push(groupKey)
  } else {
    // 否则移除 groupKey，只保留当前 group 的已选 tool name
    newChecked = [
      ...newChecked.filter(x => !toolNames.includes(x) && x !== groupKey),
      ...toolNames.filter(name => newChecked.includes(name))
    ]
  }

  checked.value = newChecked
  emit('change', [...checked.value])
}

const toggleGroup = (server: McpServer) => {
  if (!props.enable || !server.status.enable) return
  const groupKey = `${server.name}:*`
  const tools = server.status.tools || []
  const toolNames = tools.map((x: any) => x.name)
  let newChecked = checked.value.filter(x => x !== groupKey && !toolNames.includes(x))
  // 之前没选则追加groupKey，否则取消
  if (!checked.value.includes(groupKey)) {
    newChecked.push(groupKey)
  }
  checked.value = newChecked
  emit('change', [...checked.value])
}

const isGroupChecked = (server: McpServer) => {
  return checked.value.includes(`${server.name}:*`)
}
const isToolChecked = (tool: any, server: McpServer) => {
  const groupKey = `${server.name}:*`
  if (checked.value.includes(groupKey)) return true
  return checked.value.includes(tool.name)
}

const getCheckboxId = (val: string) => `select-${val}`

// Get missing command list for environment check
const getLossCmd = () => {
  return lossCmd.value
}

// Shake animation for warning button
const shakeElement = () => {
  const warningBtn = document.querySelector('.btn-warning') as HTMLElement
  if (!warningBtn) {
    console.warn("Warning button not found! Available buttons:", document.querySelectorAll('button'))
    return
  }
  try {
    warningBtn.classList.add('shake')
    setTimeout(() => {
      warningBtn.classList.remove('shake')
    }, 600)
  } catch (error) {
    console.error("CSS animation failed:", error)
  }
}

const onSwitchServer = async (server: McpServer, enable: boolean) => {
  if (enable) {
    await doActiveMcp(server)
    return
  }
  try {
    server.loading = true
    const url = `/mcp?act=disable&uuid=${server.uuid}`
    const resp = await request.post(url)
    if (!resp || (resp as any)['errmsg']) {
      alert((resp as any)['errmsg'])
      return
    }
    server.status.active = false
    server.status.enable = false
    alert('SUCCESS')
  } catch (err) {
    alert('ERROR:' + err)
  } finally {
    server.loading = false
  }
}

// Handle environment update callback
const handleEnvUpdate = async (mcpEnv: McpEnvMeta) => {
  // Update app store mcpEnv state
  app.setMcpEnv(mcpEnv)
  // Re-check MCP environment after update
  await checkMcpEnv()
}

// Expose methods to parent component
defineExpose({
  getLossCmd,
  shakeElement
})

</script>

<template>
  <tippy interactive :theme="app.getTheme" arrow placement="top-start" trigger="click">
    <slot name="default" v-if="$slots['default']" />
    <button class="btn-icon btn-tools" v-else />
    <template #content>
      <div class="tools-panel">
        <template v-for="(server, i) in servers" :key="i">
          <div class="dropdown-item" :class="{'disabled': !server.status.enable || !enable}"
            @click.stop="toggleGroup(server)">
            <input type="checkbox" :disabled="!server.status.enable || !enable" :checked="isGroupChecked(server)"
              @mouseup.prevent @mousedown.prevent />
            <span class="dropdown-group flex-stretch">{{ server.name }}</span>
            <SwitchInput v-if="!server.loading" :id="`switch-server-${server.name}`" size="small" :disabled="!enable"
              :modelValue="!!server.status.enable" @change="(val) => onSwitchServer(server, val)"
              style="margin-left:8px;" />
            <button v-else class="btn-icon btn-loading" />
          </div>
          <div v-for="tool in server.status.tools" @click.stop="toggleCheck(tool, server)" :key="tool.value"
            class="dropdown-item" :class="{'disabled': !server.status.enable || !enable}">
            <input type="checkbox" :id="getCheckboxId(tool.name)" :disabled="!server.status.enable || !enable"
              :checked="isToolChecked(tool, server)" @mouseup.prevent @mousedown.prevent />
            <span class="dropdown-label">{{ tool.name }}</span>
          </div>
        </template>
      </div>
    </template>
  </tippy>
  <button class="btn-warning"
   v-if="lossCmd.length" 
   @click="showSetupEnvModal">
    ⚠️ 缺少必要组件，请点击修复
  </button>
</template>

<style scoped>
.btn-tools{
  margin-right: 10px;
}
.btn-icon{
  width: 20px;
  height: 20px;
  min-width: 20px;
  min-height: 20px;
}
.btn-icon:hover{
  background-color: #aaa;
}
.tools-panel {
  margin: 4px 0 0 0;
  max-height: 220px;
  overflow-y: auto;
  overflow-x: hidden;
  min-width: 12rem;
  max-width: var(--fk-max-width-input);
}
.tools-panel .dropdown-group {
  font-weight: bold;
  display: flex;
}
.tools-panel .dropdown-item {
  display: flex;
  align-items: center;
  padding: 6px 5px;
  cursor: pointer;
  user-select: none;
}
.tools-panel .dropdown-item[disabled],
.tools-panel .dropdown-item.disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
.tools-panel .dropdown-item:hover {
  background: #f0f6ff;
}
.tools-panel .dropdown-item input[type="checkbox"] {
  margin-right: 8px;
}
.btn-warning {
  outline: none;
  border-width: 0;
  background-color: #fff3cd;
  /* border: 1px solid #ffeaa7; */
  color: #856404;
  padding: 8px 12px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  text-align: left;
  position: relative;
  transition: all 0.2s ease;
}

.btn-warning:hover {
  background-color: #ffeaa7;
  border-color: #fdcb6e;
}
/* CSS class to trigger shake animation */
.btn-warning.shake {
  animation: shake 0.6s ease-in-out;
}
@-webkit-keyframes shake {
  0%, 100% { transform: translateX(0); }
  10%, 30%, 50%, 70%, 90% { transform: translateX(-10px); }
  20%, 40%, 60%, 80% { transform: translateX(10px); }
}
@keyframes shake {
  0%, 100% { transform: translateX(0); }
  10%, 30%, 50%, 70%, 90% { transform: translateX(-10px); }
  20%, 40%, 60%, 80% { transform: translateX(10px); }
}


</style>