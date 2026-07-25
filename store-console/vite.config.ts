import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 门店后台独立 Vite 工程：独立构建产物、独立环境变量、独立 API client。
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5181,
    host: true,
    // 本地开发：将 /api/v2/* 代理到本地后端，避免跨域 CORS 预检失败。
    proxy: {
      '/api/v2': {
        target: 'http://127.0.0.1:8081',
        changeOrigin: true,
      },
    },
  },
  preview: {
    port: 5181,
  },
})
