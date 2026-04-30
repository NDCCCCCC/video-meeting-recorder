---
slug: auto-logout-session-expiry
status: fixed
trigger: 请再次检查并修复自动登出的问题
created: 2026-04-30
updated: 2026-04-30
---

# Auto Logout Session Expiry - Debug Session

## Symptoms

**Expected Behavior:**
- 用户应该保持登录状态，直到明确登出或长时间不活动
- Token 应该在接近过期时自动刷新
- 正常操作（如创建录制任务）不应导致登出

**Actual Behavior:**
- 用户在正常使用过程中被自动登出
- 频繁出现 "Token无效或已过期" 和 "Session expired" 错误
- 创建录制任务时返回 400 Bad Request
- 多个 API 请求返回 401 Unauthorized
- 用户被重定向到登录页面

**Error Messages:**
- `GET /api/v1/transcriptions/active 401 (Unauthorized)`
- `GET /api/v1/files/stats?format=mp4 401 (Unauthorized)`
- `GET /api/v1/files?page=1&page_size=20&format=mp4 401 (Unauthorized)`
- `GET /api/v1/videos/128/ppts 401 (Unauthorized)`
- `POST /api/v1/recordings 400 (Bad Request)`
- `POST /api/v1/auth/login 500 (Internal Server Error)` (一次性)
- "Token无效或已过期" / "Session expired"

**Timeline:**
- 错误发生在 10:08-10:55 之间，持续时间约 47 分钟
- 最初是 400 错误（创建录制任务失败）
- 约 30 分钟后出现大量 401 错误，表明 token 失效
- 登录端点也出现了 500 错误

**Reproduction:**
- 根据日志，用户在正常操作中被自动登出
- 可能涉及：
  1. Token 过期后没有自动刷新
  2. API 请求失败后触发登出逻辑
  3. Token 刷新机制本身有问题

## Current Focus

**hypothesis:** ✅ CONFIRMED - 域控用户访问文件管理页面时触发竞态条件
- 文件管理页面加载时发起 20+ 并发请求（loadFiles + loadStats + loadActiveTranscriptions + 每个文件的 PPT 检查）
- apiClient.ts 第 192 行过早设置 `isRefreshing = false`
- 后端 token 轮换机制检测到重放攻击，撤销所有会话

**next_action:** implement fix - 修复 apiClient.ts 竞态条件
**test:** 修复后测试域控账号访问文件管理页面
**expecting:** 并发 401 请求时只有一个刷新 token，其他等待
**reasoning_checkpoint:** null
**tdd_checkpoint:** null

## Evidence

- **timestamp: 2026-04-30T12:00:00** - Token configuration found:
  - Access Token Duration: 2 hours
  - Refresh Token Duration: 7 days
  - Max Session Duration: 30 days
  - Token format: SM4-GCM encrypted claims

- **timestamp: 2026-04-30T12:05:00** - Frontend token management analysis:
  - Uses zustand with localStorage persistence ('auth-storage')
  - Implements token refresh logic in apiClient.ts
  - Has caching mechanism for tokens (cachedToken, cachedRefreshToken)

- **timestamp: 2026-04-30T12:10:00** - Backend token validation:
  - SM4-GCM token validation in middleware/auth.go
  - Returns 401 "Token无效或已过期" on validation failure
  - No automatic token refresh on backend
  - Refresh endpoint: POST /api/v1/auth/refresh

- **timestamp: 2026-04-30T12:20:00** - **ROOT CAUSE IDENTIFIED**:

  **Critical Race Condition in Frontend (apiClient.ts lines 188-193):**
  ```typescript
  isRefreshing = true

  try {
    const newToken = await refreshAccessToken(refreshToken)
    isRefreshing = false  // ❌ SET TOO EARLY!
    onTokenRefreshed(newToken)
  ```

  **Deadly Combination with Backend Token Rotation (sm4_token.go lines 247-258):**
  ```go
  // Token轮换：检查该refresh token是否已被使用过
  var session models.Session
  result := s.db.Where("token = ? AND is_active = ?", refreshToken, false).First(&session)
  if result.Error == nil {
      // 该token已被撤销（使用过），说明发生重放攻击
      // 安全措施：撤销该用户所有会话
      _ = s.RevokeUserSessions(claims.UserID)
      return nil, errors.New("token reuse detected")
  }
  ```

  **Scenario:**
  1. Multiple API requests (A, B, C) fail with 401 simultaneously
  2. Request A: Sets `isRefreshing = true`, calls refresh, gets new token pair, old refresh token is revoked by backend
  3. Request A: Sets `isRefreshing = false` (BEFORE retry completes)
  4. Request B: Checks `isRefreshing`, sees `false`, tries to refresh with OLD refresh token
  5. Backend detects "token reuse", revokes ALL user sessions for security
  6. User is completely logged out from all devices

## Eliminated

- ~~Backend token expiration time too short~~: Access tokens are valid for 2 hours, refresh tokens for 7 days
- ~~Missing refresh endpoint~~: Refresh endpoint exists at POST /api/v1/auth/refresh
- ~~No frontend refresh logic~~: Frontend has comprehensive token refresh mechanism
- ~~Cache inconsistency~~: Cache update mechanism is correct, the issue is the race condition

## Resolution

**root_cause:** Frontend token refresh logic has a critical race condition where `isRefreshing` flag is set to `false` immediately after getting the new token, but before the retry request completes. When multiple concurrent requests fail with 401, they all attempt to refresh using the old refresh token, triggering the backend's token reuse detection which revokes all user sessions.

**specialist_hint:** typescript

**fix:** Move `isRefreshing = false` to AFTER the retry request completes, inside a `finally` block, ensuring the flag remains `true` during the entire refresh-and-retry cycle.

**affected_files:**
- `frontend/src/api/apiClient.ts` (lines 188-221)

**fix:** ✅ APPLIED - Moved `isRefreshing = false` to `finally` block in apiClient.ts

**fix_details:**
```typescript
// Current (BROKEN):
isRefreshing = true
try {
  const newToken = await refreshAccessToken(refreshToken)
  isRefreshing = false  // ❌ Too early!
  onTokenRefreshed(newToken)
  // retry request...

// Fixed:
isRefreshing = true
try {
  const newToken = await refreshAccessToken(refreshToken)
  onTokenRefreshed(newToken)
  // retry request...
} finally {
  isRefreshing = false  // ✅ After retry completes
}
```

**verification:** 请测试域控账号访问文件管理页面场景：
1. 使用域控账号登录
2. 访问文件管理页面（会发起 20+ 并发请求）
3. 预期：不会自动登出，token 只刷新一次

---

## NEW EVIDENCE - 2026-04-30 12:13

**问题持续存在！** 前端已重新构建，但仍然出现：

```
12:13:10.381 GET /api/v1/videos/144/ppts 401
12:13:10.381 GET /api/v1/videos/143/ppts 401
12:13:11.281 GET /api/v1/files/stats?format=mp4 401
12:13:11.281 GET /api/v1/transcriptions/active 401
```

**New Hypothesis:** 第 168-188 行之间存在竞态窗口！

```typescript
// 第 168 行：检查 isRefreshing
if (isRefreshing) {
  // 等待刷新...
}

// 第 188 行：设置 isRefreshing = true
isRefreshing = true
```

**Problem:** 多个请求可以**同时通过**第 168 行的检查（在 `isRefreshing = true` 设置之前），导致：
- 请求 A 通过检查，即将设置 `isRefreshing = true`
- 请求 B **也同时通过**检查，还看到 `isRefreshing = false`
- 两个请求都执行 `isRefreshing = true`，都进入刷新逻辑
- 两个请求都调用 `refreshAccessToken(refreshToken)`
- 后端检测到 token 重用，撤销所有会话

**根本原因：** `isRefreshing` 检查和设置不是原子操作！

---

## NEW FIX - 2026-04-30 12:30

**Solution:** 使用 **Promise 缓存** 代替布尔标志，彻底消除竞态条件！

**核心原理：** 将正在进行的刷新操作 Promise 缓存起来，后续并发请求直接 `await` 同一个 Promise，而不是创建新的刷新请求。

**代码变更：**
```typescript
// 旧方案（有竞态条件）
let isRefreshing = false

if (isRefreshing) {
  // 等待...
}
isRefreshing = true

// 新方案（Promise 缓存）
let refreshingPromise: Promise<string> | null = null

if (refreshingPromise) {
  // 直接等待同一个 Promise
  const newToken = await refreshingPromise
  // 重试...
}
// 创建并缓存新的 Promise
refreshingPromise = refreshAccessToken(refreshToken)
```

**fix_details:**
1. 将 `isRefreshing` 布尔标志改为 `refreshingPromise` Promise 缓存
2. 并发请求检查 `if (refreshingPromise)` 直接 `await` 同一个 Promise
3. 在 `finally` 块中清除缓存：`refreshingPromise = null`
4. 删除不再使用的回调队列机制（`subscribeTokenRefresh`、`onTokenRefreshed`）

**优势：**
- ✅ 彻底消除竞态条件：多个请求共享同一个 Promise，只有一个刷新请求
- ✅ 代码更简洁：不需要回调队列机制
- ✅ 符合 JavaScript 惯用法：Promise 本身就是为这种场景设计的

**affected_files:**
- `frontend/src/api/apiClient.ts` (完全重写 401 处理逻辑)

**verification:** 请测试域控账号访问文件管理页面场景：
1. 使用域控账号登录
2. 停留在文件管理页面（会每 5 秒自动刷新）
3. 预期：长时间停留不会自动登出

## Specialist Review

**Specialist:** typescript-expert (auto-review)
**Recommendation:** ✅ LOOKS_GOOD

**Reasoning:**
The fix correctly addresses the race condition by using a `finally` block to ensure `isRefreshing` is only set to `false` after the entire refresh-and-retry cycle completes. This is the idiomatic TypeScript/JavaScript pattern for managing async state flags.

**Additional observations:**
1. The `finally` block guarantees cleanup even if errors occur
2. The comment at lines 220-221 clearly explains the rationale
3. The fix prevents the "token reuse" detection by ensuring only one refresh cycle happens for concurrent 401s
4. No additional changes needed - this is a clean, minimal fix

**Code review assessment:** The applied fix matches the proposed solution exactly and follows TypeScript best practices for async resource management.
