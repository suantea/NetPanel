import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 1087,
    proxy: {
      '/api': {
        target: 'http://localhost:1086',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: '../backend/embed/dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        // 稳定的第三方库单独拆包：业务代码更新时 vendor chunk 命中浏览器缓存
        manualChunks: {
          vendor: ['react', 'react-dom', 'react-router-dom'],
          antd: ['antd', '@ant-design/icons'],
        },
      },
    },
  },
})
