import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { applyDocumentLang, i18n } from './i18n'
import { isDesktop, isMacDesktop, isWindowsDesktop } from './lib/desktop'
import { installProseExternalLinks } from './lib/openExternal'
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
app.use(i18n)
applyDocumentLang()
useAuthStore(pinia).probe()
useUploadStore(pinia).bindDesktopDrop()
if (isDesktop()) {
  void import('./stores/updates').then(({ useUpdateStore }) => {
    useUpdateStore(pinia).start()
  })
}
installProseExternalLinks()
app.mount('#app')
