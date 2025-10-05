<script setup lang="ts">
import { onMounted, ref, unref, PropType } from 'vue'
import { VueFinalModal } from 'vue-final-modal'
import FormModel from '@/widgets/FormModel.vue';
import { request } from '@/support/index';

const props = defineProps({
  config: {
    type: Object as PropType<ModelMeta>,
    default: () => {}
  },
})

const errmsg = ref<string>('')
const models = ref<ModelResp>({})
const config = ref<ModelMeta>(props.config)
const theForm = ref<typeof FormModel>()
const emit = defineEmits(['submit', 'cancel'])
const doLoad = async () => {
  try {
    const url = `/setting?act=get-model`
    const resp = await request.get(url) as any
    if (resp?.useModel && !props.config?.provider) {
      config.value = resp.useModel as ModelMeta
    }
    models.value = resp.models || {}
  } catch (err) {
    console.log('load setting:', err)
  }
}

const doSubmit = async () => {
  const data = unref(theForm)!.getFormModel()
  if (!data) {
    errmsg.value =  'invalid data'
    return
  }
  return emit('submit', data)
}

onMounted(async () => {
  await doLoad()
})

// Remove signup function - no longer needed
</script>

<template>
  <VueFinalModal modalId="theLLMConfigModal" 
    class="swiflow-modal-wrapper" content-class="modal-content"
    overlay-transition="vfm-fade" content-transition="vfm-fade">
    <h2 class="modal-title">{{ $t('menu.modelSet') }}</h2>

    <div class="door-box">
      <img src="/images/art-llm.png" class="art-image">

      <!-- Show API key form directly -->
      <div class="form-content">
        <FormModel :config="config" ref="theForm" :models="models" />
      </div>
    </div>
    <div class="actions">
      <button class="btn-submit" @click="doSubmit">
        {{ $t('common.save') }}
      </button>
      <button class="btn-cancel" @click="emit('cancel')">
        {{ $t('common.cancel') }}
      </button>
    </div>
  </VueFinalModal>
</template>

<style scoped>
@import url('@/styles/modal.css');
:global(.modal-content){
  min-width: 680px!important;
  max-width: 680px!important;
}

/* Form content styling */
.form-content {
  width: 100%;
  box-sizing: border-box;
}

@media (max-width: 760px) {
  .art-image {
    display: none;
  }
  :global(.modal-content){
    min-width: var(--fk-max-width-input)!important;
    max-width: var(--fk-max-width-input)!important;
  }
}
</style>