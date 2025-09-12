<script setup lang="ts">
import { VueFinalModal } from 'vue-final-modal'
import { PropType, onMounted, ref } from 'vue'
import { request, alert } from '@/support';
import { useAppStore } from '@/stores/app'
import FormSetBot from '@/widgets/FormSetBot.vue'

const app = useAppStore()
const props = defineProps({
  loading: {
    type: Boolean,
    default: () => false
  },
  model: {
    type: Object as PropType<BotEntity|undefined>,
    default: null
  },
})

const model = ref(props.model)
const setBotForm = ref<typeof FormSetBot>()
const emit = defineEmits(['submit', 'cancel'])

const doLoadBot = async () => {
  const uuid = props.model?.uuid as string
  try {
    const url = `/bot?act=get-bot&uuid=${uuid}`
    const resp = await request.post(url) as BotEntity
    model.value = { ...resp, tools: resp.tools || [] }
  } catch (err) {
    console.error('get-bot:', err)
  }
}
const doSumbitRole = async (bot: BotEntity) => {
  try {
    const url = `/bot?act=set-bot&uuid=${bot.uuid}`
    const resp = await request.post(url, bot) as BotEntity
    if (!resp || (resp as any).errmsg) {
      return alert((resp as any).errmsg)
    }
  } catch (err) {
    console.error('set bot:', err)
  } finally {
    emit('submit')
  }
}

const doSubmitForm = async () => {
  const theForm = setBotForm.value
  const formData = theForm!.getFormModel()
  formData && doSumbitRole(formData)
}

onMounted(async () => {
  await doLoadBot()
})
</script>

<template>
  <VueFinalModal
    modalId="theBotModal"
    class="swiflow-modal-wrapper"
    content-class="modal-content"
    overlay-transition="vfm-fade"
    content-transition="vfm-fade"
  >
    <FormSetBot ref="setBotForm"
      :servers="app.getMcpList"
      from="bot-model" :model="model" 
    />
    <div class="actions">
      <button class="btn-submit" @click="doSubmitForm">
        {{ $t('common.save') }}
      </button>
      <button class="btn-cancel" @click="emit('cancel')">
        {{ $t('common.cancel') }}
      </button>
    </div>
  </VueFinalModal>
</template>

<style>
@import url('@/styles/modal.css');
</style>