<script setup lang="ts">
import { allProviders } from '@/config/models'
import { shownProviders } from '@/config/models'
import { PropType, watch, computed } from 'vue'
import { onMounted, ref, unref } from 'vue'
import { FormKit } from '@formkit/vue'

const props = defineProps({
  config: {
    type: Object as PropType<ModelMeta>,
    default: () => {}
  },
  models: {
    type: Object as PropType<ModelResp>,
    default: () => {}
  }
})

const theForm = ref()
const formModel = ref(props.config || {})
watch(() => props.config, (data, old: any) => {
  if (!data || !data.provider) {
    return
  }
  if (data.provider != old?.provider) {
    formModel.value = {...data} 
    doSwitch(data.provider)
    return
  }
  Object.assign(formModel.value, {...data})
})

onMounted(() => {
  const config = props.config
  if (!config?.provider) {
    doSwitch('doubao')
  }
})

const providers = computed(() => {
  return shownProviders.map(key => ({
    label: allProviders[key].provider || '',
    value: key
  }))
})

const endpoints = computed(() => {
  var result: Record<string, string> = {}
  Object.keys(allProviders).forEach(key => {
    result[key] = allProviders[key].apiUrl || ''
  })
  return result
})
const getFormModel = () => {
  const context = theForm.value.node?.context;
  const isValid = unref(context?.state?.valid);
  if (!isValid) {
    console.log(isValid, context?.state, 'state')
    return false
  }
  return formModel.value
}

const doSwitch = (provider: string) => {
  if (!provider) {
    return
  }
  const config = props.models[provider]
  if (provider && config && config.apiKey) {
    Object.assign(formModel.value, {
      apiUrl: config.apiUrl,
      apiKey: config.apiKey,
      useModel: config.default,
    })
  } else if (provider && !config) {
    Object.assign(formModel.value, {
      apiUrl: '', apiKey: '', useModel: ''
    })
  }
  // default endpoint
  if (!formModel.value.apiUrl) {
    const config = endpoints.value || {}
    formModel.value.apiUrl = config[provider]
  }
  if (!formModel.value.useModel) {
    const selected =  allProviders[provider] || {}
    formModel.value.useModel = selected?.useModel
  }
}

defineExpose({ getFormModel })
</script>

<template>
  <FormKit
    type="form"
    ref="theForm"
    :actions="false"
    v-model="formModel"
  >
    <FormKit
      type="select" name="provider"
      :label="$t('setting.provider')"
      :options="providers"
      validation="required"
      v-model="formModel.provider"
      @change="() => doSwitch(formModel.provider)"
    />
    <FormKit
      type="password" name="apiKey"
      :label="$t('setting.apiSecret')"
      v-model="formModel.apiKey"
      validation="required"
      autocomplete="off"
    />
    <FormKit
      type="text" name="apiUrl"
      :label="$t('setting.apiBaseUrl')"
      v-model="formModel.apiUrl"
      validation="required"
    />
    <FormKit
      type="text" name="useModel"
      :label="$t('setting.useModel')"
      v-model="formModel.useModel"
      validation="required"
    />
  </FormKit>
</template>

<style scoped>
@import url('@/styles/form.css');
form{
  margin-top: 25px;
}
.form-item {
  display: flex;
  height: 32px;
  margin: 12px 12px;
  justify-content: space-between;
}
.form-item > label{
  width: 8rem;
  display: block;
  font-size: 1.1rem;
  font-weight: bold;
  text-align: right;
}
.form-item.err-msg{
  width: 100%;
  display: block;
  text-align: center;
}
</style>
