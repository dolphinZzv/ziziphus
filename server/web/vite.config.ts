import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'url'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://47.95.200.101:10011',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://47.95.200.101:10011',
        ws: true,
      },
      '/files': {
        target: 'http://47.95.200.101:10011',
        changeOrigin: true,
      },
    },
  },
})
