<script setup lang="ts">
import { alert, confirm } from '@/support';
import { VueFinalModal } from 'vue-final-modal'
import { ref, unref,PropType, onMounted } from 'vue';
import { mcpTestNew, mcpSaveNew } from '@/logics/mcp';
import { mcpTestSet, mcpSaveSet } from '@/logics/mcp';
import { mcpDelete, checkNetEnv } from '@/logics/mcp';
import { checkMcpEnv, doInstall } from '@/logics/mcp';
import FormMcpSet from '@/widgets/FormMcpSet.vue';
import FormMcpNew from '@/widgets/FormMcpNew.vue';
import { extractServers } from '@/logics/mcp'
import { useWebSocket } from '@/hooks/index'

const props = defineProps({
  loading: {
    type: Boolean,
    default: () => false
  },
  model: {
    type: Object as PropType<McpServer>,
    default: null
  },
})
const socket = useWebSocket()
const emit = defineEmits([
  'submit', 'cancel', 'delete'
])
const config = ref<McpServer>(props.model)
const setForm = ref<typeof FormMcpSet>()
const newForm = ref<typeof FormMcpNew>()

const netEnv = ref('')
const mcpEnv = ref({
  python: '', uvx: '',
  nodejs: '', npx: '',
  windows: false,
})
const isOK = ref(false)
const testing = ref(false)
const loading = ref(false)
const pyRuning = ref(false)
const jsRuning = ref(false)
const stdout = ref({
  show: false, uuid: '',
  lines: [] as string[],
})


const doInstallPython = async () => {
  pyRuning.value = true
  netEnv.value = await checkNetEnv()
  await doInstall(netEnv.value, 'py-uvx', ok => {
    pyRuning.value = false
    if (ok) alert('Python环境安装成功')
  })
}
const doInstallNodejs = async () => {
  jsRuning.value = true
  netEnv.value = await checkNetEnv()
  await doInstall(netEnv.value, 'js-npx', ok => {
    jsRuning.value = false
    if (ok) alert('Node.js环境安装成功')
  })
}

onMounted(() => {
  loading.value = true
  checkMcpEnv((info) => {
    mcpEnv.value = info
    loading.value = false
  })
  socket.useHandle('message', onMsg)
})

const onMsg = (msg: SocketMsg) => {
  const {uuid, lines} = stdout.value
  if (msg.action === 'stream' && msg.chatid == uuid) {
    lines.push(msg.detail as string)
  }
}

const doTest  = async () => {
  stdout.value.uuid = ''
  stdout.value.lines = []
  if (props.model.name) {
    return await doTestSet()
  }
  return await doTestNew()
}

const doTestNew = async () => {
  const data = unref(newForm)!.getFormModel()
  if (!data.servers) { return alert('invalid data') }
  try {
    testing.value = true
    const config = JSON.parse(data.servers)
    const server = extractServers(config)
    stdout.value.uuid = server.uuid || ''
    const resp = await mcpTestNew(server)
    if (resp.errmsg) {
      alert(resp.errmsg)
      stdout.value.show = true
      return
    }
    alert('SUCCESS')
    isOK.value = true
  } catch (err) {
    alert((err as Error).message)
  } finally {
    testing.value = false
  }
}
const doTestSet = async () => {
  const data = unref(setForm)!.getFormModel()
  if (!data) { return alert('invalid data')}
  try {
    isOK.value = false
    testing.value = true
    const uuid = props.model.uuid || data['uuid']
    stdout.value.uuid = props.model.name || uuid
    const resp = await mcpTestSet(data, uuid)
    if (resp?.errmsg) {
      alert(resp.errmsg)
      stdout.value.show = true
      return
    }
    isOK.value = true
    alert("SUCCESS")
  } catch (err) {
    alert((err as Error).message)
    console.error('test-cfg:', err)
  } finally {
    testing.value = false
  }
}

const doSave  = async () => {
  stdout.value.uuid = ''
  stdout.value.show = false
  stdout.value.lines = []
  if (props.model.uuid) {
    return await doSaveSet()
  }
  return await doSaveNew()
}

const doSaveNew = async () => {
  const data = unref(newForm)!.getFormModel()
  if (!data.servers) { return alert('invalid data') }
  try {
    const config = JSON.parse(data.servers)
    const server = extractServers(config)
    const resp = await mcpSaveNew(server)
    if (resp.errmsg) {
      alert(resp.errmsg)
      return
    }
    emit('submit', resp)
  } catch (err) {
    return alert((err as Error).message)
  }
}
const doSaveSet = async () => {
  const data = unref(setForm)!.getFormModel()
  if (!data) { return alert('invalid data')}
  try {
    const uuid = props.model.uuid || data['uuid']
    const resp = await mcpSaveSet(data, uuid)
    if (resp?.errmsg) {
      alert(resp.errmsg)
      return
    }
    emit('submit', resp)
  } catch (err) {
    alert((err as Error).message)
  }
}

const doDelete = async () => {
  const ok = await confirm('此操作不可恢复确定要删除么？')
  if (!ok) {
    return
  }
  try {
    const uuid = props.model.uuid
    const resp = await mcpDelete({}, uuid)
    if (resp?.errmsg) {
      alert(resp.errmsg)
      return
    }
    emit('delete')
  } catch (err) {
    alert((err as Error).message)
  }
}
</script>

<template>
  <VueFinalModal
    modalId="theMcpConfigModal"
    class="swiflow-modal-wrapper"
    content-class="modal-content"
    overlay-transition="vfm-fade"
    content-transition="vfm-fade"
  >
    <h2 class="modal-title">{{ $t('menu.mcpSet') }}</h2>
    <div class="door-box">
      <div class="env-block" v-if="!stdout.show">
        <div class="env-title">Python 环境</div>
        <div class="env-subtitle">
          通过 uvx 命令方式启动的 Mcp Server 依赖 Python 运行环境及 uvx 工具包
        </div>
        <template v-if="loading">
          <div class="env-loading">正在检测 Python 环境...</div>
        </template>
        <template v-else-if="mcpEnv['python'] || mcpEnv['uvx']">
          <div class="env-tip" v-if="mcpEnv['python']">
            Python: {{ mcpEnv['python'] as string }}
            <span  class="env-success" v-if="mcpEnv['python']">
              success
            </span>
            <span  class="env-failure" v-else>
              未检测到 Python
              <a href="https://www.python.org/" target="_blank">
                前往官网下载安装
              </a>
            </span>
          </div>
          <div class="env-tip" v-if="mcpEnv['uvx']">
            uvx: {{ mcpEnv['uvx'] as string }}
            <span  class="env-success">success</span>
          </div>
          <span  class="env-failure" v-else>
              uvx: 未检测到 uvx
              <a href="https://docs.astral.sh/uv/getting-started/installation/" target="_blank">
                前往官网下载
              </a>
          </span>
          <div class="env-tip" v-if="mcpEnv['uvx'] && mcpEnv['python']">
            Python 环境已安装，可正常使用。
          </div>
        </template>
        <template v-else>
          <div class="env-failure">未检测到 Python 环境，请先安装！</div>
          <ol class="env-list">
            <li>方法一：
              <a href="https://www.python.org/" target="_blank">
                前往 Python 官网下载安装
              </a>
            </li>
            <li v-show="!mcpEnv.windows">方法二：
              <button :disabled="pyRuning" @click="doInstallPython">
                {{ pyRuning ? '安装中...' : '一键脚本安装' }}
              </button>
            </li>
          </ol>
          <div class="env-tip" v-show="!mcpEnv.windows">
            脚本安装会自动下载并配置 Python、uvx，适用于大多数用户。
          </div>
        </template>
        <!-- node.js start -->
        <div class="env-title node-title">Node.js 环境</div>
        <div class="env-subtitle">
          通过 npx 命令方式启动的 Mcp Server 依赖 Node.js 运行环境及 npx 工具包
        </div>
        <template v-if="loading">
          <div class="env-loading">正在检测 Node.js 环境...</div>
        </template>
        <template v-else-if="mcpEnv['nodejs'] && mcpEnv['npx']">
          <div class="env-tip" v-if="mcpEnv['nodejs']">
            Node.js: {{ mcpEnv['nodejs'] as string }}
            <span  class="env-success">success</span>
          </div>
          <div class="env-tip" v-if="mcpEnv['uvx']">
            npx: {{ mcpEnv['npx'] as string }}
            <span  class="env-success">success</span>
          </div>
          <div class="env-tip">Node.js 环境已安装，可正常使用。</div>
        </template>
        <template v-else>
          <div class="env-failure">未检测到 Node.js 环境，请先安装！</div>
          <ol class="env-list">
            <li>方法一：
              <a href="https://nodejs.org/" target="_blank">
                前往 Node.js 官网下载安装
              </a>
            </li>
            <li v-show="!mcpEnv.windows">方法二：
              <button :disabled="jsRuning" @click="doInstallNodejs">
                {{ jsRuning ? '安装中...' : '一键脚本安装' }}
              </button>
            </li>
          </ol>
          <div class="env-tip"  v-show="!mcpEnv.windows">
            脚本安装会自动下载并配置 Node.js、npx，适用于大多数用户。
          </div>
        </template>
      </div>
      <div class="stdout-block" v-if="stdout.show">
        <div class="stdout-title">调试输出</div>
        <pre class="stdout-content">{{ stdout.lines.join('\n') }}</pre>
      </div>
      <FormMcpNew v-if="!config.name" ref="newForm"/>
      <FormMcpSet v-else :config="config" ref="setForm"/>
    </div>
    <div class="actions">
      <button class="btn-submit" :disabled="!isOK" @click="doSave">
        {{ $t('common.saveCfg') }}
      </button>
      <button v-if="!testing" class="btn-submit" @click="doTest">
        {{ $t('common.testCfg') }}
      </button>
      <button v-else class="btn-submit btn-loading">
        {{ $t('common.testCfg') }}
      </button>
      <button class="btn-cancel" @click="emit('cancel')">
        {{ $t('common.cancel') }}
      </button>
      <template v-if="config.name">
      <div class="flex-stretch"></div>
      <button class="btn-delete" @click="doDelete">
        {{ $t('common.delete') }}
      </button>
      </template>
    </div>
  </VueFinalModal>
</template>

<style scoped>
@import url('@/styles/modal.css');
.door-box{
  max-width: 680px;
  min-width: 680px;
}
.env-block {
  margin: 0 8px;
  min-width: 160px;
}
.env-title {
  margin-top: 8px;
  font-size: 16px;
  font-weight: bold;
}
.node-title{
  margin-top: 15px;
}
.env-subtitle {
  font-size: 13px;
  margin-bottom: 10px;
  color: var(--color-secondary);
}
.env-success {
  color: #1aaf5d;
  font-weight: bold;
  margin-bottom: 8px;
}
.env-failure {
  color: #e53935;
  font-weight: bold;
  margin-bottom: 0px;
  font-size: 13px;
}
.env-loading {
  color: #888;
  margin-bottom: 8px;
}
.env-list {
  margin: 10px 0 10px 20px;
  padding-left: 0;
}
.env-tip {
  font-size: 12px;
  color: #888;
  margin-top: 4px;
}
.env-block a,
.env-block button{
  border: 0;
  padding: 0;
  text-decoration: underline;
}
.stdout-block {
  max-width: 50%;
  max-height: 50%;
  margin-right: 10px;
}
.stdout-title{
  font-size: 12px;
  font-weight: bold;
  margin-top: -5px;
}
.stdout-content {
  text-wrap: auto;
  margin-top: 5px;
  padding: 5px 5px;
  overflow-y: scroll;
  min-height: 22.5rem;
}
</style>