<script setup lang="ts">
import { ref, computed } from 'vue'
import { request, alert } from '@/support'
import { VueFinalModal } from 'vue-final-modal'
import FormModel from '@/widgets/FormModel.vue'
import { checkMcpEnv, doInstall, checkNetEnv } from '@/logics/mcp'

const emit = defineEmits(['submit', 'cancel'])

// Current step state (1-4)
const totalSteps = 4
const currentStep = ref(1)

// API Key configuration state
const modelForm = ref<typeof FormModel>()
const modelConfig = ref<ModelMeta>()
const models = ref<ModelResp>({})
const apiConfigured = ref(false)

// Python environment state
const mcpEnv = ref({
  python: '', uvx: '',
  nodejs: '', npx: '',
  windows: false,
})
const pyInstalling = ref(false)
const envConfigured = ref(false)

// Sample tasks
const sampleTasks = [
  {
    id: 'web-scraping',
    title: '网页数据抓取',
    description: '抓取指定网站的数据并进行分析',
    icon: '🕷️'
  },
  {
    id: 'code-review',
    title: '代码审查助手',
    description: '分析代码质量，提供改进建议',
    icon: '🔍'
  },
  {
    id: 'data-analysis',
    title: '数据分析报告',
    description: '从CSV文件生成数据分析报告',
    icon: '📊'
  },
  {
    id: 'api-testing',
    title: 'API接口测试',
    description: '自动化测试REST API接口',
    icon: '🧪'
  }
]
const selectedTask = ref('')

// Navigation methods
const nextStep = () => {
  if (currentStep.value < totalSteps) {
    currentStep.value++
  }
}

const prevStep = () => {
  if (currentStep.value > 1) {
    currentStep.value--
  }
}

const goToStep = (step: number) => {
  currentStep.value = step
}

// Step validation
const canProceed = computed(() => {
  switch (currentStep.value) {
    case 1: return true // Introduction step
    case 2: return apiConfigured.value // API Key configured
    case 3: return envConfigured.value // Python environment ready
    case 4: return selectedTask.value !== '' // Task selected
    default: return false
  }
})

// API Key configuration
const loadModelConfig = async () => {
  try {
    const url = `/setting?act=get-model`
    const resp = await request.get(url) as any
    models.value = resp.models || {}
    if (resp && resp.useModel) {
      modelConfig.value = resp.useModel as ModelMeta
      apiConfigured.value = !!(resp.useModel.apiKey)
    }
  } catch (err) {
    console.error('Failed to load model config:', err)
  } finally {
    if (!modelConfig.value || !modelConfig.value.provider) {
      modelConfig.value = {provider: 'doubao'} as ModelMeta
    }
  }
}

const saveApiConfig = async () => {
  const data = modelForm.value?.getFormModel()
  if (!data) {
    alert('Please fill in all required fields')
    return
  }
  
  try {
    const url = `/setting?act=set-model`
    const resp = await request.post(url, data)
    const errmsg = (resp as any)?.errmsg
    if (errmsg && errmsg !== 'success') {
      alert(errmsg)
      return
    }
    apiConfigured.value = true
    alert('API configuration saved successfully!')
    // Auto proceed to next step after successful save
    nextStep()
  } catch (err) {
    alert('Failed to save API configuration')
  }
}

// Python environment setup
const checkPythonEnv = async () => {
  checkMcpEnv((info) => {
    mcpEnv.value = info
    const wasConfigured = envConfigured.value
    envConfigured.value = !!(info.python && info.uvx)
    // Auto proceed to next step if environment becomes ready
    if (!wasConfigured && envConfigured.value) {
      nextStep()
    }
  })
}

const installPython = async () => {
  pyInstalling.value = true
  try {
    const netEnv = await checkNetEnv()
    await doInstall(netEnv, 'uvx-py', (success) => {
      pyInstalling.value = false
      if (success) {
        alert('Python environment installed successfully!')
        checkPythonEnv() // Recheck environment
        // Auto proceed to next step after successful installation
        nextStep()
      } else {
        alert('Failed to install Python environment')
      }
    })
  } catch (err) {
    pyInstalling.value = false
    alert('Installation failed')
  }
}

// Task selection
const selectTask = (taskId: string) => {
  selectedTask.value = taskId
}

// Complete setup
const completeSetup = () => {
  emit('submit', {
    selectedTask: selectedTask.value
  })
}

const onCancel = () => {
  emit('cancel')
}

// Initialize on mount
const initializeWelcome = async () => {
  await loadModelConfig()
  await checkPythonEnv()
}
</script>

<template>
  <VueFinalModal
    class="swiflow-modal-wrapper"
    content-class="welcome-modal"
    :click-to-close="false"
    :esc-to-close="false"
    @opened="initializeWelcome"
  >
    <div class="welcome-container">
      <!-- Header with progress indicator -->
      <div class="welcome-header">
        <!-- <h2>欢迎使用 Swiflow</h2> -->
        <div class="progress-indicator">
          <div 
            v-for="step in totalSteps" 
            :key="step" class="progress-step"
            :class="{ 
              'active': step === currentStep, 
              'completed': step < currentStep,
              'clickable': step <= currentStep
            }"
            @click="step <= currentStep && goToStep(step)"
          >
            <span class="step-number">{{ step }}</span>
          </div>
        </div>
      </div>

      <!-- Step Content -->
      <div class="welcome-content">
        <!-- Step 1: Introduction -->
        <div v-if="currentStep === 1" class="step-content">
          <h3>欢迎使用 Swiflow AI 工作流平台</h3>
          <div class="intro-content display-block">
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

        <!-- Step 2: API Key Configuration -->
        <div v-if="currentStep === 2" class="step-content">
          <h3>配置 AI 模型</h3>
          <div class="api-config-content">
            <p class="step-description">请配置您的AI模型提供商和API密钥以开始使用</p>
            <FormModel 
              v-if="modelConfig" 
              class="display-block"
              :config="modelConfig" 
              :models="models" 
              ref="modelForm"
            />

          </div>
        </div>

        <!-- Step 3: Python Environment -->
        <div v-if="currentStep === 3" class="step-content">
          <h3>配置 Python 环境</h3>
          <div class="config-status" v-if="envConfigured">
            <span class="status-badge success">✓ 环境就绪</span>
          </div>
          <div class="env-config-content">
            <p class="step-description">Python环境用于执行代码分析、数据处理等高级功能</p>
            
            <div class="display-block">
              <div class="env-status ">
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
        </div>

        <!-- Step 4: Sample Tasks -->
        <div v-if="currentStep === 4" class="step-content">
          <h3>选择示例任务</h3>
          <p class="step-description">选择一个示例任务来开始您的 Swiflow 之旅</p>
          <div class="tasks-grid display-block">
            <div 
              v-for="task in sampleTasks" 
              :key="task.id"
              class="task-card"
              :class="{ 'selected': selectedTask === task.id }"
              @click="selectTask(task.id)"
            >
              <div class="task-icon">{{ task.icon }}</div>
              <h4 class="task-title">{{ task.title }}</h4>
              <p class="task-description">{{ task.description }}</p>
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
            <!-- Step 2: API Configuration buttons -->
            <button 
              v-if="currentStep === 2" 
              class="btn-secondary" 
              @click="saveApiConfig"
              :disabled="!modelForm"
            >
              保存配置
            </button>
            
            <!-- Step 3: Environment Configuration buttons -->
            <template v-if="currentStep === 3 && !envConfigured">
              <button 
                class="btn-secondary" 
                @click="installPython"
                :disabled="pyInstalling"
              >
                {{ pyInstalling ? '安装中...' : '安装 Python 环境' }}
              </button>
              <button class="btn-outline" @click="checkPythonEnv">
                重新检测
              </button>
            </template>
            
            <!-- Navigation buttons -->
            <!-- Show "Next Step" for steps without specific actions (Step 1 and Step 4) -->
            <!-- Also show for Step 3 when environment is already configured -->
            <button 
              v-if="((currentStep === 1 || currentStep === 4) || (currentStep === 3 && envConfigured)) && currentStep < totalSteps" 
              class="btn-primary" 
              @click="nextStep"
              :disabled="!canProceed"
            >
              下一步
            </button>
            
            <!-- For Step 2: Save Config acts as next step -->
            <!-- For Step 3: Install/Recheck acts as next step when env not configured -->
            <!-- These are already handled above in step-specific buttons -->
            
            <button 
              v-if="currentStep === totalSteps" 
              class="btn-primary" 
              @click="completeSetup"
              :disabled="!canProceed"
            >
              开始使用
            </button>
          </div>
        </div>
      </div>
    </div>
  </VueFinalModal>
</template>

<style scoped>
:global(.welcome-modal) {
  display: flex;
  flex-direction: column;
  padding: 30px;
  border-radius: 12px;
  background: var(--bg-main, #fff);
  color: var(--vfm-text, #000);
  max-width: 700px;
  width: 90%;
  max-height: 80vh;
  min-height: 50vh;
  overflow: hidden;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
  position: relative;
  z-index: 1001;
}

.welcome-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 25px;
}

/* Header Styles */
.welcome-header {
  text-align: center;
}

.welcome-header h2 {
  font-size: 1.8rem;
  font-weight: bold;
  margin: 0 0 20px 0;
  color: var(--vfm-text, #000);
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
  background-color: #e9ecef;
  color: #6c757d;
  font-weight: 600;
  transition: all 0.3s ease;
  position: relative;
}

.progress-step.clickable {
  cursor: pointer;
}

.progress-step.active {
  background-color: #4a6cf7;
  color: white;
  transform: scale(1.1);
}

.progress-step.completed {
  background-color: #28a745;
  color: white;
}

.progress-step:not(:last-child)::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 100%;
  width: 15px;
  height: 2px;
  background-color: #e9ecef;
  transform: translateY(-50%);
}

.progress-step.completed:not(:last-child)::after {
  background-color: #28a745;
}

/* Content Styles */
.welcome-content {
  flex: 1;
  overflow-y: auto;
  padding: 0 5px;
}

.step-content {
  text-align: center;
  animation: fadeIn 0.3s ease-in;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.step-content h3 {
  font-size: 1.4rem;
  font-weight: 600;
  margin: 0 0 15px 0;
  color: var(--vfm-text, #000);
}

.step-description {
  font-size: 1rem;
  color: #6c757d;
  margin-bottom: 20px;
  line-height: 1.5;
}

.display-block {
  display: flex;
  min-height: 285px;
  align-items: stretch;
  flex-direction: column;
}

/* Step 1: Introduction */
.intro-content {
  text-align: left;
}

.feature-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 20px;
}

.feature-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 15px;
  border-radius: 8px;
  background-color: #f8f9fa;
  transition: transform 0.2s ease;
}

.feature-item:hover {
  transform: translateY(-2px);
}

.feature-icon {
  font-size: 1.5rem;
  flex-shrink: 0;
}

.feature-text h4 {
  margin: 0 0 5px 0;
  font-size: 1rem;
  font-weight: 600;
}

.feature-text p {
  margin: 0;
  font-size: 0.9rem;
  color: #6c757d;
  line-height: 1.4;
}

.epigraph {
  font-style: italic;
  text-align: center;
  padding: 15px;
  background-color: #f5f5f5;
  border-radius: 8px;
  margin: 20px 0 0 0;
}

/* Step 2: API Configuration */
.config-status {
  margin-bottom: 15px;
}

.status-badge {
  display: inline-block;
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 0.9rem;
  font-weight: 500;
}

.status-badge.success {
  background-color: #d4edda;
  color: #155724;
}

.api-config-content {
  margin: 0 auto;
  text-align: left;
  max-width: fit-content;
}

.api-config-content .step-description {
  text-align: center;
  margin-bottom: 20px;
}

/* Center the FormModel component */
.api-config-content :deep(.form-container) {
  text-align: left;
  display: inline-block;
  width: 100%;
  max-width: 400px;
}

.config-actions {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

/* Step 3: Environment Configuration */
.env-config-content {
  text-align: left;
}

.env-status {
  background-color: #f8f9fa;
  border-radius: 8px;
  padding: 20px;
  margin: 20px 0;
}

.env-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #e9ecef;
}

.env-item:last-child {
  border-bottom: none;
}

.env-label {
  font-weight: 600;
  color: var(--vfm-text, #000);
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

.env-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-top: 20px;
}

/* Step 4: Task Selection */
.tasks-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 15px;
  margin-top: 20px;
}

.task-card {
  padding: 20px;
  border: 2px solid #e9ecef;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.3s ease;
  text-align: center;
}

.task-card:hover {
  border-color: #4a6cf7;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(74, 108, 247, 0.15);
}

.task-card.selected {
  border-color: #4a6cf7;
  background-color: #f0f4ff;
}

.task-icon {
  font-size: 2rem;
  margin-bottom: 10px;
}

.task-title {
  font-size: 1rem;
  font-weight: 600;
  margin: 0 0 8px 0;
  color: var(--vfm-text, #000);
}

.task-description {
  font-size: 0.9rem;
  color: #6c757d;
  margin: 0;
  line-height: 1.4;
}

/* Footer Styles */
.welcome-footer {
  border-top: 1px solid #e9ecef;
  padding-top: 20px;
}

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

/* Button Styles */
.btn-primary {
  background-color: #4a6cf7;
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
  font-size: 0.95rem;
  transition: all 0.2s ease;
}

.btn-primary:hover:not(:disabled) {
  background-color: #3a5ce5;
  transform: translateY(-1px);
}

.btn-primary:disabled {
  background-color: #6c757d;
  cursor: not-allowed;
  transform: none;
}

.btn-secondary {
  background-color: #6c757d;
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
  font-size: 0.95rem;
  transition: all 0.2s ease;
}

.btn-secondary:hover:not(:disabled) {
  background-color: #5a6268;
  transform: translateY(-1px);
}

.btn-secondary:disabled {
  background-color: #adb5bd;
  cursor: not-allowed;
  transform: none;
}

.btn-outline {
  background-color: transparent;
  color: #6c757d;
  border: 1px solid #6c757d;
  padding: 10px 20px;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
  font-size: 0.95rem;
  transition: all 0.2s ease;
}

.btn-outline:hover {
  background-color: #6c757d;
  color: white;
  transform: translateY(-1px);
}

/* Dark Theme Adaptations */
:root[data-theme="dark"] .epigraph {
  background-color: #2a2a2a;
}

:root[data-theme="dark"] .feature-item {
  background-color: #2a2a2a;
}

:root[data-theme="dark"] .env-status {
  background-color: #2a2a2a;
}

:root[data-theme="dark"] .task-card.selected {
  background-color: #1a2332;
}

/* Responsive Design */
@media (max-width: 600px) {
  .welcome-modal {
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