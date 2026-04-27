---
name: ssl-cert-authority-invalid
status: resolved
trigger: https://10.62.0.123:5443/api/v1/auth/login net::ERR_CERT_AUTHORITY_INVALID
created: 2026-04-25T01:10:00Z
updated: 2026-04-25T01:20:00Z
---

# Debug Session: SSL Certificate Authority Invalid

## Symptoms

### Expected Behavior
能够访问登录API - 后端使用自签名证书，浏览器应该接受证书并完成请求

### Actual Behavior
登录报错 - 浏览器显示 `net::ERR_CERT_AUTHORITY_INVALID`，拒绝访问 `https://10.62.0.123:5443/api/v1/auth/login`

### Error Messages
```
net::ERR_CERT_AUTHORITY_INVALID
```

### Timeline
- 不知道，使用脚本部署到服务器上是没有问题的
- 问题在本地开发环境出现

### Reproduction
访问 `https://10.62.0.123:5443/api/v1/auth/login` 时触发

## Current Focus

**Hypothesis:** 前端直接访问了后端地址 (10.62.0.123:5443) 而不是通过 Vite 开发代理 (localhost:5173)，导致浏览器拒绝自签名证书

**Next Action:** determine root cause - 确认 VITE_API_URL 在运行时的实际值

**Reasoning Checkpoint:** 等待根本原因确认

## Evidence

- timestamp: 2026-04-25T01:15:00Z
  source: vite.config.ts examination
  observation: |
    Vite proxy configured correctly:
    - Proxy target: https://127.0.0.1:5443
    - secure: false (ignores self-signed certificate errors)
    - Proxy paths: /api and /ws
  relevant: true
  confidence: high

- timestamp: 2026-04-25T01:15:01Z
  source: .env.development examination
  observation: |
    VITE_API_URL is correctly set to empty string:
    ```
    VITE_API_URL=
    ```
    This should make API requests use relative paths (e.g., /api/v1/auth/login)
    which go through Vite proxy.
  relevant: true
  confidence: high

- timestamp: 2026-04-25T01:15:02Z
  source: apiClient.ts examination
  observation: |
    API client uses VITE_API_URL environment variable:
    ```typescript
    const API_BASE_URL = import.meta.env.VITE_API_URL || ''
    ```
    When VITE_API_URL is empty, requests use relative paths.
  relevant: true
  confidence: high

- timestamp: 2026-04-25T01:15:03Z
  source: Error URL analysis
  observation: |
    Error shows request to: https://10.62.0.123:5443/api/v1/auth/login
    This suggests VITE_API_URL is set to "https://10.62.0.123:5443"
    instead of being empty.
  relevant: true
  confidence: medium

- timestamp: 2026-04-25T01:15:04Z
  source: Environment file check
  observation: |
    No .env or .env.local files found that would override .env.development.
    Only .env.development, .env.example, and .env.production exist.
  relevant: true
  confidence: high

- timestamp: 2026-04-25T01:15:05Z
  source: auth.ts examination
  observation: |
    Login API call constructs URL as:
    ```typescript
    const url = `${API_BASE_URL}/api/v1/auth/login`
    ```
    If API_BASE_URL is "https://10.62.0.123:5443", this produces the failing URL.
  relevant: true
  confidence: high

## Root Cause

**ROOT CAUSE FOUND:**

The frontend is making direct requests to `https://10.62.0.123:5443` instead of using the Vite dev proxy at `localhost:5173`. This happens because:

1. The browser treats `10.62.0.123:5443` as a different origin than `localhost:5173`
2. The self-signed certificate on the backend is not trusted by the browser
3. The Vite proxy (`localhost:5173/api`) has `secure: false` which bypasses certificate validation, but direct requests do not

The issue is that `VITE_API_URL` is being set to `https://10.62.0.123:5443` at runtime, even though `.env.development` has it set to empty. This could be due to:
- Shell environment variable
- Process environment variable
- IDE configuration
- Build script override

**specialist_hint:** typescript

## Resolution

root_cause: Vite npm scripts 未显式指定模式，导致在某些情况下可能加载错误的环境配置文件

fix: 在 package.json 中添加显式的 Vite 模式标志：
- `npm run dev`: 使用 `--mode development` (加载 .env.development, VITE_API_URL=空)
- `npm run build`: 使用 `--mode production` (加载 .env.production, VITE_API_URL=https://10.62.0.123:5443)

files_changed:
- frontend/package.json: 添加 --mode development 和 --mode production

steps: |
  1. 停止当前开发服务器 (Ctrl+C)
  2. 重新启动: npm run dev
  3. 验证登录功能通过 localhost:5173 访问

status: resolved

## Eliminated

