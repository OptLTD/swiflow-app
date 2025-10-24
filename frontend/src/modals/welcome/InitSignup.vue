<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  userinfo: any
  gateway?: string
}>()

// Emits for parent component communication
const emit = defineEmits<{
  signup: []
  signin: [login: any]
}>()

const waitingSignin = ref(false)
const gotoSignUp = async () => {
  try {
    waitingSignin.value = true
    const path = 'authorization?from=swiflow-app'
    const signup = document.getElementById('signupUrl')
    const gateway = props.gateway || 'https://auth.swiflow.cc'
    
    signup?.setAttribute('href', `${gateway}/${path}`)
    const result = signup && signup.click && signup.click()

    // Start checking localStorage for signin info every 300ms
    const checkSigninInterval = setInterval(() => {
      try {
        const userInfo = localStorage.getItem('login')
        if (userInfo) {
          console.log('user info detected:', userInfo)
          clearInterval(checkSigninInterval)
          
          // Parse and store signin info
          let parsedUserinfo
          try {
            parsedUserinfo = JSON.parse(userInfo)
          } catch {
            parsedUserinfo = { token: userInfo } // If not JSON, treat as token
          }
          
          // Update auth state
          emit('signin', parsedUserinfo)
        }
      } catch (error) {
        console.error('Error checking localStorage:', error)
      }
    }, 300)
    
    // Clear interval after 5 minutes to prevent infinite checking
    setTimeout(() => {
      clearInterval(checkSigninInterval)
    }, 300000) // 5 minutes timeout
    
    return result
  } catch (err) {
    console.error('Signup error:', err)
    // Reset waiting state on error
    emit('signin', props.userinfo)
  }
}
// Expose component capabilities
defineExpose({
  gotoSignUp,
})
</script>

<template>
  <div class="trial-mode-content">
    <!-- Waiting for authentication -->
    <div v-if="waitingSignin && !userinfo" class="waiting-content">
      <div class="waiting-message">
        <div class="loading-spinner">⏳</div>
        <h4>{{ $t('welcome.waitingAuth') }}</h4>
        <p>{{ $t('welcome.completeAuthInNewPage') }}</p>
        <p class="waiting-tip">{{ $t('welcome.returnAfterAuth') }}</p>
      </div>
    </div>

    <!-- signin success display -->
    <div v-else-if="waitingSignin && userinfo" class="signin-success-content">
      <h3>{{ $t('welcome.loginSuccess') }}</h3>
      <p class="step-description">{{ $t('welcome.welcomeBack') }}</p>
      <div class="signin-info-display">
        <div class="signin-info-item">
          <span class="info-icon">👤</span>
          <div class="info-content">
            <span class="info-label">{{ $t('welcome.userInfo') }}</span>
            <span class="info-value">{{ userinfo.username || userinfo.email || '体验用户' }}</span>
          </div>
        </div>
        <div class="signin-info-item" v-if="userinfo.email">
          <span class="info-icon">📧</span>
          <div class="info-content">
            <span class="info-label">{{ $t('welcome.email') }}</span>
            <span class="info-value">{{ userinfo.email }}</span>
          </div>
        </div>
        <div class="signin-info-item">
          <span class="info-icon">✅</span>
          <div class="info-content">
            <span class="info-label">{{ $t('welcome.status') }}</span>
            <span class="info-value">{{ $t('welcome.authenticated') }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Initial trial info -->
    <div v-else class="trial-info">
      <h3>{{ $t('welcome.signupTrialMode') }}</h3>
      <p class="step-description">{{ $t('welcome.signupTrialDesc') }}</p>
      <div class="trial-features">
        <div class="feature-item">
          <span class="feature-icon">🚀</span>
          <span>{{ $t('welcome.freeTrialFeature') }}</span>
        </div>
        <div class="feature-item">
          <span class="feature-icon">⚡</span>
          <span>{{ $t('welcome.quickStart') }}</span>
        </div>
        <div class="feature-item">
          <span class="feature-icon">🎯</span>
          <span>{{ $t('welcome.noApiKeyNeeded') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.trial-mode-content {
  text-align: center;
}

.waiting-content {
  padding: 40px 20px;
}

.waiting-message {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 15px;
}

.loading-spinner {
  font-size: 32px;
  animation: spin 2s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.waiting-message h4 {
  margin: 0;
  color: var(--color-primary);
  font-size: 18px;
}

.waiting-message p {
  margin: 0;
  color: var(--color-text-secondary);
  line-height: 1.5;
}

.waiting-tip {
  font-size: 14px;
  font-style: italic;
}

.signin-success-content {
  padding: 20px;
}

.signin-success-content h3 {
  color: var(--color-success);
  margin-bottom: 10px;
}

.step-description {
  color: var(--color-text-secondary);
}

.signin-info-display {
  background: var(--bg-menu);
  border-radius: 8px;
  padding: 20px;
  text-align: left;
  border: 1px solid var(--color-tertiary);
}

.signin-info-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--color-divider);
}

.signin-info-item:last-child {
  border-bottom: none;
}

.info-icon {
  font-size: 18px;
  width: 24px;
  text-align: center;
}

.info-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.info-label {
  font-size: 12px;
  color: var(--color-text-secondary);
  font-weight: 500;
}

.info-value {
  font-size: 14px;
  color: var(--text-main);
  font-weight: 600;
}

.trial-info h3 {
  color: var(--color-primary);
  margin-bottom: 10px;
}

.trial-features {
  display: flex;
  flex-direction: column;
  gap: 15px;
  margin: 0 auto;
  margin-top: 25px;
  text-align: left;
  max-width: 400px;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: var(--bg-menu);
  border-radius: 8px;
  border: 1px solid var(--color-tertiary);
  transition: all 0.3s ease;
}

.feature-item:hover {
  background: var(--color-bg-hover);
  border-color: var(--color-primary);
}

.feature-icon {
  font-size: 18px;
  flex-shrink: 0;
}

.feature-item span:last-child {
  font-size: 14px;
  color: var(--text-main);
  font-weight: 500;
}
</style>