<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { VueFinalModal } from 'vue-final-modal'
import { alert } from '@/support/index'
import { doInstall, checkMcpEnv, checkNetEnv } from '@/logics/mcp'
import { useAppStore } from '@/stores/app'

const emit = defineEmits(['cancel'])
const app = useAppStore()

// Environment state
const netEnv = ref('')
const mcpEnv = ref(app.getMcpEnv)

// Installation states
const pyInstalling = ref(false)
const jsInstalling = ref(false)

// Timer management for periodic checking
let pythonCheckInterval: NodeJS.Timeout | null = null
let nodejsCheckInterval: NodeJS.Timeout | null = null

// Check environment status
const checkEnvironment = async () => {
  try {
    checkMcpEnv((info) => {
      app.setMcpEnv(info)
      mcpEnv.value = info
      if (isAllConfigured()) {
        alert('Environment setup completed successfully!')
      }
    })
  } catch (err) {
    console.error('Environment check error:', err)
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
    try {
      // Check environment and wait for state update
      await checkEnvironment()
      // Check if Python environment is configured after the async call
      if (isPythonConfigured()) {
        pyInstalling.value = false
        if (pythonCheckInterval) {
          clearInterval(pythonCheckInterval)
          pythonCheckInterval = null
        }
      }
    } catch (err) {
      console.error('Error checking Python environment:', err)
    }
  }, 3000)
  
  // Set a maximum timeout of 5 minutes to prevent infinite checking
  setTimeout(() => {
    if (pyInstalling.value && pythonCheckInterval) {
      clearInterval(pythonCheckInterval)
      pythonCheckInterval = null
      pyInstalling.value = false
      alert('Python installation timeout - please try again')
    }
  }, 300000) // 5 minutes
}

// Start periodic checking for Node.js environment
const keepCheckNodejsEnv = () => {
  // Clear any existing interval
  if (nodejsCheckInterval) {
    clearInterval(nodejsCheckInterval)
  }
  
  // Start periodic checking every 3 seconds until installation is complete
  nodejsCheckInterval = setInterval(async () => {
    try {
      // Check environment and wait for state update
      await checkEnvironment()
      // Check if Node.js environment is configured after the async call
      if (isNodejsConfigured()) {
        jsInstalling.value = false
        if (nodejsCheckInterval) {
          clearInterval(nodejsCheckInterval)
          nodejsCheckInterval = null
        }
      }
    } catch (err) {
      console.error('Error checking Node.js environment:', err)
    }
  }, 3000)
  
  // Set a maximum timeout of 5 minutes to prevent infinite checking
  setTimeout(() => {
    if (jsInstalling.value && nodejsCheckInterval) {
      clearInterval(nodejsCheckInterval)
      nodejsCheckInterval = null
      jsInstalling.value = false
      alert('Node.js installation timeout - please try again')
    }
  }, 300000) // 5 minutes
}

// Install Python environment
const installPython = async () => {
  try {
    pyInstalling.value = true
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
        alert('Failed to install Python environment')
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
    alert('Failed to install Python environment')
  }
}

// Install Node.js environment
const installNodejs = async () => {
  try {
    jsInstalling.value = true
    netEnv.value = await checkNetEnv()
    
    // Start periodic checking for installation progress
    keepCheckNodejsEnv()
    
    await doInstall(netEnv.value, 'js-npx', (success) => {
      if (!success) {
        // Stop checking if installation failed
        if (nodejsCheckInterval) {
          clearInterval(nodejsCheckInterval)
          nodejsCheckInterval = null
        }
        jsInstalling.value = false
        alert('Failed to install Node.js environment')
      }
      // If success, let the periodic check handle the completion
    })
  } catch (err) {
    // Stop checking on error
    if (nodejsCheckInterval) {
      clearInterval(nodejsCheckInterval)
      nodejsCheckInterval = null
    }
    jsInstalling.value = false
    console.error('Node.js installation error:', err)
    alert('Failed to install Node.js environment')
  }
}

// Check if environments are properly configured
const isPythonConfigured = () => {
  return !!(mcpEnv.value.python && mcpEnv.value.uvx)
}

const isNodejsConfigured = () => {
  return !!(mcpEnv.value.nodejs && mcpEnv.value.npx)
}

const isAllConfigured = () => {
  return isPythonConfigured() && isNodejsConfigured()
}



// Cleanup function to clear all intervals
const cleanup = () => {
  if (pythonCheckInterval) {
    clearInterval(pythonCheckInterval)
    pythonCheckInterval = null
  }
  if (nodejsCheckInterval) {
    clearInterval(nodejsCheckInterval)
    nodejsCheckInterval = null
  }
}

// Handle close modal with cleanup
const handleClose = () => {
  cleanup()
  emit('cancel')
}

onMounted(() => {
  checkEnvironment()
})

onUnmounted(() => {
  cleanup()
})
</script>

<template>
  <VueFinalModal
    modalId="theSetupEnvModal"
    class="swiflow-modal-wrapper"
    content-class="modal-content"
    overlay-transition="vfm-fade"
    content-transition="vfm-fade"
  >
    <h2> Environment Configuration </h2>
    <div class="env-block">
      <!-- Python Environment Section -->
      <div class="env-section">
        <div class="env-title">
          <span class="env-icon">🐍</span>
          Python Environment
          <a href="https://www.python.org/" target="_blank" class="official-link">Official Website</a>
        </div>
        <div class="env-subtitle">
          Python environment is required for code analysis, data processing, and advanced AI tools
        </div>
        
        <template v-if="isPythonConfigured()">
          <div class="env-status-grid">
            <div class="env-item">
              <span class="env-label">Python:</span>
              <span class="env-value available">{{ mcpEnv.python }}</span>
            </div>
            <div class="env-item">
              <span class="env-label">UVX:</span>
              <span class="env-value available">{{ mcpEnv.uvx }}</span>
            </div>
          </div>
          <div class="env-success-message">
            ✅ Python environment is properly configured
          </div>
        </template>
        <template v-else>
          <div class="env-status-grid">
            <div class="env-item">
              <span class="env-label">Python:</span>
              <span class="env-value unavailable">{{ mcpEnv.python || 'Not installed' }}</span>
            </div>
            <div class="env-item">
              <span class="env-label">UVX:</span>
              <span class="env-value unavailable">{{ mcpEnv.uvx || 'Not installed' }}</span>
            </div>
          </div>
          <div class="env-error-message">
            ❌ Python environment is not properly configured
          </div>
        </template>
      </div>

      <!-- Node.js Environment Section -->
      <div class="env-section">
        <div class="env-title">
          <span class="env-icon">📦</span>
          Node.js Environment
          <a href="https://nodejs.org/" target="_blank" class="official-link">Official Website</a>
        </div>
        <div class="env-subtitle">
          Node.js environment is required for running JavaScript-based MCP servers and tools
        </div>
        
        <template v-if="isNodejsConfigured()">
          <div class="env-status-grid">
            <div class="env-item">
              <span class="env-label">Node.js:</span>
              <span class="env-value available">{{ mcpEnv.nodejs }}</span>
            </div>
            <div class="env-item">
              <span class="env-label">NPX:</span>
              <span class="env-value available">{{ mcpEnv.npx }}</span>
            </div>
          </div>
          <div class="env-success-message">
            ✅ Node.js environment is properly configured
          </div>
        </template>
        <template v-else>
          <div class="env-status-grid">
            <div class="env-item">
              <span class="env-label">Node.js:</span>
              <span class="env-value unavailable">{{ mcpEnv.nodejs || 'Not installed' }}</span>
            </div>
            <div class="env-item">
              <span class="env-label">NPX:</span>
              <span class="env-value unavailable">{{ mcpEnv.npx || 'Not installed' }}</span>
            </div>
          </div>
          <div class="env-error-message">
            ❌ Node.js environment is not properly configured
          </div>
        </template>
      </div>
    </div>

    <div class="actions">
      <div class="install-buttons">
        <button 
          class="btn-secondary" @click="installNodejs"
          :disabled="jsInstalling || isNodejsConfigured()" 
        >
          {{ jsInstalling ? 'Installing Node.js...' : (isNodejsConfigured() ? 'Node.js Installed' : 'Install Node.js') }}
        </button>
        <button 
          class="btn-secondary" @click="installPython"
          :disabled="pyInstalling || isPythonConfigured()" 
        >
          {{ pyInstalling ? 'Installing Python...' : (isPythonConfigured() ? 'Python Installed' : 'Install Python') }}
        </button>
      </div>
      <button class="btn-outline" @click="handleClose">
        Cancel
      </button>
    </div>
  </VueFinalModal>
</template>

<style scoped>
@import url('@/styles/modal.css');
.setup-env-modal {
  max-width: 700px;
  width: 90%;
  max-height: 80vh;
}

.modal-header {
  text-align: center;
  margin-bottom: 30px;
}

.modal-header h2 {
  margin: 0 0 10px 0;
  font-size: 24px;
  font-weight: 600;
  color: var(--vfm-text, #333);
}

.modal-description {
  margin: 0;
  color: var(--text-muted, #666);
  font-size: 14px;
}

.env-block {
  margin: 10px 10px;
  /* max-height: 60vh; */
  overflow-y: auto;
  padding-right: 10px;
}

.env-section {
  padding: 15px;
  margin-bottom: 20px;
  border-radius: 8px;
  border: 1px solid var(--border-color, #e9ecef);
  background-color: var(--light-bg, #f8f9fa);
}

.env-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--vfm-text, #333);
}

.official-link {
  margin-left: 12px;
  font-size: 14px;
  font-weight: 400;
  color: #3498db;
  text-decoration: none;
  transition: color 0.2s ease;
}

.official-link:hover {
  color: #2980b9;
  text-decoration: underline;
}

.env-icon {
  font-size: 20px;
}

.env-subtitle {
  font-size: 13px;
  color: var(--text-muted, #666);
  margin-bottom: 15px;
  line-height: 1.4;
}

.env-loading {
  text-align: center;
  padding: 20px;
  color: var(--text-muted, #666);
  font-style: italic;
}

.env-status-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-bottom: 15px;
}

.env-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background-color: white;
  border-radius: 4px;
  border: 1px solid var(--border-color, #e9ecef);
}

.env-label {
  font-weight: 500;
  min-width: 60px;
}

.env-value {
  font-family: monospace;
  font-size: 12px;
  padding: 2px 6px;
  border-radius: 3px;
  flex: 1;
}

.env-value.available {
  background-color: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.env-value.unavailable {
  background-color: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.env-success-message {
  padding: 10px;
  background-color: #d4edda;
  color: #155724;
  border-radius: 4px;
  border: 1px solid #c3e6cb;
  text-align: center;
  font-weight: 500;
}

.env-error-message {
  padding: 10px;
  background-color: #f8d7da;
  color: #721c24;
  border-radius: 4px;
  border: 1px solid #f5c6cb;
  text-align: center;
  font-weight: 500;
  margin-bottom: 15px;
}

.actions {
  justify-content: space-between !important;
  border-top: 1px solid var(--color-divider)!important;
}

.install-btn {
  padding: 6px 12px;
  background-color: var(--primary-color, #4a6cf7);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 500;
  font-size: 13px;
  transition: all 0.2s ease;
}

.install-btn:hover:not(:disabled) {
  background-color: var(--primary-hover, #3b5ae0);
}

.install-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.install-note {
  font-size: 12px;
  color: var(--text-muted, #666);
  font-style: italic;
  margin-top: 8px;
}

.overall-status {
  margin-top: 20px;
  padding: 15px;
  border-radius: 8px;
  text-align: center;
  font-weight: 500;
}

.status-success {
  background-color: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.status-warning {
  background-color: #fff3cd;
  color: #856404;
  border: 1px solid #ffeaa7;
}


.install-buttons {
  display: flex;
  gap: 10px;
}

.btn-primary,
.btn-secondary,
.btn-outline {
  padding: 10px 20px;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
  font-size: 14px;
  transition: all 0.2s ease;
  border: none;
}

.btn-primary {
  background-color: var(--primary-color, #4a6cf7);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background-color: var(--primary-hover, #3b5ae0);
}

.btn-secondary {
  background-color: var(--secondary-color, #6c757d);
  color: white;
}

.btn-secondary:hover:not(:disabled) {
  background-color: var(--secondary-hover, #5a6268);
}

.btn-outline {
  background-color: transparent;
  color: var(--vfm-text, #333);
  border: 1px solid var(--border-color, #e9ecef);
}

.btn-outline:hover:not(:disabled) {
  background-color: var(--light-bg, #f8f9fa);
}

.btn-primary:disabled,
.btn-secondary:disabled,
.btn-outline:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

@media (max-width: 768px) {
  .setup-env-modal {
    width: 95%;
    margin: 10px;
  }
  
  .env-status-grid {
    grid-template-columns: 1fr;
  }
  
  .footer-actions {
    flex-direction: column;
    gap: 10px;
  }
  
  .install-buttons {
    width: 100%;
    justify-content: center;
  }
}
</style>