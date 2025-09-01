<script setup lang="ts">
import { ref, computed, PropType } from 'vue';
import { VueFinalModal } from 'vue-final-modal';
import { FormKit } from '@formkit/vue';
import { alert } from '@/support/index';
import { mcpTestNew } from '@/logics/mcp';
import { mcpSaveNew } from '@/logics/mcp';

const props = defineProps({
  mcpConfig: {
    type: Object as PropType<any>,
    required: true
  },
  show: {
    type: Boolean,
    default: false
  }
})

const isOK = ref(false)
const testing = ref(false)
const emit = defineEmits(['close', 'submit'])
// Parse env fields: { KEY: "{{key@type::tips}}" }
function parseEnv(envObj: Record<string, string>) {
  if (!envObj) return [];
  return Object.entries(envObj).map(([key, val]) => parseField(val, key));
}
// 新增：解析args数组
function parseArgs(argsArr: string[] = []) {
  // 只处理 {{key@type::tips}} 格式
  return argsArr
    .map((val) => parseField(val))
    .filter(item => item && item.key);
}
// 通用解析函数
function parseField(val: string, keyFromEnv?: string) {
  const match = val.match(/{{([^@]+)@([^:]+)::(.+?)}}/);
  let tips = '';
  let placeholder = '';
  let type: 'text' | 'password' = 'text';
  let tipsRaw = '';
  let key = keyFromEnv || '';
  if (match) {
    tipsRaw = match[3];
    type = match[2] === 'password' ? 'password' : 'text';
    key = match[1];
  } else {
    tipsRaw = val;
  }
  // 用 e.g. 切分
  const egSplit = tipsRaw.split(/e\.g\./i);
  if (egSplit.length > 1) {
    // 去除前一段结尾和后一段开头的[(:.,，。：、\s]
    tips = egSplit[0].replace(/[(:.,，。：、\s]+$/, '');
    placeholder = egSplit[1].replace(/^[(:.,，。：、\s]+/, '');
  } else {
    tips = tipsRaw;
  }
  return { key, varName: key, type, tips, placeholder };
}

const envList = computed(() => {
  const argFields = Array.isArray(props.mcpConfig.args) ? parseArgs(props.mcpConfig.args) : [];
  const envFields = parseEnv(props.mcpConfig.env || {});
  // 合并，key不重复，参数优先
  const all = [...argFields];
  for (const env of envFields) {
    if (env.key && !all.find(e => e.key === env.key)) {
      all.push(env);
    }
  }
  return all;
});

// FormKit表单模型
const formModel = ref<Record<string, string>>({})
// 初始化formModel
if (props.mcpConfig.env) {
  for (const k of Object.keys(props.mcpConfig.env)) {
    formModel.value[k] = ''
  }
}
// 初始化args字段
if (Array.isArray(props.mcpConfig.args)) {
  for (const arg of parseArgs(props.mcpConfig.args)) {
    if (arg.key && !(arg.key in formModel.value)) {
      formModel.value[arg.key] = ''
    }
  }
}

const hasEnv = computed(() => Object.keys(props.mcpConfig.env || {}).length > 0)
const hasArgs = computed(() => Array.isArray(props.mcpConfig.args) && parseArgs(props.mcpConfig.args).length > 0)
const envArgsTitle = computed(() => {
  if (hasEnv.value && hasArgs.value) return '参数与环境变量配置';
  if (hasEnv.value) return 'Env 配置';
  if (hasArgs.value) return '参数配置';
  return '配置';
})

const buildMcpConfig = () => {
  // 组装数据结构
  const name = props.mcpConfig.name;
  const uuid = props.mcpConfig.uuid || name;
  const command = props.mcpConfig.command;
  let args: string[] = [];
  if (Array.isArray(props.mcpConfig.args)) {
    args = props.mcpConfig.args.map((arg: string) => {
      const match = arg.match(/{{([^@]+)@[^:]+::.+}}/);
      if (match && formModel.value[match[1]] !== undefined) {
        return formModel.value[match[1]];
      }
      return arg;
    });
  }
  let env: Record<string, string> = {};
  if (props.mcpConfig.env) {
    for (const [envKey, envVal] of Object.entries(props.mcpConfig.env) as [string, string][]) {
      const match = envVal.match(/{{([^@]+)@[^:]+::.+}}/);
      if (match) {
        const varName = match[1];
        env[envKey] = formModel.value[varName] ?? '';
      } else {
        env[envKey] = formModel.value[envKey] ?? '';
      }
    }
  }
  const result = {
    mcpServers: {
      [uuid]: {
        command,
        name,
        uuid,
        args,
        env
      }
    }
  };
  return result
}

// 测试方法
const doTestMcp = async () => {
  try {
    testing.value = true
    const config = buildMcpConfig()
    const resp = await mcpTestNew(config)
    if (resp?.errmsg) {
      return alert(resp?.errmsg)
    } else {
      isOK.value = true
      return alert('success')
    }
  } catch (err) {
    return alert((err as Error).message)
  } finally{
    testing.value = false
  }
}

// 保存方法
const doInstall = async () => {
  if (!isOK.value) {
    return alert('请先检测')
  }
  try {
    const config = buildMcpConfig()
    const resp = await mcpSaveNew(config)
    if (resp.errmsg) {
     return alert(resp?.errmsg)
    }
    emit('submit', resp)
  } catch (err) {
    return alert((err as Error).message)
  }
}
</script>

<template>
  <VueFinalModal
    v-model="props.show"
    modalId="theMcpInstallModal"
    class="swiflow-modal-wrapper"
    content-class="modal-content"
    overlay-transition="vfm-fade"
    content-transition="vfm-fade"
    @update:modelValue="emit('close')"
  >
    <h2 class="modal-title">MCP Install</h2>
    <div class="mcp-modal-content">
      <!-- Left: MCP config info -->
      <div class="mcp-modal-section mcp-modal-section-left">
        <div class="mcp-card">
          <div class="mcp-card-title">{{ props.mcpConfig.name }}</div>
          <div class="mcp-card-desc">{{ props.mcpConfig.description }}</div>
          <div class="mcp-card-cmd-label">完整命令：</div>
          <div class="mcp-card-cmd">
            <code>{{ props.mcpConfig.command }}
              <template v-if="Array.isArray(props.mcpConfig.args) && props.mcpConfig.args.length"> 
                {{ props.mcpConfig.args.join(' ') }}
              </template>
              <template v-else-if="props.mcpConfig.args"> 
                {{ props.mcpConfig.args }}
              </template>
            </code>
          </div>
          <div class="mcp-card-row">
            <span class="mcp-badge" :class="'badge-' + (props.mcpConfig.command || 'other')">{{ props.mcpConfig.command }}</span>
            <span v-if="props.mcpConfig.homepage" class="mcp-card-link">
              <a :href="props.mcpConfig.homepage" target="_blank">{{ props.mcpConfig.homepage }}</a>
            </span>
          </div>
          <div v-if="props.mcpConfig.tags && props.mcpConfig.tags.length" class="mcp-card-tags">
            <span v-for="tag in props.mcpConfig.tags" :key="tag" class="mcp-card-tag">{{ tag }}</span>
          </div>
        </div>
      </div>
      <!-- Right: Env config info -->
      <div class="mcp-modal-section mcp-modal-section-right">
        <div class="mcp-modal-env-title">{{ envArgsTitle }}</div>
        <FormKit type="form" :actions="false" v-model="formModel">
          <div v-if="envList.length === 0" class="mcp-modal-env-empty">
            无环境变量配置
          </div>
          <FormKit
            v-for="env in envList"
            v-model="formModel[env.key]"
            :key="env.key" :name="env.key" 
            :placeholder="env.placeholder"
            :type="env.type" :help="env.tips"
            :label="env.varName || env.key" 
          />
        </FormKit>
      </div>
    </div>
    <div class="actions">
      <button class="btn-submit" :disabled="!isOK" @click="doInstall">
        {{ $t('common.install') }}
      </button>
      <button v-if="!testing" class="btn-submit" @click="doTestMcp">
        {{ $t('common.checkit') }}
      </button>
      <button v-else class="btn-submit btn-loading">
        {{ $t('common.checkit') }}
      </button>
      <button class="btn-cancel" @click="emit('close')">
        {{ $t('common.cancel') }}
      </button>
    </div>
  </VueFinalModal>
</template>

<style scoped>
@import url('@/styles/modal.css');
@import url('@/styles/form.css');
@import url('@/styles/mcp.css');
.mcp-modal-content {
  display: flex;
  padding: 0.5rem;
  font-size: 1.05em;
  max-width: 680px;
}
.mcp-modal-section {
  flex: 1;
  min-width: 220px;
  min-height: 360px;
}
.mcp-modal-section code {
  background-color: unset!important;
}
.mcp-modal-section-left {
  gap: 0.7em;
  display: flex;
  padding-right: 24px;
  flex-direction: column;
  border-right: 1px solid #eee;
}
.mcp-modal-section-right {
  gap: 0.7em;
  display: flex;
  flex-direction: column;
  padding-left: 24px;
}
.mcp-modal-env-title {
  font-weight: bold;
  margin-bottom: 8px;
  color: var(--color-primary);
  font-size: 1.08em;
}
.mcp-modal-env-empty {
  color: var(--color-secondary);
  font-size: 0.98em;
}
.actions {
  margin-top: 24px;
  text-align: right;
}
</style>