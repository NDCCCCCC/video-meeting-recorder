import { defineConfig, Plugin } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import https from 'https'
import http from 'http'

// 自定义 HTTPS 代理插件 - 绕过 http-proxy 的 TLS 问题
function httpsProxyPlugin(): Plugin {
  const agent = new https.Agent({ rejectUnauthorized: false })

  return {
    name: 'https-proxy',
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        if (!req.url?.startsWith('/api')) {
          return next()
        }

        const options: https.RequestOptions = {
          hostname: '127.0.0.1',
          port: 5443,
          path: req.url,
          method: req.method,
          headers: {
            ...req.headers,
            host: '127.0.0.1:5443',
          },
          agent,
        }

        const proxyReq = https.request(options, (proxyRes) => {
          // 转发 CORS 头和状态码
          res.writeHead(proxyRes.statusCode!, proxyRes.headers)
          proxyRes.pipe(res, { end: true })
        })

        proxyReq.on('error', (err) => {
          console.error('[Proxy Error]', err.message)
          if (!res.headersSent) {
            res.writeHead(502, { 'Content-Type': 'application/json' })
            res.end(JSON.stringify({ error: err.message }))
          }
        })

        req.pipe(proxyReq, { end: true })
      })
    },
  }
}

export default defineConfig({
  plugins: [react(), httpsProxyPlugin()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
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
