import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import https from 'https'

const httpsAgent = new https.Agent({ rejectUnauthorized: false })

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
        agent: httpsAgent,
      },
      '/ws': {
        target: 'wss://127.0.0.1:5443',
        ws: true,
        secure: false,
        agent: httpsAgent,
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
