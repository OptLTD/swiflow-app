<script setup lang="ts">
import { ref, computed, reactive } from 'vue'
import { VueFinalModal } from 'vue-final-modal'
import InitIntro from './welcome/InitIntro.vue'
import InitSignup from './welcome/InitSignup.vue'
import InitSetKey from './welcome/InitSetKey.vue'
import InitSetEnv from './welcome/InitSetEnv.vue'
import InitSample from './welcome/InitSample.vue'

// Types for better type safety
type WizardStep = 1 | 2 | 3 | 4
type WizardMode = 'trial' | 'apikey' | ''

interface WizardState {
  currentStep: WizardStep
  selectedMode: WizardMode
  apiConfigured: boolean
  envConfigured: boolean
  pyInstalling: boolean
  selectedTask: any | null
  userinfo: any | null
  loading: boolean
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
  apiConfigured: false,
  envConfigured: false,
  pyInstalling: false,
  selectedTask: null,
  loading: false,
  userinfo: null
}

// Merge default state with initial state from props
const state = reactive<WizardState>({
  ...defaultState,
  ...props.initialState
})

// Widget component refs
const initSignupRef = ref<typeof InitSignup>()
const initSetKeyRef = ref<typeof InitSetKey>()
const initSetEnvRef = ref<typeof InitSetEnv>()
const initSampleRef = ref<typeof InitSample>()

// Navigation methods with improved logic
const nextStep = () => {
  if (state.currentStep < totalSteps) {
    state.currentStep++
  }
}

const goToStep = (step: WizardStep) => {
  if (step <= state.currentStep) {
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
    case 4: return state.selectedTask !== null // Task selected
    default: return false
  }
})

const handleSaveApiConfig = async () => {
  try {
    if (initSetKeyRef.value?.saveApiConfig) {
      state.loading = true
      await initSetKeyRef.value.saveApiConfig()
    }
  } catch (err) {
    console.error('API config save error:', err)
  } finally {
    state.loading = false
  }
}

const handleSaveApiSuccess = () => {
  state.apiConfigured = true
  nextStep()
}

const handleCheckPythonEnv = () => {
  // Delegate to InitSetEnv widget
  if (initSetEnvRef.value?.checkPythonEnv) {
    initSetEnvRef.value.checkPythonEnv()
  }
}

const handleInstallPython = () => {
  // Delegate to InitSetEnv widget
  if (initSetEnvRef.value?.installPython) {
    initSetEnvRef.value.installPython()
  }
}

const handleEnvUpdated = (envInfo: any) => {
  // Environment info is now managed internally by InitSetEnv
  state.envConfigured = !!(envInfo.python && envInfo.uvx)
}

const handleModeChanged = (mode: WizardMode) => {
  state.selectedMode = mode
}

const handleSignin = (login: any) => {
  state.userinfo = login
}

const handleSignup = () => {
  // Delegate to InitSignup widget
  if (initSignupRef.value?.gotoSignUp) {
    initSignupRef.value.gotoSignUp()
  }
}

// Task selection with validation
const selectTask = (task: any) => {
  state.selectedTask = task
}

const submitTask = (task: any) => {
  emit('submit', task)
}

const completeSetup = () => {
  emit('submit', state.selectedTask)
}

const onCancel = () => {
  emit('cancel')
}
</script>

<template>
  <VueFinalModal 
    :esc-to-close="false" 
    :click-to-close="false" 
    class="swiflow-modal-wrapper"
    content-class="welcome-modal">
    <div class="welcome-container">
      <div class="welcome-header">
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

      <!-- Loading Indicator -->
      <div v-if="state.loading" class="loading-indicator">
        <div class="loading-spinner">⏳</div>
        <span>{{ $t('welcome.loading') }}</span>
      </div>

      <!-- Step Content -->
      <div class="welcome-content">
        <!-- Step 1: Introduction -->
        <InitIntro v-if="state.currentStep === 1" />

        <!-- Step 2: Configuration -->
        <div v-if="state.currentStep === 2">
          <!-- Trial mode: Login widget -->
          <InitSignup v-if="state.selectedMode === 'trial'" 
            :gateway="props.gateway" :userinfo="state.userinfo"
            @signin="handleSignin" ref="initSignupRef" />

          <!-- API Key mode: Configuration widget -->
          <InitSetKey v-if="state.selectedMode === 'apikey'" 
            @save="handleSaveApiSuccess" ref="initSetKeyRef" 
          />
        </div>

        <!-- Step 3: Python Environment -->
        <InitSetEnv v-if="state.currentStep === 3"  
          ref="initSetEnvRef" @env-updated="handleEnvUpdated"
        />

        <!-- Step 4: Sample Tasks -->
        <InitSample v-if="state.currentStep === 4"  ref="initSampleRef"
          @select-task="selectTask" @submit-task="submitTask"
        />
      </div>

      <!-- Footer with navigation -->
      <div class="welcome-footer">
        <div class="footer-actions">
          <!-- Always show skip guide button on the left -->
          <button class="btn-outline" @click="onCancel">
            {{ $t('welcome.skipGuide') }}
          </button>

          <!-- Action buttons on the right -->
          <div class="action-buttons">
            <!-- Step 1: Mode selection buttons -->
            <button :disabled="state.loading" v-if="(state.currentStep === 1)" 
              class="btn-primary" @click="nextStep">
              {{ $t('welcome.nextStep') }}
            </button>

            <!-- Step 2: Mode-specific buttons -->
            <template v-if="state.currentStep === 2">
              <!-- Mode switch button -->
              <button class="btn-outline" :disabled="state.loading"
                @click="handleModeChanged(state.selectedMode === 'trial' ? 'apikey' : 'trial')">
                {{ state.selectedMode === 'trial' ? $t('welcome.setApiKey') : $t('welcome.noApiKey') }}
              </button>

              <!-- Trial mode buttons -->
              <button v-if="state.selectedMode === 'trial' && !state.userinfo" 
                @click="handleSignup" class="btn-primary" :disabled="state.loading">
                {{ $t('welcome.register') }}
                <a target="_blank" id="signupUrl" style="display: none;" />
              </button>

              <!-- Continue button for trial mode when waiting for auth or login completed -->
              <button v-if="state.selectedMode === 'trial' && state.userinfo" 
                @click="nextStep" class="btn-primary" :disabled="state.loading">
                {{ $t('welcome.nextStep') }}
              </button>

              <!-- API Key mode buttons -->
              <button v-if="state.selectedMode === 'apikey'" class="btn-primary" 
                @click="handleSaveApiConfig" :disabled="state.loading">
                {{ state.loading ? $t('welcome.saving') : $t('welcome.saveConfig') }}
              </button>
            </template>

            <!-- Step 3: Environment Configuration buttons -->
            <template v-if="state.currentStep === 3 && !state.envConfigured">
              <button class="btn-outline" :disabled="state.loading" 
                @click="handleCheckPythonEnv">
                {{ state.loading ? $t('welcome.checking') : $t('welcome.recheck') }}
              </button>
              <button class="btn-primary" @click="handleInstallPython" 
                :disabled="state.pyInstalling || state.loading">
                {{ state.pyInstalling ? $t('welcome.installing') : $t('welcome.installPython') }}
              </button>
            </template>
            <template v-if="state.currentStep === 3 && state.envConfigured">
              <button class="btn-primary" @click="nextStep" 
                :disabled="!canProceed || state.loading">
                {{ $t('welcome.nextStep') }}
              </button>
            </template>

            <button v-if="state.currentStep === 4" class="btn-primary"
              @click="completeSetup" :disabled="!canProceed || state.loading">
              {{ state.selectedTask ? $t('welcome.startTrial') : $t('welcome.startExperience') }}
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
  height: 100%;
  flex-direction: column;
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
  min-height: 450px;
  max-height: 450px;
}

.welcome-footer {
  padding-top: 15px;
  flex-shrink: 0;
  margin-top: auto;
  border-top: 1px solid var(--color-divider);
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

/* Common Components - removed unused step-content and step-description as they're defined in child components */

/* Status Messages - removed error-message styles as error handling logic has been removed */

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

/* Waiting and Loading States - removed unused waiting styles as they're defined in child components */

/* Animations */
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* Dark Theme - removed unused dark theme styles for components defined in child components and error-message */

/* Responsive Design */
@media (max-width: 600px) {
  :global(.welcome-modal) {
    max-width: 95%;
    padding: 20px;
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