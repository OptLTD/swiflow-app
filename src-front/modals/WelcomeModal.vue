<script setup lang="ts">
import { ref } from 'vue'
import { useAppStore } from '@/stores/app'
import { request } from '@/support'
import { VueFinalModal } from 'vue-final-modal'

const app = useAppStore()
const props = defineProps({
  epigraph: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['submit', 'cancel'])

const onSubmit = async () => {
  emit('submit')
}

const onCancel = () => {
  emit('cancel')
}
</script>

<template>
  <VueFinalModal
    class="flex justify-center items-center"
    content-class="welcome-modal"
    :click-to-close="false"
  >
    <div class="welcome-container">
      <div class="welcome-header">
        <h2>欢迎使用专属版本</h2>
      </div>
      <div class="welcome-content">
        <div class="epigraph" v-if="epigraph">
          <p>{{ epigraph }}</p>
        </div>
        <div class="welcome-message">
          <p>感谢您选择我们的专属版本，我们为您提供了定制化的功能和服务。</p>
          <p>开始使用前，请花一点时间熟悉界面和功能。</p>
        </div>
      </div>
      <div class="welcome-footer">
        <button class="btn-primary" @click="onSubmit">开始使用</button>
      </div>
    </div>
  </VueFinalModal>
</template>

<style scoped>
.welcome-modal {
  display: flex;
  flex-direction: column;
  padding: 25px;
  border-radius: 8px;
  background: var(--vfm-bg, #fff);
  color: var(--vfm-text, #000);
  max-width: 500px;
  width: 90%;
}

.welcome-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.welcome-header {
  text-align: center;
  margin-bottom: 10px;
}

.welcome-header h2 {
  font-size: 1.5rem;
  font-weight: bold;
  margin: 0;
}

.welcome-content {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.epigraph {
  font-style: italic;
  text-align: center;
  padding: 15px;
  background-color: #f5f5f5;
  border-radius: 6px;
  margin: 10px 0;
}

.welcome-message p {
  margin: 8px 0;
  line-height: 1.5;
}

.welcome-footer {
  display: flex;
  justify-content: center;
  margin-top: 10px;
}

.btn-primary {
  background-color: #4a6cf7;
  color: white;
  border: none;
  padding: 8px 20px;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 500;
  transition: background-color 0.2s;
}

.btn-primary:hover {
  background-color: #3a5ce5;
}

/* 暗色主题适配 */
:root[data-theme="dark"] .epigraph {
  background-color: #2a2a2a;
}
</style>