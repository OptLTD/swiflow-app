<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { toast } from 'vue3-toastify'
import { request } from '@/support/index'
import BasicMenu from './widgets/BasicMenu.vue'
import SetHeader from './widgets/SetHeader.vue'
import { useI18n } from 'vue-i18n'
import { getProviders } from '@/config/models'
// @ts-ignore
import CodeMirror from '@/widgets/CodeMirror.vue'
import { showLLMConfigPopup } from '@/logics/popup'


const items = ref<ToolEntity[]>([])
const active = ref('' as string)
const current = ref({} as ToolEntity)

// Use global i18n to translate builtin tool labels when available
const { t, te } = useI18n({ 
  inheritLocale: true, 
  useScope: 'global' 
})

// Intent: Provide localized name/description for specific builtin tools
const labelOf = (item: ToolEntity) => {
  const key = item.uuid.replace('-', '')
  const keyName = `builtin.${key}Name`
  const keyDesc = `builtin.${key}Desc`
  const name =  te(keyName) ? t(keyName) : item.name 
  const desc = te(keyDesc) ? t(keyDesc) : item?.desc 
  return { name, desc }
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

const doSubmitTool = async (item: ToolEntity) => {
  try {
    const url = `/tool?act=set-tool&uuid=${item.uuid}`
    const resp = await request.post(url, item)
    if (resp && (resp as any)['errmsg']) {
      return toast((resp as any)['errmsg'])
    }
    return toast('TOOL SAVE SUCCESS')
  } catch (err) {
    console.error('get mcp:', err)
    return toast('ERROR:' + err)
  }
}

const builtin = [
  `chat2llm`, `image-ocr`,
  `command`, `python3`, `get-intent`
]
const isBaseLLM = computed(() => {
  const llmBaseType = [`chat2llm`, `image-ocr`, `get-intent`]
  return llmBaseType.includes(current.value?.uuid)
})
</script>

<template>
  <SetHeader :title="$t('menu.toolSet')" />
  <div id="tool-setting" class="set-view">
    <div id="tool-menu" class="set-menu">
      <BasicMenu :items="items" :active="active" 
        @click="onSelect" :keyby="'uuid'">
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
          </div>
        </template>
      </BasicMenu>
    </div>
    <div id="tool-panel" class="set-main">
      <div v-if="!active" class="tool-empty">
        {{ $t('common.noTools') }}
      </div>
      <div v-else class="tool-detail">
        <h3>{{ labelOf(current).name }}</h3>
        <p class="tool-kind" v-if="!isBaseLLM">
          {{ labelOf(current).desc }}
        </p>
        <div v-if="isBaseLLM" class="tool-config">
          <div class="form-group">
            <FormKit type="select" name="provider"
              :label="$t('setting.provider')" :disabled="true"
              v-model="toolProvider" :options="getProviders()" 
            >
              <template #suffixIcon>
                <button class="set-provider" @click="onChangeProvider">
                  <svg class="icon" viewBox="0 0 32 32" aria-hidden="true">
                    <circle cx="16" cy="16" r="4" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-miterlimit="10" stroke-width="2.08" />
                    <path d="M27.758,10.366l-1-1.732c-0.552-0.957-1.775-1.284-2.732-0.732L23.5,8.206C21.5,9.36,19,7.917,19,5.608V5c0-1.105-0.895-2-2-2h-2 c-1.105,0-2,0.895-2,2v0.608c0,
                      2.309-2.5,3.753-4.5,2.598L7.974,7.902C7.017,7.35,5.794,7.677,5.242,8.634l-1,1.732 c-0.552,0.957-0.225,2.18,0.732,2.732L5.5,13.402c2,1.155,2,4.041,0,5.196l-0.526,
                      0.304c-0.957,0.552-1.284,1.775-0.732,2.732 l1,1.732c0.552,0.957,1.775,1.284,2.732,0.732L8.5,23.794c2-1.155,4.5,0.289,4.5,2.598V27c0,1.105,0.895,2,2,2h2 c1.105,0,
                      2-0.895,2-2v-0.608c0-2.309,2.5-3.753,4.5-2.598l0.526,0.304c0.957,0.552,2.18,0.225,2.732-0.732l1-1.732 c0.552-0.957,0.225-2.18-0.732-2.732L26.5,18.598c-2-1.155-2-4.041,
                      0-5.196l0.526-0.304C27.983,12.546,28.311,11.323,27.758,10.366z" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-miterlimit="10" stroke-width="2.08"
                    />
                  </svg>
                </button>
              </template>
            </FormKit>
            <FormKit type="button" @click="onSubmitConfig">
              {{ $t('common.saveCfg') }}
            </FormKit>
          </div>
          <!-- Default prompt editor -->
          <div class="editor-group">
            <code-mirror class="editor-input" v-model="current.desc" />
          </div>
        </div>
        <pre v-else class="tool-prompt">{{ current.desc }}</pre>
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
.tool-config {
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
</style>