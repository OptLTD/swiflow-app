import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve, dirname } from 'node:path'
import { fileURLToPath, URL } from 'node:url';
import { visualizer } from 'rollup-plugin-visualizer'
import VueI18nPlugin from '@intlify/unplugin-vue-i18n/vite'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    VueI18nPlugin({
      /* options */
      // locale messages resource pre-compile option
      include: resolve(dirname(fileURLToPath(import.meta.url)), 'src-front/locales/**'),
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(
        new URL('./src-front', import.meta.url)
      ),
    }
  },
  build: {
    chunkSizeWarningLimit: 2000,
    rollupOptions: {
      plugins: [
        visualizer({
          filename: 'dist/stats.html',
          template: 'treemap',
          gzipSize: true,
          brotliSize: true,
        })
      ],
      output: {
        manualChunks(id) {
          // Split core Vue ecosystem
          if (id.includes('node_modules/vue') || id.includes('node_modules/vue-router') || id.includes('node_modules/pinia')) return 'vue-core'
          // Common libs
          if (id.includes('node_modules/axios')) return 'axios'
          if (id.includes('node_modules/lodash') || id.includes('node_modules/lodash-es')) return 'lodash'
          if (id.includes('node_modules/vue3-toastify')) return 'toastify'
          if (id.includes('node_modules/@intlify')) return 'i18n'
          if (id.includes('node_modules/@formkit')) return 'formkit'
          if (id.includes('node_modules/mermaid')) return 'mermaid'
          if (id.includes('node_modules/cytoscape')) return 'cytoscape'
          if (id.includes('node_modules/katex')) return 'katex'
          if (id.includes('node_modules/highlight.js')) return 'highlight'
          if (id.includes('node_modules/markdown-it')) return 'markdownit'
          if (id.includes('node_modules/codemirror') || id.includes('node_modules/@codemirror')) return 'codemirror'
          if (id.includes('node_modules/xlsx')) return 'xlsx'
          if (id.includes('node_modules/vue-final-modal')) return 'vfm'
          if (id.includes('node_modules/@vueuse')) return 'vueuse'
          // PDF viewer stack
          if (id.includes('node_modules/pdfjs-dist') || id.includes('node_modules/@tato30/vue-pdf')) return 'pdfjs'
          // UI tooltip stack
          if (id.includes('node_modules/vue-tippy') || id.includes('node_modules/tippy.js')) return 'tippy'
          if (id.includes('node_modules/@popperjs/core')) return 'popper'
          if (id.includes('node_modules')) return 'vendor'
        }
      }
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
