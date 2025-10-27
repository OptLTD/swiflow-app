import { createApp } from 'vue'
import { createPinia } from 'pinia'
import "tippy.js/dist/tippy.css"
import 'tippy.js/themes/light.css'
import 'normalize.css/normalize.css'
import './styles/index.css'
import App from './App.vue'
import VueTippy from 'vue-tippy'
import '@formkit/themes/genesis'
import 'vue-final-modal/style.css'
import { createVfm } from 'vue-final-modal'
import { i18n } from '@/config/i18n'
import 'vue3-toastify/dist/index.css';
import Vue3Toastify from 'vue3-toastify';
import {ToastContainerOptions} from 'vue3-toastify';
import { plugin, defaultConfig } from '@formkit/vue'
import IconButton from '@/widgets/IconButton.vue'


const app = createApp(App)
app.use(createPinia())
app.use(createVfm())
app.use(Vue3Toastify, {
  autoClose: 3000,
  hideProgressBar: true,
  pauseOnFocusLoss: false,
} as ToastContainerOptions);
app.use(plugin, defaultConfig)
// Register global components
app.component('Icon', IconButton)
app.use(VueTippy, {
  directive: 'tippy',
  component: 'tippy',
  defaultProps: {
    zIndex: 2048,
    placement: 'top',
    allowHTML: true,
  },
})
app.use(i18n)
app.mount('#app')