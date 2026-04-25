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
    port: 5173,
    proxy: {
      '/api': {
        target: 'https://127.0.0.1:5443',
        changeOrigin: true,
        secure: false,
        configure: (proxy, _options) => {
          proxy.on('proxyReq', (proxyReq, req, res) => {
            // 添加日志用于调试
            console.log('[Proxy] Request:', req.method, req.url, '->', proxyReq.getHeader('host'))
          })
        },
      },
      '/ws': {
        target: 'wss://127.0.0.1:5443',
        ws: true,
        secure: false,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom', 'react-router-dom'],
          'antd': ['antd', 'dayjs'],
          'query': ['@tanstack/react-query'],
          'motion': ['framer-motion'],
        },
      },
    },
  },
})
