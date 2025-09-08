<script setup lang="ts">
import { ref, computed, reactive } from 'vue'
import { request, alert } from '@/support'
import { VueFinalModal } from 'vue-final-modal'
import FormModel from '@/widgets/FormModel.vue'
import { doInstall } from '@/logics/mcp'
import { checkMcpEnv } from '@/logics/mcp'
import { checkNetEnv } from '@/logics/mcp'

// Types for better type safety
type WizardStep = 1 | 2 | 3 | 4
type WizardMode = 'trial' | 'apikey' | ''

interface WizardState {
  currentStep: WizardStep
  selectedMode: WizardMode
  waitingForAuth: boolean
  apiConfigured: boolean
  envConfigured: boolean
  pyInstalling: boolean
  selectedTask: string
  loading: boolean
  error: string
  login: any 
}

const props = defineProps({
  gateway: {
    type: String,
    default: 'https://auth.swiflow.com'
  },
  initialState: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['submit', 'cancel'])

// Centralized state management
const totalSteps = 4
const defaultState: WizardState = {
  currentStep: 1 as WizardStep,
  selectedMode: 'apikey',
  waitingForAuth: false,
  apiConfigured: false,
  envConfigured: false,
  pyInstalling: false,
  selectedTask: '',
  loading: false,
  error: '',
  login: null
}

// Merge default state with initial state from props
const state = reactive<WizardState>({
  ...defaultState,
  ...props.initialState
})

// API Key configuration state
const modelForm = ref<typeof FormModel>()
const modelConfig = ref<ModelMeta>()
const models = ref<ModelResp>({})

// Python environment state
const mcpEnv = ref({
  python: '', uvx: '',
  nodejs: '', npx: '',
  windows: false,
})

// Sample tasks configuration
const sampleTasks = [
  {
    id: 'web-scraping',
    title: '网页数据抓取',
    brief: '抓取指定网站的数据并进行分析',
    icon: '🕷️'
  },
  {
    id: 'code-review',
    title: '代码审查助手',
    brief: '分析代码质量，提供改进建议',
    icon: '🔍'
  },
  {
    id: 'data-analysis',
    title: '数据分析报告',
    brief: '从CSV文件生成数据分析报告',
    icon: '📊'
  },
  {
    id: 'api-testing',
    title: 'API接口测试',
    brief: '自动化测试REST API接口',
    icon: '🧪'
  }
]

// Navigation methods with improved logic
const nextStep = () => {
  if (state.currentStep < totalSteps) {
    // Clear any previous errors
    state.error = ''
    
    // Always go to next step normally
    state.currentStep++
  }
}

const goToStep = (step: WizardStep) => {
  if (step <= state.currentStep) {
    state.error = ''
    state.currentStep = step
  }
}

// Step validation with better logic
const canProceed = computed(() => {
  switch (state.currentStep) {
    case 1: return state.selectedMode !== '' // Mode selected
    case 2: 
      if (state.selectedMode === 'apikey') {
        return state.apiConfigured
      } else if (state.selectedMode === 'trial') {
        return true // Trial mode can always proceed
      }
      return false
    case 3: return state.envConfigured // Python environment ready
    case 4: return state.selectedTask !== '' // Task selected
    default: return false
  }
})

// API Key configuration with better error handling
const loadModelConfig = async () => {
  try {
    state.loading = true
    state.error = ''
    
    const url = `/setting?act=get-model`
    const resp = await request.get(url) as any
    models.value = resp.models || {}
    
    if (resp && resp.useModel) {
      modelConfig.value = resp.useModel as ModelMeta
      state.apiConfigured = !!(resp.useModel.apiKey)
    }
  } catch (err) {
    console.error('Failed to load model config:', err)
    state.error = 'Failed to load model configuration'
  } finally {
    state.loading = false
    if (!modelConfig.value || !modelConfig.value.provider) {
      modelConfig.value = {provider: 'doubao'} as ModelMeta
    }
  }
}

const saveApiConfig = async () => {
  const data = modelForm.value?.getFormModel()
  if (!data) {
    state.error = 'Please fill in all required fields'
    return
  }
  
  try {
    state.loading = true
    state.error = ''
    
    const url = `/setting?act=set-model`
    const resp = await request.post(url, data)
    const errmsg = (resp as any)?.errmsg
    
    if (errmsg && errmsg !== 'success') {
      state.error = errmsg
      return
    }
    
    state.apiConfigured = true
    // alert('API configuration saved successfully!')
    // Auto proceed to next step after successful save
    nextStep()
  } catch (err) {
    state.error = 'Failed to save API configuration'
    console.error('API config save error:', err)
  } finally {
    state.loading = false
  }
}

// Python environment setup with better state management
const checkPythonEnv = async () => {
  try {
    state.loading = true
    state.error = ''
    checkMcpEnv((info) => {
      mcpEnv.value = info
      state.envConfigured = !!(info.python && info.uvx)
    })
  } catch (err) {
    state.error = 'Failed to check Python environment'
    console.error('Python env check error:', err)
  } finally {
    state.loading = false
  }
}

const keepCheckPythonEnv = () => {
  // Start periodic checking every 3 seconds until installation is complete
  const checkInterval = setInterval(async () => {
    try {
      // Check environment and wait for state update
      await checkPythonEnv()
      // Check if environment is configured after the async call
      if (state.envConfigured) {
        state.pyInstalling = false
        clearInterval(checkInterval)
        localStorage.removeItem('welcome')
        alert('Python environment installed successfully!')
      }
    } catch (err) {
      console.error('Error checking Python environment:', err)
    }
  }, 3000)
  
  // Set a maximum timeout of 5 minutes to prevent infinite checking
  setTimeout(() => {
    if (state.pyInstalling) {
      clearInterval(checkInterval)
      state.pyInstalling = false
      state.error = 'Installation timeout - please try again'
    }
  }, 300000) // 5 minutes
}

const installPython = async () => {
  state.pyInstalling = true
  state.error = ''
  
  try {
    const netEnv = await checkNetEnv()
    // Start installation process
    localStorage.setItem('welcome', 'python-install')
    await doInstall(netEnv, 'uvx-py', (success) => {
      if (!success) {
        state.pyInstalling = false
        state.error = 'Failed to install Python environment'
      }
    })
    keepCheckPythonEnv()
  } catch (err) {
    state.pyInstalling = false
    state.error = 'Installation failed'
    console.error('Python install error:', err)
  }
}

// Mode selection methods with better state management
const selectMode = (mode: WizardMode) => {
  state.selectedMode = mode
  state.error = ''
  
  if (mode === 'trial') {
    // For trial mode, reset auth waiting state
    state.waitingForAuth = false
  }
}

const gotoSignUp = async () => {
  try {
    state.waitingForAuth = true // Show waiting screen
    const path = 'authorization?from=swiflow-app'
    const signup = document.getElementById('signupUrl')
    signup?.setAttribute('href', `${props.gateway}/${path}`)
    const result = signup && signup.click && signup.click()

    // Start checking localStorage for login info every 300ms
    const checkLoginInterval = setInterval(() => {
      try {
        const loginInfo = localStorage.getItem('login')
        if (loginInfo) {
          console.log('Login info detected:', loginInfo)
          clearInterval(checkLoginInterval)
          state.waitingForAuth = false
          // Store login info in state for display
          try {
            state.login = JSON.parse(loginInfo)
          } catch {
            state.login = { token: loginInfo } // If not JSON, treat as token
          }
        }
      } catch (error) {
        console.error('Error checking localStorage:', error)
      }
    }, 300)
    
    // Clear interval after 5 minutes to prevent infinite checking
    setTimeout(() => {
      clearInterval(checkLoginInterval)
    }, 300000) // 5 minutes timeout
    
    return result
  } catch (err) {
    state.error = 'Failed to open signup page'
    console.error('Signup error:', err)
  }
}

// Task selection with validation
const selectTask = (taskId: string) => {
  state.selectedTask = taskId
  state.error = ''
}

// Wrapper functions for click handlers to fix TypeScript type issues
const handleCheckPythonEnv = () => {
  checkPythonEnv()
}

const handleInstallPython = () => {
  installPython()
}

// Complete setup with validation
const completeSetup = () => {
  if (!state.selectedTask) {
    state.error = 'Please select a task to continue'
    return
  }
  
  emit('submit', {
    selectedTask: state.selectedTask
  })
}

const onCancel = () => {
  emit('cancel')
}

// Initialize on mount with better error handling
const initializeWelcome = async () => {
  try {
    await Promise.all([
      loadModelConfig(),
      checkPythonEnv()
    ])
  } catch (err) {
    state.error = 'Failed to initialize welcome wizard'
    console.error('Initialization error:', err)
  }
}
</script>

<template>
  <VueFinalModal @opened="initializeWelcome" :click-to-close="false" :esc-to-close="false" class="swiflow-modal-wrapper"
    content-class="welcome-modal">
    <div class="welcome-container">
      <!-- Header with progress indicator -->
      <div class="welcome-header">
        <!-- <h2>欢迎使用 Swiflow</h2> -->
        <div class="progress-indicator">
          <div v-for="step in totalSteps" :key="step" class="progress-step" :class="{ 
              'active': step === state.currentStep, 
              'completed': step < state.currentStep,
              'clickable': step <= state.currentStep
            }" @click="step <= state.currentStep && goToStep(step as WizardStep)">
            <span class="step-number">{{ step }}</span>
          </div>
        </div>
      </div>

      <!-- Error Display -->
      <div v-if="state.error" class="error-message">
        <span class="error-icon">⚠️</span>
        <span>{{ state.error }}</span>
      </div>

      <!-- Loading Indicator -->
      <div v-if="state.loading" class="loading-indicator">
        <div class="loading-spinner">⏳</div>
        <span>Loading...</span>
      </div>

      <!-- Step Content -->
      <div class="welcome-content">
        <!-- Step 1: Introduction -->
        <div v-if="state.currentStep === 1" class="step-content">
          <h3>欢迎使用 Swiflow AI 工作流平台</h3>
          <div class="intro-content display-block">
            <!-- Feature introduction -->
            <div class="feature-grid">
              <div class="feature-item">
                <span class="feature-icon">🤖</span>
                <div class="feature-text">
                  <h4>智能AI助手</h4>
                  <p>支持多种AI模型，提供强大的对话和任务处理能力</p>
                </div>
              </div>
              <div class="feature-item">
                <span class="feature-icon">🔧</span>
                <div class="feature-text">
                  <h4>丰富工具集成</h4>
                  <p>集成Python、Node.js等开发工具，支持MCP协议扩展</p>
                </div>
              </div>
              <div class="feature-item">
                <span class="feature-icon">⚡</span>
                <div class="feature-text">
                  <h4>自动化工作流</h4>
                  <p>创建和执行复杂的自动化任务，提升工作效率</p>
                </div>
              </div>
              <div class="feature-item">
                <span class="feature-icon">📊</span>
                <div class="feature-text">
                  <h4>数据分析处理</h4>
                  <p>强大的数据处理和分析能力，支持多种数据格式</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Step 2: Configuration -->
        <div v-if="state.currentStep === 2" class="step-content">
          <!-- Trial mode: Waiting for authentication -->
          <div v-if="state.selectedMode === 'trial'" class="trial-mode-content">
            <div v-if="state.waitingForAuth && !state.login" class="waiting-content">
              <div class="waiting-message">
                <div class="loading-spinner">⏳</div>
                <h4>等待认证中...</h4>
                <p>请在新打开的页面中完成注册和认证流程</p>
                <p class="waiting-tip">认证完成后，请返回此页面继续配置</p>
              </div>
            </div>
            <div v-else-if="state.login" class="login-success-content">
              <h3>登录成功</h3>
              <p class="step-description">欢迎回来！您已成功登录体验模式</p>
              <div class="login-info-display">
                <div class="login-info-item">
                  <span class="info-icon">👤</span>
                  <div class="info-content">
                    <span class="info-label">用户信息:</span>
                    <span class="info-value">{{ state.login.username || state.login.email || '体验用户' }}</span>
                  </div>
                </div>
                <div class="login-info-item" v-if="state.login.email">
                  <span class="info-icon">📧</span>
                  <div class="info-content">
                    <span class="info-label">邮箱:</span>
                    <span class="info-value">{{ state.login.email }}</span>
                  </div>
                </div>
                <div class="login-info-item">
                  <span class="info-icon">✅</span>
                  <div class="info-content">
                    <span class="info-label">状态:</span>
                    <span class="info-value">已认证</span>
                  </div>
                </div>
              </div>
            </div>
            <div v-else class="trial-info">
              <h3>注册体验模式</h3>
              <p class="step-description">点击下方按钮开始注册体验，无需配置API密钥</p>
              <div class="trial-features">
                <div class="feature-item">
                  <span class="feature-icon">🚀</span>
                  <span>免费体验完整功能</span>
                </div>
                <div class="feature-item">
                  <span class="feature-icon">⚡</span>
                  <span>快速上手使用</span>
                </div>
                <div class="feature-item">
                  <span class="feature-icon">🎯</span>
                  <span>无需准备API密钥</span>
                </div>
              </div>
            </div>
          </div>

          <!-- API Key mode: Configuration -->
          <div v-if="state.selectedMode === 'apikey'" class="api-config-content">
            <h3>配置 AI 模型</h3>
            <p class="step-description">请配置您的AI模型提供商和API密钥以开始使用</p>
            <FormModel v-if="modelConfig" :config="modelConfig" :models="models" ref="modelForm" />
          </div>
        </div>

        <!-- Step 3: Python Environment -->
        <div v-if="state.currentStep === 3" class="step-content">
          <h3>配置 Python 环境</h3>
          <p class="step-description">Python环境用于执行代码分析、数据处理等高级功能</p>
          <div class="env-config-content">
            <div class="env-status">
              <div class="env-item">
                <span class="env-label">Python:</span>
                <span class="env-value" :class="mcpEnv.python ? 'available' : 'unavailable'">
                  {{ mcpEnv.python || '未安装' }}
                </span>
              </div>
              <div class="env-item">
                <span class="env-label">UVX:</span>
                <span class="env-value" :class="mcpEnv.uvx ? 'available' : 'unavailable'">
                  {{ mcpEnv.uvx || '未安装' }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- Step 4: Sample Tasks -->
        <div v-if="state.currentStep === 4" class="step-content">
          <h3>选择示例任务</h3>
          <p class="step-description">选择一个示例任务来开始您的 Swiflow 之旅</p>
          <div class="tasks-grid">
            <div v-for="task in sampleTasks" :key="task.id" class="task-card" @click="selectTask(task.id)"
              :class="{ 'selected': state.selectedTask === task.id }">
              <div class="task-icon">{{ task.icon }}</div>
              <h4 class="task-title">{{ task.title }}</h4>
              <p class="task-brief">{{ task.brief }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Footer with navigation -->
      <div class="welcome-footer">
        <div class="footer-actions">
          <!-- Always show skip guide button on the left -->
          <button class="btn-outline" @click="onCancel">
            跳过引导
          </button>

          <!-- Action buttons on the right -->
          <div class="action-buttons">
            <!-- Step 1: Mode selection buttons -->
            <button :disabled="state.loading" v-if="(state.currentStep === 1)" class="btn-primary" @click="nextStep">
              下一步
            </button>

            <!-- Step 2: Mode-specific buttons -->
            <template v-if="state.currentStep === 2">
              <!-- Mode switch button -->
              <button class="btn-outline" @click="selectMode(state.selectedMode === 'trial' ? 'apikey' : 'trial')"
                :disabled="state.loading">
                {{ state.selectedMode === 'trial' ? '设置API Key' : '我没有API Key' }}
              </button>

              <!-- Trial mode buttons -->
              <button v-if="state.selectedMode === 'trial' && !state.waitingForAuth && !state.login" 
                class="btn-primary" @click="gotoSignUp" :disabled="state.loading" >
                注册体验
                <a target="_blank" id="signupUrl" style="display: none;" />
              </button>

              <!-- Continue button for trial mode when waiting for auth or login completed -->
              <button v-if="state.selectedMode === 'trial' && (state.waitingForAuth || state.login)" 
                @click="nextStep" class="btn-primary" :disabled="state.loading">
                下一步
              </button>

              <!-- API Key mode buttons -->
              <button v-if="state.selectedMode === 'apikey'" class="btn-primary" 
                @click="saveApiConfig" :disabled="!modelForm || state.loading">
                {{ state.loading ? '保存中...' : '保存配置' }}
              </button>
            </template>

            <!-- Step 3: Environment Configuration buttons -->
            <template v-if="state.currentStep === 3 && !state.envConfigured">
              <button class="btn-outline" @click="handleCheckPythonEnv" 
                :disabled="state.loading">
                {{ state.loading ? '检测中...' : '重新检测' }}
              </button>
              <button class="btn-primary" @click="handleInstallPython" 
                :disabled="state.pyInstalling || state.loading">
                {{ state.pyInstalling ? '安装中...' : '安装 Python 环境' }}
              </button>
            </template>
            <template v-if="state.currentStep === 3 && state.envConfigured">
              <button class="btn-primary" @click="nextStep" :disabled="!canProceed || state.loading">
                下一步
              </button>
            </template>

            <button v-if="state.currentStep === totalSteps" :disabled="!canProceed || state.loading" 
              class="btn-primary" @click="completeSetup">
              开始使用
            </button>
          </div>
        </div>
      </div>
    </div>
  </VueFinalModal>
</template>

<style scoped>
/* CSS Variables for consistent theming */
:global(.welcome-modal) {
  --primary-color: #4a6cf7;
  --success-color: #28a745;
  --danger-color: #dc3545;
  --secondary-color: #6c757d;
  --light-bg: #f8f9fa;
  --border-color: #e9ecef;
  --text-muted: #6c757d;
  --border-radius: 8px;
  --transition: all 0.3s ease;
  
  display: flex;
  flex-direction: column;
  padding: 30px;
  border-radius: 12px;
  background: var(--bg-main, #fff);
  color: var(--vfm-text, #000);
  max-width: 700px;
  width: 90%;
  max-height: 80vh;
  height: 540px;
  overflow: hidden;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
  position: relative;
  z-index: 1001;
}

/* Layout Components */
.welcome-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 25px;
}

.welcome-header {
  text-align: center;
}

.welcome-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  padding: 0 5px;
  min-height: 400px;
  max-height: 400px;
}

.welcome-footer {
  border-top: 1px solid var(--border-color);
  padding-top: 15px;
  flex-shrink: 0;
  margin-top: auto;
}

/* Progress Indicator */
.progress-indicator {
  display: flex;
  justify-content: center;
  gap: 15px;
  margin-bottom: 10px;
}

.progress-step {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--border-color);
  color: var(--text-muted);
  font-weight: 600;
  transition: var(--transition);
  position: relative;
  cursor: pointer;
}

.progress-step.active {
  background-color: var(--primary-color);
  color: white;
  transform: scale(1.1);
}

.progress-step.completed {
  background-color: var(--success-color);
  color: white;
}

.progress-step:not(:last-child)::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 100%;
  width: 15px;
  height: 2px;
  background-color: var(--border-color);
  transform: translateY(-50%);
}

.progress-step.completed:not(:last-child)::after {
  background-color: var(--success-color);
}

/* Common Components */
.step-content {
  text-align: center;
  animation: fadeIn 0.3s ease-in;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.step-content h3 {
  font-size: 1.4rem;
  font-weight: 600;
  margin: 0 0 8px 0;
  color: var(--vfm-text, #000);
}

.step-description {
  font-size: 1rem;
  color: var(--text-muted);
  margin-bottom: 25px;
  line-height: 1.5;
}

/* Status Messages */
.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background-color: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
  border-radius: var(--border-radius);
  margin-bottom: 15px;
  font-size: 0.9rem;
}

.loading-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 20px;
  color: var(--text-muted);
}

.loading-spinner {
  font-size: 1.5rem;
  animation: pulse 2s infinite;
}

/* Grid Layouts */
.feature-grid,
.tasks-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 15px;
}

.feature-grid {
  margin-bottom: 30px;
}

.tasks-grid {
  margin-top: 15px;
  max-height: 280px;
}

/* Card Components */
.feature-item,
.task-card {
  padding: 15px;
  border-radius: var(--border-radius);
  background-color: var(--light-bg);
  transition: var(--transition);
}

.feature-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.feature-item:hover {
  transform: translateY(-2px);
}

.task-card {
  border: 2px solid var(--border-color);
  cursor: pointer;
  text-align: center;
  min-height: 100px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.task-card:hover {
  border-color: var(--primary-color);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(74, 108, 247, 0.15);
}

.task-card.selected {
  border-color: var(--primary-color);
  background-color: #f0f4ff;
}

/* Content Sections */
.intro-content {
  text-align: left;
}

.trial-mode-content,
.api-config-content,
.login-success-content {
  text-align: center;
}

.api-config-content {
  margin: 0 auto;
  max-width: fit-content;
}

.env-config-content {
  text-align: left;
}

/* Status Displays */
.env-status,
.login-info-display,
.trial-features {
  background-color: var(--light-bg);
  border-radius: 12px;
  padding: 20px;
  margin: 20px auto;
  max-width: 75%;
}

.env-item,
.login-info-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--border-color);
}

.env-item:last-child,
.login-info-item:last-child {
  border-bottom: none;
}

.env-value {
  font-family: monospace;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.9rem;
}

.env-value.available {
  background-color: #d4edda;
  color: #155724;
}

.env-value.unavailable {
  background-color: #f8d7da;
  color: #721c24;
}

/* Button Styles */
.footer-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 10px;
}

.btn-primary,
.btn-secondary,
.btn-outline {
  padding: 10px 20px;
  border-radius: var(--border-radius);
  cursor: pointer;
  font-weight: 500;
  font-size: 0.95rem;
  transition: var(--transition);
  border: none;
}

.btn-primary {
  background-color: var(--primary-color);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background-color: #3a5ce5;
  transform: translateY(-1px);
}

.btn-outline {
  background-color: transparent;
  color: var(--text-muted);
  border: 1px solid var(--text-muted);
}

.btn-outline:hover {
  background-color: var(--text-muted);
  color: white;
  transform: translateY(-1px);
}

.btn-outline.selected {
  background-color: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

/* Waiting and Loading States */
.waiting-content {
  text-align: center;
  justify-content: center;
  align-items: center;
}

.waiting-message {
  padding: 40px 20px;
}

.waiting-message .loading-spinner {
  font-size: 3rem;
  margin-bottom: 20px;
}

.waiting-tip {
  font-style: italic;
  color: var(--primary-color) !important;
}

/* Animations */
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* Dark Theme */
:root[data-theme="dark"] .feature-item,
:root[data-theme="dark"] .env-status,
:root[data-theme="dark"] .login-info-display,
:root[data-theme="dark"] .trial-features {
  background-color: #2a2a2a;
}

:root[data-theme="dark"] .task-card.selected {
  background-color: #1a2332;
}

:root[data-theme="dark"] .error-message {
  background-color: #2d1b1f;
  color: #f8d7da;
  border-color: #842029;
}

:root[data-theme="dark"] .login-info-item {
  border-bottom-color: #404040;
}

/* Responsive Design */
@media (max-width: 600px) {
  :global(.welcome-modal) {
    max-width: 95%;
    padding: 20px;
  }
  
  .feature-grid,
  .tasks-grid {
    grid-template-columns: 1fr;
  }
  
  .footer-actions {
    flex-direction: column;
    gap: 15px;
  }
  
  .footer-actions > * {
    width: 100%;
  }
}
</style>