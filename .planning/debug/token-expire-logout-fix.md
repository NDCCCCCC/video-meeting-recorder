# Token过期导致系统登出问题修复

**日期:** 2026-05-12
**状态:** 已修复
**影响页面:** 文件管理页面

## 问题描述

当token过期时，文件管理页面会产生多个unauthorized请求（如`recordings/69/preview/stream/index.m3u8`、`files/stats`等），导致系统多次登出，出现竞态条件。

## 根本原因

文件管理页面加载时会并发发起三个请求：
- `loadFiles()` - 获取文件列表
- `loadStats()` - 获取统计信息  
- `loadActiveTranscriptions()` - 获取活跃转录任务

当token过期时，这些请求都返回401，每个请求都会独立触发`handleUnauthorized()`函数，导致：
1. 多次清除token
2. 多次跳转登录页
3. 可能的竞态条件

## 修复方案

**文件:** `frontend/src/api/apiClient.ts`

### 1. 添加登出标志

```typescript
// 正在登出的标志，防止并发请求重复登出
let isLoggingOut = false
```

### 2. 更新handleUnauthorized()函数

```typescript
function handleUnauthorized() {
  // 防止并发请求重复触发登出
  if (isLoggingOut) {
    return
  }

  isLoggingOut = true
  clearToken()
  // ... 跳转登录页
}
```

### 3. 更新saveToken()函数

```typescript
const saveToken = (accessToken: string, refreshToken: string): void => {
  // 重置登出标志，允许后续的登出操作
  isLoggingOut = false
  // ... 其余token保存逻辑
}
```

## 工作原理

```
Token过期时的流程：
┌─────────────────────────────────────────────────────────────┐
│ 并发请求A、B、C同时返回401                                     │
├─────────────────────────────────────────────────────────────┤
│ 请求A: handleUnauthorized() → 设置isLoggingOut=true → 登出    │
│ 请求B: handleUnauthorized() → 检测到isLoggingOut=true → 跳过  │
│ 请求C: handleUnauthorized() → 检测到isLoggingOut=true → 跳过  │
├─────────────────────────────────────────────────────────────┤
│ 用户登录后: saveToken() → 重置isLoggingOut=false             │
│ 下次token过期时又可以正常登出                                    │
└─────────────────────────────────────────────────────────────┘
```

## 测试方法

1. 等待token即将过期（或手动使token无效）
2. 访问文件管理页面
3. 验证只跳转一次到登录页
4. 登录后验证可以正常使用

## 相关代码

- `frontend/src/api/apiClient.ts` - API客户端和401处理逻辑
- `frontend/src/pages/files/index.tsx:196-198` - 并发请求触发点
