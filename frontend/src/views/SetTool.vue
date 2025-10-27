<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { toast } from 'vue3-toastify'
import { request } from '@/support/index'
import BasicMenu from './widgets/BasicMenu.vue'
import SetHeader from './widgets/SetHeader.vue'
import { computed, onMounted, ref } from 'vue'
import { getProviders } from '@/config/models'
import { getLabel } from '@/config/builtin.ts'
// @ts-ignore
import CodeMirror from '@/widgets/CodeMirror.vue'
import { showInputModal } from '@/logics/popup'
import { showLLMConfigPopup } from '@/logics/popup'

const items = ref<ToolEntity[]>([])
const active = ref('' as string)
const current = ref({} as ToolEntity)

// @ts-ignore
const { t, te } = useI18n({ 
  inheritLocale: true, 
  useScope: 'global' 
})

// Intent: Provide localized name/description for specific builtin tools
const labelOf = (item: ToolEntity) => {
  return getLabel(item.uuid, item.name, item?.desc)
}

onMounted(async () => {
  await doLoad()
})

const onSelect = (item: ToolEntity) => {
  active.value = item.name
  current.value = item
  // Initialize editable fields for configurable tools
  // const data: any = (current.value as any)?.data || {}
  llmConfig.value = (current.value as any)?.data || {}
  toolProvider.value = llmConfig.value.provider || ''
}

const doLoad = async () => {
  try {
    const url = `/tool?act=get-tools`
    const resp = await request.get(url)
    items.value = (resp || []) as ToolEntity[]
    // Auto-select the first item for better UX
    if (items.value.length) {
      active.value = items.value[0].uuid
      current.value = items.value[0]
    } else {
      active.value = ''
      current.value = {} as ToolEntity
    }
  } catch (err) {
    console.error('list tools:', err)
    return toast('ERROR:' + err)
  }
}


// Local state for configurable tools (provider and default prompt)
const llmConfig = ref({} as ModelMeta)
const toolProvider = ref('' as string)
const onChangeProvider = () => {
  showLLMConfigPopup(llmConfig.value, (data: any) => {
    current.value.data = data
    doSubmitTool(current.value)
  })
}

const onSubmitConfig = async () => {
  if (!current.value?.uuid) return
  await doSubmitTool(current.value)
}

const onCreatePy3Alias = async () => {
  await doSubmitTool(current.value)
}

const doSubmitTool = async (item: ToolEntity) => {
  try {
    const url = `/tool?act=set-tool&uuid=${item.uuid}`
    const resp = await request.post(url, item)
    if (resp && (resp as any)['errmsg']) {
      return toast((resp as any)['errmsg'])
    }
    return toast('SUCCESS')
  } catch (err) {
    console.error('get mcp:', err)
    return toast('ERROR:' + err)
  }
}

const newPy3Alias = () => {
  const item = {
    uuid: '', name: '',
    type: 'py3-alias', desc: '',
  } as ToolEntity
  active.value = item.uuid
  current.value = item
}

const editPy3Alias = (tool: ToolEntity) => {
  const props = {
    input: tool?.desc || '',
    tips: t('builtin.editDescTips'),
    title: t('builtin.editDescTitle'),
  }
  showInputModal(props, async (text: string) => {
    tool.desc = text
    await doSubmitTool(tool)
    return true
  })
}
const handleGenerate = async () => {
  const props = {
    tips: t('builtin.editDescTips'),
    title: t('builtin.editDescTitle'),
  }
  showInputModal(props, async (desc: string) => {
    const url = `/tool?act=gen-code`
    const resp = await request.post<any>(url, { desc })
    if (!resp || resp.errmsg) {
      return false
    }
    current.value.text = resp.result
    return true
  })
}

const builtin = [
  `chat2llm`, `image-ocr`,
  `command`, `python3`, `get-intent`
]
const isBuiltin = computed(() => {
  return builtin.includes(current.value?.name)
})
const isBaseLLM = computed(() => {
  const llmBaseType = [`chat2llm`, `image-ocr`, `get-intent`]
  return llmBaseType.includes(current.value?.uuid)
})
const isPy3Alias = computed(() => {
  return current.value?.type === 'py3-alias'
})
</script>

<template>
  <SetHeader :title="$t('menu.toolSet')" />
  <div id="tool-setting" class="set-view">
    <div id="tool-menu" class="set-menu">
      <BasicMenu :items="items" :active="active" @click="onSelect" :keyby="'uuid'">
        <template v-slot="{ item }">
          <div class="item-header">
            <h5>{{ labelOf(item).name }}</h5>
            <p>{{ labelOf(item).desc }}</p>
          </div>
          <div class="item-action">
            {{ builtin.includes(item.name)
            ? $t('common.builtin')
            : $t('common.custom')
            }}
            <a  v-if="item.type == 'py3-alias'"
              class="btn-modify" @click="editPy3Alias(item)">
              {{ $t('common.edit') }}
            </a>
          </div>
        </template>
      </BasicMenu>
      <button class="btn-add-new" @click="newPy3Alias">
        {{ $t('builtin.addPy3Alias') }}
      </button>
    </div>
    <div id="tool-panel" class="set-main">
      <div class="tool-detail">
        <div v-if="isBaseLLM" class="tool-config">
          <h3>{{ labelOf(current).name }}</h3>
          <div class="form-group">
            <FormKit type="select" name="provider" 
              :label="$t('setting.provider')" :disabled="true"
              v-model="toolProvider" :options="getProviders()">
              <template #suffixIcon>
                <Icon icon="icon-setting" size="small" 
                  color="var(--fk-bg-button)" 
                  @click="onChangeProvider" 
                />
              </template>
            </FormKit>
            <FormKit type="button" 
              @click="onSubmitConfig"
              :label="$t('common.save')"
            />
          </div>
          <!-- Default prompt editor -->
          <div class="editor-group">
            <code-mirror class="editor-input" 
              v-model="current.text" text-lang="md" 
            />
          </div>
        </div>
        <div v-else-if="isPy3Alias" class="py3-alias">
          <h3>{{ $t('builtin.py3aliasName') }}</h3>
          <div class="form-group">
            <FormKit type="text" label="Name" 
              name="name" v-model="current.name" 
            />
            <FormKit type="button" 
              :label="$t('common.save')" 
              @click="onCreatePy3Alias"
            />
            <div class="flex-stretch"></div>
            <FormKit type="button"  
              label="Generate Code"
              @click="handleGenerate"
            />
          </div>
          <!-- Default prompt editor -->
          <div class="editor-group">
            <code-mirror class="editor-input" 
              v-model="current.text" text-lang="python" 
            />
          </div>
        </div>
        <div v-else-if="isBuiltin">
          <h3>{{ labelOf(current).name }}</h3>
          <p class="tool-kind">
            {{ labelOf(current).desc }}
          </p>
          <pre class="tool-prompt">
            {{ current.desc }}
          </pre>
        </div>
      </div>
    </div>
  </div>

</template>

<style scoped>
@import url('@/styles/setting.css');

.tool-empty {
  text-align: center;
  color: var(--color-tertiary);
  font-size: 1.1rem;
  padding: 40px 0;
}

.tool-detail {
  padding: 10px 12px;
  /* Make children stack vertically and allow the config to grow */
  display: flex;
  flex-direction: column;
  height: 100%;
}
.tool-kind {
  margin: 0;
  color: var(--color-secondary);
}
.tool-prompt {
  white-space: pre-wrap; 
  background: var(--bg-light);
  border: 1px solid var(--color-divider);
  border-radius: 6px;
  padding: 10px 12px;
}

/* Config section should take remaining height for editor to fill */
.tool-config, .py3-alias {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0; /* allow flex children to shrink/grow */
}
.form-group{
  display: flex;
  column-gap: 10px;
  align-items: center;
}
.form-group :deep(.formkit-label) {
  z-index: 10;
  padding: 2px 5px;
  margin-top: -10px;
  position: absolute;
  background: var(--bg-light);
}
.form-group :deep([data-disabled]) {
  opacity: unset;
}
.editor-group {
  display: flex;
  flex: 1;
  min-height: 0; 
}
.editor-input {
  flex: 1;
  height: calc(100vh - 180px);
  margin: 0px 0px 10px 0px;
}

/* 让齿轮按钮看起来是可点击的图标 */
.set-provider {
  width: 32px;
  height: 32px;
  outline: unset;
  padding: 2px 2px;
  margin-right: 2px;
  border-radius: 6px;
  color: var(--fk-bg-button);
  border: 2px solid var(--fk-bg-button);
}
.set-provider:hover {
  background-color: var(--bg-menu);
}
.set-provider .icon{
  width: 24px; 
  height: 24px;
}
#tool-menu .btn-add-new {
  margin: 5px auto;
  font-weight: normal;
  width: -webkit-fill-available;
}
</style>