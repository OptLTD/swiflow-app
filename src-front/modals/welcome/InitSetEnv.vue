<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { doInstall } from '@/logics/mcp'
import { checkMcpEnv } from '@/logics/mcp'
import { checkNetEnv } from '@/logics/mcp'
import { useAppStore } from '@/stores/app'

const emit = defineEmits<{
  envUpdated: [mcpEnv: any]
}>()

const app = useAppStore()
const netEnv = ref('')
const pyInstalling = ref(false)

// Internal state for MCP environment
const mcpEnv = ref({
  python: '',
  uvx: '',
  nodejs: '',
  npx: '',
  windows: false
})

// Timer management for periodic checking
let pythonCheckInterval: NodeJS.Timeout | null = null

// Check Python environment status
const checkPythonEnv = async () => {
  try {
    checkMcpEnv((info) => {
      app.setMcpEnv(info)
      mcpEnv.value = info
      emit('envUpdated', info)
      
      // Check if Python environment is configured after the async call
      if (info.python && info.uvx && pyInstalling.value) {
        // Python installation completed successfully
        pyInstalling.value = false
        
        // Clear the periodic checking interval
        if (pythonCheckInterval) {
          clearInterval(pythonCheckInterval)
          pythonCheckInterval = null
        }
      }
    })
  } catch (err) {
    console.error('Python environment check error:', err)
  }
}

// Start periodic checking for Python environment
const keepCheckPythonEnv = () => {
  // Clear any existing interval
  if (pythonCheckInterval) {
    clearInterval(pythonCheckInterval)
  }
  
  // Start periodic checking every 3 seconds until installation is complete
  pythonCheckInterval = setInterval(async () => {
    await checkPythonEnv()
  }, 3000)
}

// Install Python environment
const installPython = async () => {
  try {
    pyInstalling.value = true
    
    // Get network environment configuration
    netEnv.value = await checkNetEnv()
    
    // Start periodic checking for installation progress
    keepCheckPythonEnv()
    
    await doInstall(netEnv.value, 'uvx-py', (success) => {
      if (!success) {
        // Stop checking if installation failed
        if (pythonCheckInterval) {
          clearInterval(pythonCheckInterval)
          pythonCheckInterval = null
        }
        pyInstalling.value = false
      }
      // If success, let the periodic check handle the completion
    })
  } catch (err) {
    // Stop checking on error
    if (pythonCheckInterval) {
      clearInterval(pythonCheckInterval)
      pythonCheckInterval = null
    }
    pyInstalling.value = false
    console.error('Python installation error:', err)
  }
}

// Cleanup function to clear intervals
const cleanup = () => {
  if (pythonCheckInterval) {
    clearInterval(pythonCheckInterval)
    pythonCheckInterval = null
  }
}

onMounted(() => {
  checkPythonEnv()
})

onUnmounted(() => {
  cleanup()
})

// Expose component capabilities
defineExpose({
  checkPythonEnv,
  installPython,
  cleanup
})
</script>

<template>
  <div class="step-content">
    <h3>{{ $t('welcome.configurePythonEnv') }}</h3>
    <p class="step-description">{{ $t('welcome.pythonEnvDesc') }}</p>
    <div class="env-config-content">
      <div class="env-status">
        <div class="env-item">
          <span class="env-label">Python:</span>
          <span class="env-value" :class="mcpEnv.python ? 'available' : 'unavailable'">
            {{ mcpEnv.python || $t('welcome.notInstalled') }}
          </span>
        </div>
        <div class="env-item">
          <span class="env-label">UVX:</span>
          <span class="env-value" :class="mcpEnv.uvx ? 'available' : 'unavailable'">
            {{ mcpEnv.uvx || $t('welcome.notInstalled') }}
          </span>
        </div>
      </div>
      
      <!-- Installation progress indicator -->
      <div v-if="pyInstalling" class="installation-progress">
        <div class="progress-spinner">⏳</div>
        <p>{{ $t('welcome.installingPython') }}</p>
        <p class="progress-tip">
          {{ $t('welcome.installationTip') }}
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.step-content {
  text-align: center;
}

.step-content h3 {
  color: var(--color-primary);
  margin-bottom: 10px;
}

.step-description {
  color: var(--color-text-secondary);
}

.env-config-content {
  /* Use global layout patterns - no custom background needed */
}

.env-status {
  display: flex;
  flex-direction: column;
  gap: 15px;
  margin: 0 auto;
  margin-bottom: 20px;
  max-width: 400px;
}

.env-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--bg-menu);
  border-radius: 8px;
  border: 1px solid var(--color-tertiary);
  transition: all 0.3s ease;
}

.env-item:hover {
  background: var(--color-bg-hover);
}

.env-label {
  font-weight: 600;
  font-size: 14px;
  color: var(--text-main);
}

.env-value {
  font-size: 14px;
  padding: 4px 8px;
  border-radius: 4px;
  font-weight: 500;
}

.env-value.available {
  background: var(--color-success);
  color: white;
}

.env-value.unavailable {
  background: var(--color-error);
  color: white;
}

.installation-progress {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 20px;
  background: var(--bg-main);
  border-radius: 8px;
  border: 1px solid var(--color-primary);
}

.progress-spinner {
  font-size: 24px;
  animation: spin 2s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.installation-progress p {
  margin: 0;
  color: var(--color-primary);
  font-weight: 500;
}

.progress-tip {
  font-size: 12px !important;
  color: var(--color-text-secondary) !important;
  font-weight: normal !important;
}
</style>