<script setup lang="ts">
import 'vue3-emoji-picker/css'
import { Tippy } from 'vue-tippy'
import * as emoji from 'node-emoji'
import EmojiPicker from 'vue3-emoji-picker'
import { getProviders } from '@/config/models'
import { PropType, ref, watch, computed } from 'vue'
import { FormKit } from '@formkit/vue'
// @ts-ignore
import CodeMirror from '@/widgets/CodeMirror.vue';
import SelectInput from '@/widgets/SelectInput.vue'
import { showProviderPopup } from '@/logics/popup'

const props = defineProps({
  from: {
    type: String as PropType<String>,
    default: () => 'bot-model'
  },
  model: {
    type: Object as PropType<BotEntity>,
    default: () => ({})
  },
  servers: {
    type: Array as PropType<McpServer[]>,
    default: () => ([])
  },
  isMulti: {
    type: Boolean,
    default: false
  },
})

const formModel = ref({
  ...props.model,
  tools: props.model?.tools || []
})
watch(() => props.model, (data, old) => {
  if (data.uuid != old.uuid) {
    formModel.value = {...data,
      tools: data?.tools || []
    }
  }
})

const onSelectEmoji = (item: any) => {
  const name = emoji.which(item.i)
  formModel.value.emoji = name as string
}

const doSetupApiKey = () => {
  if (!props.isMulti) {
    return
  }
  showProviderPopup(formModel.value.provider, (data: any) => {
    formModel.value.provider = data['name']
    console.log('provider', data['name'])
  })
}

const showBtn = computed(() => {
  return props.from == 'setting'
})

const mcpTools = computed(() => {
  if (!props.servers?.length) {
    return [];
  }
  const result = [] as any[]
  props.servers.forEach((item) => {
    const {enable, checked, tools = []} = item.status
    if (!tools && checked) {
      return checked.map(x => {
        return {
          group: item.uuid,label:x,
          value:`${item.uuid}:${x}`,
          disabled: !enable,
        }
      })
    }
    const list = tools!.map(x => {
      return {
        group: item.uuid, label: x.name,
        value:`${item.uuid}:${x.name}`,
        disabled: !enable,
      }
    })
    result.push(...list)
  })
  return result
})

const emit = defineEmits(['submit'])
const doSubmitForm = () => {
  emit('submit', formModel.value)
}

defineExpose({ getFormModel: () => formModel.value })
</script>

<template>
  <FormKit type="form" :actions="false">
    <div class="form-group">
      <FormKit
          type="text"
          name="name"
          label="Bot Name"
          v-model="formModel.name"
          validation="required"
          outer-class="bot-name"
          placeholder="请输入Bot名称"
        >
        <template #prefix>
          <tippy interactive 
            theme="transparent" 
            trigger="click"
            :arrow="false">
            <template #default>
              <button class="btn-emoji" type="button">
                {{ emoji.get(formModel.emoji || ':man_technologist:') }}
              </button>
            </template>
            <template #content>
              <EmojiPicker @select="onSelectEmoji" native/>
            </template>
          </tippy>
        </template>
      </FormKit>
      <SelectInput v-model="formModel.tools" 
        :options="mcpTools" label="Use Tools"
        grouped :disabled="!formModel.leader"
      />
      <FormKit
        type="select"
        name="provider"
        label="Model Provider"
        :options="getProviders()"
        :disabled="!$props.isMulti"
        v-model="formModel.provider"
      >
      <template #prefix>
        <Icon icon="icon-setting" size="small" 
          :disabled="!$props.isMulti"
          color="var(--fk-bg-button)" 
          @click="doSetupApiKey"
        />
      </template>
      </FormKit>
      <FormKit v-if="showBtn" type="button" @click="doSubmitForm">Submit</FormKit>
    </div>
    <div class="form-group editor-group" :class="props.from">
      <code-mirror class="editor-input" v-model="formModel.usePrompt" />
    </div>
  </FormKit>
</template>

<style scoped>
@import url('@/styles/form.css');
.form-group{
  margin: 0;
}
.editor-input{
  border: 0px;
  margin: 5px -0px;
  height: calc(100vh - 155px);
  width: -webkit-fill-available;
}
.btn-emoji{
  outline: none;
  font-size: 20px;
  margin-top: 0px;
  margin-left: 5px;
  margin-right: -5px;
  line-height: 24px;
  width: 24px;
  height: 24px;
  min-width: 24px;
  min-height: 24px;

  border: 0;
  padding: 0;
  cursor: pointer;
  position: relative;
}
.btn-emoji::before{
  display: none;
  position: absolute;
}
.btn-emoji:hover{
  background-color: var(--bg-light);
}

.v3-emoji-picker :deep(.v3-footer) {
  display: none !important;
}
.bot-model .editor-input{
  height: calc(100vh - 300px);
  margin: 0px 0px 10px 0px;
}
.bot-name :deep(.formkit-messages) {
  position: absolute;
  right: 0;
  top: -6px;
}
.formkit-outer[data-type='button'] {
  margin-top: 19.5px;
}
.btn-gear {
  margin-left: 5px;
}
</style>
