<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { VueFinalModal } from 'vue-final-modal'

interface Props {
  input: string
  title: string
  tips: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  submit: [text: string]
  cancel: []
}>()

const input = ref('')

onMounted(() => {
  input.value = props.input || ''
})

const onSubmit = () => {
  emit('submit', input.value )
}

const onCancel = () => {
  emit('cancel')
}
</script>

<template>
  <VueFinalModal
    modalId="theInputModal"
    class="swiflow-modal-wrapper"
    content-class="modal-content modal-content-flex"
    overlay-transition="vfm-fade"
    content-transition="vfm-fade"
  >
    <h2>{{ props.title }}</h2>
    
    <div class="form-group form-group-flex">
      <label v-if="props.tips">
        {{ props.tips }}
      </label>
      <textarea  v-model="input"
        class="form-control"
      />
    </div>
    <div class="actions">
      <button class="btn-submit" @click="onSubmit">
        {{ $t('common.save') }}
      </button>
      <button class="btn-cancel" @click="onCancel">
        {{ $t('common.cancel') }}
      </button>
    </div>
  </VueFinalModal>
</template>

<style scoped>
@import url('@/styles/modal.css');

/* Add flex layout for modal content */
.modal-content-flex {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.form-group {
  margin-bottom: 16px;
}

/* Make form-group flex to fill remaining space */
.form-group-flex {
  flex: 1;
  display: flex;
  flex-direction: column;
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  padding: 8px 8px;
  font-weight: 500;
}

.form-control {
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
  resize: vertical;
  min-height: 240px;
}

/* Make textarea fill remaining space in flex container */
.form-control-flex {
  flex: 1;
  resize: none; /* Disable manual resize since it will auto-fill */
  min-height: 260px; /* Set a reasonable minimum height */
}

.form-control:focus {
  outline: none;
  border-color: #007bff;
  box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
}
</style>