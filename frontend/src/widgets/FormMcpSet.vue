<script setup lang="ts">
import { FormKit } from '@formkit/vue'
import { ref, PropType, onMounted } from 'vue'

const props = defineProps({
  config: {
    type: Object as PropType<McpServer>,
    default: () => ({})
  },
})

const isCmd = ref(true)
const formModel = ref({ 
  uuid: '',
  name: '',
  cmd: '',
  url: '',
  env: '',
  args: '',
  type: 'stdio',
})

onMounted(() => {
  const data = props.config || {}
  formModel.value.uuid = data.uuid || ''
  formModel.value.name = data.name || ''
  formModel.value.url = data.url || ''
  formModel.value.cmd = data.command || 'npx'
  formModel.value.type = data.type || 'stdio'
  if (data.env && typeof data.env === 'object') {
    formModel.value.env = Object.entries(data.env)
      .map(([k, v]) => `${k}=${v}`)
      .join('\n')
  } else {
    formModel.value.env = data.env || ''
  }
  // 逆向转换args数组为字符串
  if (data.args && Array.isArray(data.args)) {
    formModel.value.args = data.args.join('\n')
  } else {
    formModel.value.args = data.args || ''
  }
  isCmd.value == !data.type || data.type == 'stdio'
})
 

const getFormModel = () => {
  const result = {} as McpServer
  const value = formModel.value
  result.env = Object.fromEntries(
    value.env.split(/\n|\r|;/)
      .map(line => line.trim())
      .filter(Boolean)
      .map(line => {
        const [key, ...rest] = line.split('=')
        return [key, rest.join('=')]
      })
  )
  result.uuid = props.config.uuid || ''
  result.args = value.args.split(/\n|\r|;/)
    .map(line => line.trim()).filter(Boolean)
  return Object.assign({}, value, result)
}

// 修正类型声明，options为数组
const types: { label: string, value: string }[] = [
  { value: 'stdio', label: 'STDIO' },
  { value: 'http', label: 'Streamable HTTP' }
]

defineExpose({ getFormModel })
</script>

<template>
  <FormKit
    type="form"
    v-model="formModel"
    :actions="false"
  >
    <FormKit
      type="radio"
      name="type"
      label="Mcp Type"
      :disabled="true"
      :options="types"
      v-model="formModel.type"
      :validation="'required'"
      outer-class="mcp-type"
    />
    <FormKit
      v-if="isCmd"
      type="text"
      name="name"
      label="Name"
      help="Display Name"
      v-model="formModel.name"
      :validation="'required'"
    />
    <FormKit
      v-if="isCmd"
      type="text"
      name="command"
      label="Command"
      help="npx or uvx command"
      v-model="formModel.cmd"
      :validation="'required'"
    />
    <FormKit
      v-else
      type="textarea"
      name="url"
      label="Url"
      help="Remote Url Address"
      v-model="formModel.url"
      :validation="'required'"
    />
    <FormKit
      v-if="isCmd"
      type="textarea"
      name="args"
      label="Args"
      v-model="formModel.args"
    />
    <FormKit
      v-if="isCmd"
      type="textarea"
      name="env"
      label="Env"
      v-model="formModel.env"
    />
  </FormKit>
</template>

<style scoped>
@import url('@/styles/form.css');
</style>
