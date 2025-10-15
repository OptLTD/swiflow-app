import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve, dirname } from 'node:path'
import { fileURLToPath, URL } from 'node:url';
import VueI18nPlugin from '@intlify/unplugin-vue-i18n/vite'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    VueI18nPlugin({
      /* options */
      // locale messages resource pre-compile option
      include: resolve(dirname(fileURLToPath(import.meta.url)), 'src/locales/**'),
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(
        new URL('./src', import.meta.url)
      ),
    }
  },
  build: {
    chunkSizeWarningLimit: 2000,
    rollupOptions: {
      plugins: []
    }
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': {
        changeOrigin: true,
        target: 'http://localhost:11235',
        rewrite: (path) => path.replace(/^\/api\//, '/api/')
      },
      '/socket': {
        ws: true,
        changeOrigin: true,
        target: 'ws://localhost:11235',
        rewrite: (path) => path.replace(/^\/socket/, '/socket')
      }
    },
    hmr: {
      overlay: true
    },
  },
})
