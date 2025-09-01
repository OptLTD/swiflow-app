<script setup lang="ts">
import { PropType, ref, watch, computed } from 'vue'
// @ts-ignore
import CodeMirror from '@/widgets/CodeMirror.vue';

const props = defineProps({
  model: {
    type: Object as PropType<MemEntity>,
    default: () => {}
  },
})

const isDisabled = computed(() => {
  return !props.model.bot
})

const theForm = ref<HTMLFormElement>()
const formModel = ref(props.model || {})
watch(() => props.model, (data, old) => {
  if (data.id != old.id) {
    formModel.value = {...data} as MemEntity
    return
  }
  Object.assign(formModel.value, {...data})
})

const emit = defineEmits(['submit'])
const doSubmitForm = () => {
  const form = theForm.value
  if (!form!.checkValidity()) {
    return
  }
  emit('submit', formModel.value)
}

const getFormModel = () => {
  const form = theForm.value
  if (form!.checkValidity()) {
    return formModel.value
  }
  return null
}

defineExpose({ getFormModel })
</script>

<template>
  <form ref="theForm">
    <div class="form-group">
      <FormKit
        type="text"
        name="subject"
        label="Subject"
        validation="required"
        outer-class="subject"
        v-model="formModel.subject"
        placeholder="请输入 Subject"
      />
      <FormKit type="button" @click="doSubmitForm">
        Submit
      </FormKit>
    </div>
    <div class="form-group editor-group">
      <code-mirror class="editor-input" 
        v-model="formModel.content" 
        :disabled="isDisabled"
      >
        <template #disabled-overlay>
          <div class="disabled-overlay">
            <div class="disabled-message">
              <p>请先选择关联的 Bot</p>
            </div>
          </div>
        </template>
      </code-mirror>
    </div>
  </form>
</template>

<style scoped>
@import url('@/styles/form.css');
.mem-header {
  margin: 0 0;
}
.mem-header>button{
  padding: 5px 10px;
  margin: -5px auto;
  margin-left: 25px;
}
.subject :deep(.formkit-messages) {
  position: absolute;
  right: 0;
  top: -6px;
}
.subject :deep(.formkit-wrapper) {
  width: var(--fk-max-width-input);
}
.formkit-outer[data-type='button'] {
  margin-top: 19.5px;
}
.editor-group{
  flex-direction: column;
}
.editor-input{
  border: 0px;
  margin: 5px -5px;
  width: -webkit-fill-available;
  height: calc(100vh - 150px);
  min-height: calc(100vh - 150px);
}

.disabled-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  pointer-events: all;
  user-select: none;
}

.disabled-message {
  background: #f5f5f5;
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 20px;
  text-align: center;
}

.disabled-message p {
  margin: 0;
  color: #666;
  font-size: 14px;
}
</style>
