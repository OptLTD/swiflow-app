<script setup lang="ts">
import { onMounted, ref, unref } from 'vue'
import { VueFinalModal } from 'vue-final-modal'
import FormModel from '@/widgets/FormModel.vue';
import { request, alert } from '@/support/index';

const props = defineProps({
  from: {
    type: String,
    default: ''
  },
  provider: {
    type: String,
    default: ''
  },
})

const errmsg = ref<string>('')
const config = ref<ModelMeta>()
const models = ref<ModelResp>({})
const theForm = ref<typeof FormModel>()
const emit = defineEmits(['submit', 'cancel'])
const doLoad = async (name: string) => {
  try {
    const url = `/setting?act=get-model`
    const resp = await request.get(url) as any
    models.value = resp.models || {}
    if (resp && resp.useModel) {
      config.value = resp.useModel as ModelMeta
    }
    if (props.from == 'provider' && models.value[name]) {
      config.value = models.value[props.from] as ModelMeta
    }
  } catch (err) {
  } finally {
    if (!config.value || !config.value.provider) {
      config.value = {provider: 'doubao'} as ModelMeta
    }
  }
}

const doSubmit = async () => {
  const data = unref(theForm)!.getFormModel()
  if (!data) {
    errmsg.value =  'invalid data'
    return
  }
  if (props.from == 'provider') {
    doSaveProvider(data)
  } else {
    doSaveUseModel(data)
  }
}

const doSaveUseModel = async (data: any) => {
  try {
    const url = `/setting?act=set-model`
    const resp = await request.post(url, data)
    errmsg.value = (resp as any)?.errmsg || 'success'
  } catch (err) {
    errmsg.value = err as string
  } finally {
    if (errmsg.value=='success') {
      // alert('SUCCESS')
      emit('submit')
    }
  }
}

const doSaveProvider = async (data: any) => {
  try {
    const url = `/setting?act=set-provider`
    const resp = await request.post(url, data)
    errmsg.value = (resp as any)?.errmsg || 'success'
  } catch (err) {
    errmsg.value = err as string
  } finally {
    if (errmsg.value=='success') {
      emit('submit', data)
      // alert('SUCCESS')
    }
  }
}

onMounted(async () => {
  await doLoad(props.provider)
})

// Remove signup function - no longer needed
</script>

<template>
  <VueFinalModal modalId="theProviderModal" 
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