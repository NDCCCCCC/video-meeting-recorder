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
        target: 'https://localhost:5443',
        changeOrigin: true,
        secure: false,
        configure: (proxy) => {
          proxy.on('proxyReq', (proxyReq, req, res) => {
            console.log('[Proxy] Request:', req.method, req.url)
          })
          proxy.on('proxyRes', (proxyRes, req, res) => {
            console.log('[Proxy] Response:', proxyRes.statusCode, req.url)
          })
          proxy.on('error', (err, req, res) => {
            console.log('[Proxy] Error:', err.message)
            res.writeHead(500, { 'Content-Type': 'text/plain' })
            res.end(JSON.stringify({ error: err.message }))
          })
        },
      },
      '/ws': {
        target: 'wss://localhost:5443',
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
