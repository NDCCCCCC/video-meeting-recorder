var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import https from 'https';
// 自定义 HTTPS 代理插件（仅开发环境使用）
// 生产构建不经过此插件，直接使用 .env.production 中的 VITE_API_URL
function httpsProxyPlugin() {
    var agent = new https.Agent({ rejectUnauthorized: false });
    return {
        name: 'https-proxy',
        configureServer: function (server) {
            server.middlewares.use(function (req, res, next) {
                var _a;
                if (!((_a = req.url) === null || _a === void 0 ? void 0 : _a.startsWith('/api'))) {
                    return next();
                }
                var options = {
                    hostname: '127.0.0.1',
                    port: 5443,
                    path: req.url,
                    method: req.method,
                    headers: __assign(__assign({}, req.headers), { host: '127.0.0.1:5443' }),
                    agent: agent,
                };
                var proxyReq = https.request(options, function (proxyRes) {
                    // 转发 CORS 头和状态码
                    res.writeHead(proxyRes.statusCode, proxyRes.headers);
                    proxyRes.pipe(res, { end: true });
                });
                proxyReq.on('error', function (err) {
                    console.error('[Proxy Error]', err.message);
                    if (!res.headersSent) {
                        res.writeHead(502, { 'Content-Type': 'application/json' });
                        res.end(JSON.stringify({ error: err.message }));
                    }
                });
                req.pipe(proxyReq, { end: true });
            });
        },
    };
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
});
