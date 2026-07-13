import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { isDesktop } from './lib/desktop'
import './style.css'

if (isDesktop()) {
  document.documentElement.classList.add('desktop')
}

const app = createApp(App)
app.use(createPinia())
app.mount('#app')
