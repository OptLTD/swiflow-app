<script setup lang="ts">
import { PropType, ref, watch } from 'vue'

const props = defineProps({
  config: {
    type: Object as PropType<CfgBrowserMeta>,
    default: () => {}
  },
})

const theForm = ref<HTMLFormElement>()
const formModel = ref(props.config || {})
watch(() => props.config, (data) => {
  data.engine =data.engine || 'bing'
  Object.assign(formModel.value, {...data})
})

const getFormModel = () => {
  const form = theForm.value
  if (form!.checkValidity()) {
    return formModel.value
  }
  return null
}

const engines = {
  'google': 'Google Search',
  'baidu': 'Baidu Search',
  'bing': 'Bing Search'
}

defineExpose({ getFormModel })
</script>

<template>
  
  <form ref="theForm" class="break">
      <div class="form-item engine-list">
        <label>Search Engine：</label>
        <div v-for="(name, key) in engines" :key="key">
          <input :id="key" type="radio" 
            :value="key" :true-value="key"
            v-model="formModel.engine"
          />
          <label class="option-label" :for="key">
            {{ name }}
          </label>
        </div>
      </div>
      <div class="form-item">
        <label>Ignore Sites：</label>
        <textarea v-model="formModel.ignores"/>
      </div>
  </form>
</template>

<style scoped>
@import url('@/styles/form.css');
.form-item > label{
  display: block;
  margin: 12px 0px;
  font-size: 1.1rem;
  font-weight: bold;
}
.engine-list input{
  margin: 0 0.5rem;
  margin-left: 1rem;
}
textarea{
  min-height: 5rem;
  width: -webkit-fill-available;
}
</style>
