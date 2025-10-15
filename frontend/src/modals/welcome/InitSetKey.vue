<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { request } from '@/support/index'
import FormModel from '@/widgets/FormModel.vue'

const emit = defineEmits<{
  save: [data: any]
}>()

// Form reference and state
const modelForm = ref<typeof FormModel>()
const loading = ref(false)
const error = ref('')

// Internal state for model configuration
const modelConfig = ref<ModelMeta>()
const models = ref<ModelResp>({})

// Load model configuration from server
const loadModelConfig = async () => {
  try {
    loading.value = true
    error.value = ''
    
    const url = `/setting?act=get-model`
    const resp = await request.get(url) as any
    models.value = resp.models || {}
    
    let config: ModelMeta
    if (resp && resp.useModel) {
      config = resp.useModel as ModelMeta
    } else {
      config = {provider: ''} as ModelMeta
    }
    
    modelConfig.value = config
    return { modelConfig: config, models: models.value }
  } catch (err) {
    console.error('Failed to load model config:', err)
    error.value = 'Failed to load model configuration'
    throw err
  } finally {
    loading.value = false
  }
}

// Save API configuration to server
const saveApiConfig = async () => {
  const data = getFormModel()
  if (!data) {
    error.value = 'Please fill in all required fields'
    return false
  }
  
  try {
    loading.value = true
    error.value = ''
    
    const url = `/setting?act=set-model`
    const resp = await request.post(url, data)
    const errmsg = (resp as any)?.errmsg
    
    if (errmsg && errmsg !== 'success') {
      error.value = errmsg
      return false
    }
    
    // Emit save event to parent
    emit('save', data)
    return true
  } catch (err) {
    error.value = 'Failed to save API configuration'
    console.error('API config save error:', err)
    return false
  } finally {
    loading.value = false
  }
}

// Expose form capabilities to parent
const getFormModel = () => {
  return modelForm.value?.getFormModel()
}

// Initialize on mount
onMounted(async () => {
  if (!modelConfig.value || !models.value || Object.keys(models.value).length === 0) {
    await loadModelConfig()
  }
})

// Expose component capabilities
defineExpose({ handleSave: saveApiConfig })
</script>

<template>
  <div class="api-config-content">
    <h3>{{ $t('welcome.configureAiModel') }}</h3>
    <p class="step-description">
      {{ $t('welcome.configureAiModelDesc') }}
    </p>
    <FormModel
      v-if="modelConfig"
      :config="modelConfig"
      :models="models"
      ref="modelForm"
    />
  </div>
</template>

<style scoped>
.api-config-content {
  color: var(--text-main);
  text-align: center;
}

.api-config-content form{
  margin: 0 auto;
  text-align: left;
  max-width: var(--fk-max-width-input);
}

.api-config-content h3 {
  margin-bottom: 10px;
  color: var(--color-primary);
}

.step-description {
  line-height: 1.5;
  color: var(--color-text-secondary);
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-weight: 600;
  color: var(--text-main);
  font-size: 14px;
}

.form-group input,
.form-group select {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid var(--color-tertiary);
  border-radius: 8px;
  background: var(--bg-menu);
  color: var(--text-main);
  font-size: 14px;
  transition: all 0.3s ease;
  box-sizing: border-box;
}

.form-group input:focus,
.form-group select:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px var(--color-primary-light);
}

.form-group input::placeholder {
  color: var(--color-text-secondary);
}

.form-help {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-top: 5px;
  line-height: 1.4;
}

.form-help a {
  color: var(--color-primary);
  text-decoration: none;
}

.form-help a:hover {
  text-decoration: underline;
}

.test-connection {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 15px;
  padding: 12px 16px;
  background: var(--bg-main);
  border: 1px solid var(--color-tertiary);
  border-radius: 8px;
}

.test-connection.success {
  background: var(--color-success-light);
  border-color: var(--color-success);
  color: var(--color-success);
}

.test-connection.error {
  background: var(--color-error-light);
  border-color: var(--color-error);
  color: var(--color-error);
}

.test-icon {
  font-size: 16px;
}

.test-message {
  flex: 1;
  font-size: 14px;
  font-weight: 500;
}

/* Dark theme hover shadow */
@media (prefers-color-scheme: dark) {
  .form-group input:hover,
  .form-group select:hover {
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  }
}
</style>