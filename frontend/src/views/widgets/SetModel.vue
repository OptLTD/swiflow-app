<script setup lang="ts">
import { toast } from 'vue3-toastify';
import { debounce, isEmpty } from 'lodash-es';
import { ref, watch, computed } from 'vue'
import { onMounted, PropType, } from 'vue'
import { request } from '@/support/index';
import CardBox from '@/widgets/CardBox.vue';
import FormModel from '@/widgets/FormModel.vue';
import { allProviders } from '@/config/models';
import { shownProviders } from '@/config/models'

const props = defineProps({
  models: {
    type: Object as PropType<ModelResp>,
    default: () => {}
  }
})

const modelConfig = ref<ModelResp>({})
const showModels = computed(() => {
  return shownProviders
})

const modelLabel = computed(() => {
  var result: Record<string, string> = {}
  Object.keys(allProviders).forEach(key => {
    result[key] = allProviders[key].provider || ''
  })
  return result
})

const doSubmitSetting = async (name: string) => {
  try {
    const item = modelConfig.value[name]
    const { apiKey, apiUrl, models } = item
    const data = { apiKey, apiUrl, models }
    const url = `/setting?type=model&name=${name}`
    const resp = await request.post(url, data) as any
    if (resp['errmsg']) {
      return toast.error(resp['error'])
    }
    console.log('submit setting', resp, data)
  } catch (err) {
    console.error('Failed to load history:', err)
  }
}

const debounceSubmit = debounce(doSubmitSetting, 300)
const onInputChange = (key: string) => {
  return debounceSubmit(key)
}

const doUpdatePropsData = () => {
  for (var name in props.models) {
    if (!isEmpty(props.models[name])) {
      Object.assign(modelConfig.value[name], props.models[name])
    }
  }
}


watch(() => props.models, doUpdatePropsData)

onMounted(async () => {
  // prepare model data
  shownProviders.forEach(name => {
    if (!modelConfig.value[name]) {
      const cfg = allProviders[name]
      modelConfig.value[name] = {
        provider: cfg.provider, apiKey: '',
        apiUrl: cfg.apiUrl || '',
        models: cfg.models || [],
        useModel: cfg.useModel || (cfg.models && cfg.models[0] ? cfg.models[0] : ''),
        haskey: name !== 'ollama',
        status: 'ready',
      }
    }
  })
  doUpdatePropsData()
})
</script>

<template>
  <template v-for="key in showModels">
  <CardBox :title="`${modelLabel[key]}`">
    <div class="model-config">
      <FormModel 
        @change="onInputChange(key)"
      />
      <!-- :config="modelConfig[key] || {}" -->
    </div>
  </CardBox>
  </template>
</template>

<style scoped>
.model-config {
  padding: 1em;
}
</style>
