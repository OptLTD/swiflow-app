<script setup lang="ts">
import { Tippy } from 'vue-tippy'
import { ref,  unref, computed } from 'vue'
import { watch, onMounted, PropType } from 'vue'
import { useAppStore } from '@/stores/app'
import { request, alert } from '@/support/index';
import { showSetupEnvModal } from '@/logics/popup'
import SwitchInput from '@/widgets/SwitchInput.vue'
import { getTitle, setTitle } from '@/config/builtin'

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

// Installation status tracking
const install = ref({
  result: '',
  message: '',
  running: false,
})

// Single button control object
const buttonControl = computed(() => {
  // Priority 0: Missing components
  if (unref(lossCmd).length > 0) {
    console.log(unref(lossCmd), 'lossCmd')
    return {
      text: '⚠️ 缺少必要组件，请点击修复',
      classes: ['btn-warning'],
      onClick: showSetupEnvModal
    }
  }

  // Priority 1: Installing state
  if (install.value.running) {
    return {
      text: `🔄 ${install.value.message}`,
      classes: ['btn-installing'],
    }
  }
  
  // Priority 2: Installation success
  if (install.value.result === 'success') {
    return {
      text: `✅ ${install.value.message}`,
      classes: ['btn-success'],
    }
  }
  
  // Priority 3: Installation error
  if (install.value.result === 'error') {
    return {
      text: `❌ ${install.value.message} - Click to retry`,
      classes: ['btn-error'],
      onClick: showSetupEnvModal
    }
  }
  
  // Priority 4: Some enabled MCP servers failed to activate
  const enabledServers = servers.value.filter(s => s.status?.enable)
  if (enabledServers.some(s => !s.status?.active)) {
    return {
      text: '❌ Some MCP servers failed to activate - Click to retry',
      classes: ['btn-error'],
      onClick: startMcpEnv
    }
  }
  return null
})
onMounted(async () => {
  await loadMcpEnv()
  await loadMcpList()
  await startMcpEnv()
})
watch(() => props.tools, (val) => {
  checked.value = val || [] as string[]
  groupChecked.value = getGroupChecked()
})
watch(() => app.getMcpEnv, () => {
  startMcpEnv()
}, {deep: true})


const loadMcpEnv = async () => {
  try {
    const url = '/toolenv?act=mcp-env'
    const resp = await request.get(url)
    resp && app.setMcpEnv(resp as McpEnvMeta)
    return resp
  } catch (err) {
    console.log('failed to load mcp env:', err)
  }
}

const doActiveMcp = async (server: McpServer) => {
  try {
    server.loading = true
    install.value.result = ""
    install.value.message = `Start run ${server.name}`
    const url = `/mcp?act=active&uuid=${server.uuid}`
    const resp = await request.post(url) as McpStatus
    const data = (resp as any)
    if (data && data?.errmsg) {
      install.value.result = 'error'
      install.value.message = data.errmsg
      throw data.errmsg
    }
    if (resp) {
      resp.enable = !!resp.enable
      server.status = resp as McpStatus
      const active = server.status?.active
      install.value.result = active ? 'success' : 'error'
      install.value.message = active
        ? `Start run ${server.name} success`
        : `Start run ${server.name} failed`
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

const startMcpEnv = async () => {
  const loss = [] as string[]
  const errors = [] as string[]
  install.value.running = true
  for (var item of servers.value) {
    if (!item.status?.enable) {
      continue
    }
    if (item.uuid === 'builtin') {
      setToolTitle(item.status)
      continue
    }
    // @ts-ignore
    if (!app.getMcpEnv[item.command]) {
      loss.push(item.command)
      continue
    }
    const status = await doActiveMcp(item)
    item.status = status || {} as McpStatus
    // collection error messages
    if (install.value.result === 'error') {
      errors.push(install.value.message)
    }
  }
  install.value.running = false
  if (errors.length > 0) {
    install.value.result = 'error'
    install.value.message = errors.join('\n')
  } else {
    install.value.result = 'success'
    install.value.message = 'All mcp run success'
    setTimeout(() => {
      install.value.result = ''
      install.value.message = ''
    }, 3000)
  }
  groupChecked.value = getGroupChecked()
  lossCmd.value = [...new Set(loss)] // Remove duplicates from loss array
}

const setToolTitle = (status: McpStatus) => {
  status.tools?.forEach((tool) => {
    if (getTitle(tool.name)) {
      tool.title = getTitle(tool.name)
    }
    setTitle(tool.name, tool.title)
  })
}

const getGroupChecked = () => {
  const result = {} as Record<string, boolean>
  servers.value.forEach((server) => {
    const tools = server.status.tools || []
    const items = tools.map((x: any) => {
      return `${server.uuid}:${x.name}`
    })
    const values = checked.value.filter(x => {
      return items.includes(x)
    })
    result[server.uuid] = values.length >= items.length
  })
  return result
}

const toggleCheck = (item: any, server: McpServer) => {
  if (!props.enable || !server.status.enable) return
  const groupKey = `${server.uuid}:*`
  const tools = server.status.tools || []
  const items = tools.map((x: any) => {
    return `${server.uuid}:${x.name}`
  })
  let newChecked = [...checked.value]

  // 如果 groupKey 存在，先拆成所有 tool name
  if (newChecked.includes(groupKey)) {
    newChecked = newChecked.filter(x => x !== groupKey)
    newChecked.push(...items.filter(name => !newChecked.includes(name)))
  }

  // toggle 当前 tool
  const key = `${server.uuid}:${item.name}`
  const idx = newChecked.indexOf(key)
  if (idx === -1 && !item.enable) {
    newChecked.push(key)
  } else if (idx !== -1) {
    newChecked.splice(idx, 1)
  }

  // 如果所有 tool 都被选中，则只保留 groupKey
  if (items.every(name => newChecked.includes(name))) {
    newChecked = newChecked.filter(x => !items.includes(x))
    newChecked.push(groupKey)
  } else {
    // 否则移除 groupKey，只保留当前 group 的已选 tool name
    newChecked = [
      ...newChecked.filter(x => !items.includes(x) && x !== groupKey),
      ...items.filter(name => newChecked.includes(name))
    ]
  }

  checked.value = newChecked
  emit('change', [...checked.value])
}

const toggleGroup = (server: McpServer) => {
  if (!props.enable || !server.status.enable) return
  const groupKey = `${server.uuid}:*`
  const tools = server.status.tools || []
  const items = tools.map((x: any) => {
    return `${server.uuid}:${x.name}`
  })
  let newChecked = checked.value.filter(x => {
    return x !== groupKey && !items.includes(x)
  })
  // 之前没选则追加groupKey，否则取消
  if (!checked.value.includes(groupKey)) {
    newChecked.push(groupKey)
  }
  checked.value = newChecked
  emit('change', [...checked.value])
}

const isGroupChecked = (server: McpServer) => {
  return checked.value.includes(`${server.uuid}:*`)
}
const isToolChecked = (tool: any, server: McpServer) => {
  const groupKey = `${server.uuid}:*`
  const tookKey = `${server.uuid}:${tool.name}`
  if (checked.value.includes(groupKey)) return true
  return checked.value.includes(tookKey)
}

const getCheckboxId = (val: string) => `select-${val}`

// Check if MCP is ready: lossCmd is empty and no installation errors
const getMcpReady = () => {
  const hasLossCmd = unref(lossCmd).length > 0
  const isRunning = install.value.running === true
  const hasMcpError = install.value.result === 'error'
  
  // Return true only when lossCmd is empty and no installation errors
  return  !hasLossCmd && !isRunning && !hasMcpError
}

const isInstalling = () => {
  return install.value.running
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

const isBuiltin = (server: McpServer) => {
  return server.type === 'builtin'
}
const isDisabled = (server: McpServer): boolean => {
  return !server.status.enable || !props.enable
}

// Expose methods to parent component
defineExpose({
  getMcpReady,
  isInstalling,
  shakeElement,
})

</script>

<template>
  <tippy interactive :theme="app.getTheme" arrow placement="top-start" trigger="click">
    <slot name="default" v-if="$slots['default']" />
    <template #content>
      <h3 class="tools-header">
        {{ $t('common.switchTools') }}
      </h3>
      <div class="tools-panel">
        <template v-for="(server, i) in servers" :key="i">
          <div  :class="{'disabled': isDisabled(server)}" 
            class="dropdown-item" @click.stop="toggleGroup(server)">
            <input type="checkbox" :checked="isGroupChecked(server)" 
              :disabled="isDisabled(server)" @mouseup.prevent @mousedown.prevent />
            <span class="dropdown-group flex-stretch">{{ server.name }}</span>
            <SwitchInput v-if="!server.loading" :id="`switch-server-${server.uuid}`" 
              :disabled="!enable || isBuiltin(server)" :modelValue="!!server.status.enable" 
              @change="(val) => onSwitchServer(server, val)" size="small"  style="margin-left:8px;" 
            />
            <icon v-else  icon="icon-loading" size="mini"/>
          </div>
          <div v-for="tool in server.status.tools" @click.stop="toggleCheck(tool, server)" 
            :key="tool.value" class="dropdown-item" :class="{'disabled': isDisabled(server)}">
            <input type="checkbox" :id="getCheckboxId(tool.name)" :disabled="isDisabled(server)"
              :checked="isToolChecked(tool, server)" @mouseup.prevent @mousedown.prevent />
            <span class="dropdown-label">{{ tool.title || tool.name }}</span>
          </div>
        </template>
      </div>
    </template>
  </tippy>
  
  <!-- Single dynamic status button -->
  <button v-if="buttonControl"
    :class="buttonControl.classes"
    @click="buttonControl.onClick!()">
    {{ buttonControl.text }}
  </button>
</template>

<style scoped>
.btn-tools{
  margin-right: 10px;
}
.tools-header {
  margin: 0 0;
  margin-top: 5px;
  padding: 5px 10px;
  font-size: small;
  border-radius: 5px;
  background-color: var(--bg-menu);
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
.btn-installing {
  outline: none;
  border-width: 0;
  background-color: #e3f2fd;
  color: #1976d2;
  padding: 8px 12px;
  border-radius: 4px;
  cursor: not-allowed;
  font-size: 12px;
  text-align: left;
  position: relative;
  transition: all 0.2s ease;
  opacity: 0.8;
}

.btn-success {
  outline: none;
  border-width: 0;
  background-color: #e8f5e8;
  color: #2e7d32;
  padding: 8px 12px;
  border-radius: 4px;
  cursor: not-allowed;
  font-size: 12px;
  text-align: left;
  position: relative;
  transition: all 0.2s ease;
}

.btn-error {
  outline: none;
  border-width: 0;
  background-color: #ffebee;
  color: #c62828;
  padding: 8px 12px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  text-align: left;
  position: relative;
  transition: all 0.2s ease;
}

.btn-error:hover {
  background-color: #ffcdd2;
}

.btn-warning {
  outline: none;
  border-width: 0;
  background-color: #fff3cd;
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