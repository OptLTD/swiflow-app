import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { isDesktop, isMacDesktop, isWindowsDesktop } from './lib/desktop'
import { useAuthStore } from './stores/auth'
import { useUploadStore } from './stores/upload'
import './style.css'

if (isDesktop()) {
  document.documentElement.classList.add('desktop')
  if (isMacDesktop()) document.documentElement.classList.add('platform-darwin')
  if (isWindowsDesktop()) document.documentElement.classList.add('platform-windows')
}

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
useAuthStore(pinia).probe()
useUploadStore(pinia).bindDesktopDrop()
app.mount('#app')
