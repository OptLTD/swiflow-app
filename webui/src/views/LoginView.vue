<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'

const { t } = useI18n()
const auth = useAuthStore()

const name = ref('')
const password = ref('')
const mode = ref<'login' | 'register'>('login')
const busy = ref(false)
const error = ref('')

async function submit() {
  error.value = ''
  const n = name.value.trim()
  const p = password.value
  if (!n || !p) {
    error.value = t('login.required')
    return
  }
  if (mode.value === 'register' && p.length < 6) {
    error.value = t('login.passwordMin')
    return
  }
  busy.value = true
  try {
    if (mode.value === 'login') await auth.login(n, p)
    else await auth.register(n, p)
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="login-shell">
    <form class="login-card" @submit.prevent="submit">
      <h1 class="login-brand">Swiflow</h1>
      <p class="login-sub">{{ mode === 'login' ? t('login.subtitleLogin') : t('login.subtitleRegister') }}</p>

      <label class="login-label">
        <span>{{ t('login.name') }}</span>
        <input v-model="name" type="text" autocomplete="username" required />
      </label>
      <label class="login-label">
        <span>{{ t('login.password') }}</span>
        <input v-model="password" type="password" autocomplete="current-password" required />
      </label>

      <p v-if="error" class="login-error">{{ error }}</p>

      <button type="submit" class="login-submit" :disabled="busy">
        {{ busy ? t('common.loading') : mode === 'login' ? t('login.submitLogin') : t('login.submitRegister') }}
      </button>

      <button
        type="button"
        class="login-switch"
        :disabled="busy"
        @click="mode = mode === 'login' ? 'register' : 'login'"
      >
        {{ mode === 'login' ? t('login.switchRegister') : t('login.switchLogin') }}
      </button>
    </form>
  </div>
</template>

<style scoped>
.login-shell {
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  background:
    radial-gradient(1200px 600px at 20% -10%, #dbeafe 0%, transparent 55%),
    radial-gradient(900px 500px at 90% 10%, #e2e8f0 0%, transparent 50%),
    #f8fafc;
}
.login-card {
  width: min(100%, 360px);
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  padding: 2rem 1.75rem;
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid #e2e8f0;
  border-radius: 12px;
}
.login-brand {
  margin: 0;
  font-size: 1.75rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: #0f172a;
}
.login-sub {
  margin: -0.35rem 0 0.35rem;
  color: #64748b;
  font-size: 0.9rem;
}
.login-label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: 0.85rem;
  color: #334155;
}
.login-label input {
  border: 1px solid #cbd5e1;
  border-radius: 8px;
  padding: 0.55rem 0.7rem;
  font-size: 0.95rem;
  outline: none;
}
.login-label input:focus {
  border-color: #64748b;
  box-shadow: 0 0 0 3px rgba(100, 116, 139, 0.15);
}
.login-error {
  margin: 0;
  color: #b91c1c;
  font-size: 0.85rem;
}
.login-submit {
  margin-top: 0.25rem;
  border: none;
  border-radius: 8px;
  padding: 0.65rem 0.9rem;
  background: #0f172a;
  color: #fff;
  font-weight: 600;
}
.login-submit:disabled {
  opacity: 0.6;
}
.login-switch {
  border: none;
  background: transparent;
  color: #475569;
  font-size: 0.85rem;
  text-align: center;
  padding: 0.25rem;
}
.login-switch:hover {
  color: #0f172a;
}
</style>
