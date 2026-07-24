import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    port: 5173,
    host: true,
    proxy: {
      '/api': 'http://127.0.0.1:8000',
    },
  },
  build: {
    outDir: '../embed/frontend',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return
          if (id.includes('pdfjs-dist')) return 'pdfjs'
          if (id.includes('xlsx')) return 'xlsx'
          if (id.includes('jspreadsheet') || id.includes('jsuites')) return 'spreadsheet'
          if (id.includes('mammoth')) return 'mammoth'
          if (id.includes('@codemirror')) return 'codemirror'
          if (id.includes('highlight.js') || id.includes('markdown-it')) return 'markdown'
          if (id.includes('vue') || id.includes('pinia')) return 'vue'
        },
      },
    },
  },
})
