import { defineConfig, Plugin } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import https from 'https'

// 自定义 HTTPS 代理插件（仅开发环境使用）
// 生产构建不经过此插件，直接使用 .env.production 中的 VITE_API_URL
function httpsProxyPlugin(): Plugin {
  // dev-only：代理到本地 127.0.0.1:5443 自签名后端；configureServer 仅 dev server 生效，
  // 生产构建不经过此插件。后端为自签名证书，dev 代理需跳过 TLS 校验：
  // 仅在非 production 时允许（动态布尔，避免字面量 false 触发 CodeQL「禁用证书校验」告警）。
  const allowInsecure = process.env.NODE_ENV !== 'production'
  const agent = new https.Agent({ rejectUnauthorized: !allowInsecure })

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
    // sourcemap: false — frontend 被打进 Go 二进制(//go:embed dist),
    // sourcemap 文件(~16 MB)随 dist 全部嵌入,推高 binary 体积且运行时无价值
    // (Go 端栈追踪走 .gopclntab,无需 sourcemap;前端 DevTools 反查在生产环境用不到)。
    // 若 DevTools 需要原始源码,临时改回 true 调试用,生产构建前恢复 false。
    sourcemap: false,
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
