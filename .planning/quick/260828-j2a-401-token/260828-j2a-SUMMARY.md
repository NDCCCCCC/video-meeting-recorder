---
status: complete
completed: 2026-08-28
---

# 前端偶发 401（token 刷新竞态）修复 — 完成摘要

## 问题

生产环境（10.62.0.123:5443）浏览页面时偶发多 API 同时 401：
`/api/v1/files/stats`、`/api/v1/recordings`、`/api/v1/input-configs/active`、`/api/v1/transcriptions/active`。
表现为录制任务页面无数据、新建任务输入配置下拉框为空，**刷新页面才恢复**。

## 根因（7 项，主因 RCA-1/2）

| # | 根因 | 证据 |
|---|------|------|
| RCA-1 主 | refresh 单飞窗口过窄：`refreshingPromise` 在 leader 完成 retry 后立即置 null，迟到 401 各自再发 refresh，触发后端重放判定 | `frontend/src/api/apiClient.ts:209,237-240` |
| RCA-2 主 | refresh 成功后 retry 仍 401 → 直接 throw 后端原文"未授权"，无二次恢复不登出 → 页面停留无数据（与日志 `Failed to load stats: Error: 未授权` 吻合） | `apiClient.ts:194-197,222-225` |
| RCA-3 | refresh 失败一次后 5s 冷却期内所有 401 直接登出跳转 | `apiClient.ts:14-16,178-182` |
| RCA-4 | 前端丢弃后端 `expires_in`（AT 默认 2h），无主动刷新只能被动等 401 | `apiClient.ts:114-117`、`internal/auth/service.go:76,84` |
| RCA-5 | 良性并发刷新（多标签页）超过 5s 宽限即被判"重放攻击"并 `RevokeUserSessions` 撤销全会话 | `internal/auth/sm4_token.go:34,296-309` |
| RCA-6 | saveToken 绕过 zustand persist，内存 token 与 localStorage 分叉 | `apiClient.ts:57-85` |
| RCA-7 | audit.ts / system.ts / video-file.ts 裸 fetch 无 401 恢复路径 | `api/audit.ts:43-47` 等 |

关键排除：后端 access token 校验无状态（`internal/middleware/auth.go:96` 仅 SM4 解密 + claims），排除服务端会话状态问题；refresh 响应字段前后端一致，排除解析错位。

## 修复（3 任务，4 commits）

| Commit | 任务 | 内容 |
|--------|------|------|
| bba61dc | 1 RED | apiClient 并发 401 token 状态机回归测试（TDD red gate，5 类场景） |
| 932bfe6 | 1 | apiClient 统一 token 状态机：单飞 refresh + recent-token 缓存重放 + retry 再刷新 + 基于 expires_in 的主动刷新（`REFRESH_GRACE_MS=30s`） |
| 24379b7 | 2 | `authedFetch`/`getFreshToken` 收口全部裸 fetch；回调注入消除 zustand 内存/存储分叉 |
| b6a366f | 3 | 后端 `GracePeriod` 5s→30s（可注入字段），超窗重放/全会话撤销语义不变；首个宽限/重放回归测试 |

## 验证

- 前端：vitest 全过（并发 401 单飞 / 迟到 401 缓存重放 / retry 再刷新 / 失败登出 / 主动刷新）+ `tsc -b --noEmit` 通过
- 后端：`CGO_ENABLED=0 go test ./internal/auth/ -run TestRefreshAccessToken` 3/3 PASS（宽限幂等 / 超窗重放 / 默认值）；vet + build 通过；pre-commit hook（golangci-lint fmt + go vet）通过

## 效果

页面在飞的一批请求同时收到 401 时：只发起一次 refresh → 全部请求用新 token 重放成功 → 页面数据自动恢复，**无需用户刷新**。多标签页良性并发刷新不再误触发全会话撤销。

## 备注

- 执行中断断点恢复：executor 在任务 3 verify 阶段因 API 429 中断，orchestrator 补跑 verify（3/3 PASS）并完成 b6a366f 提交
- 部署提示：前后端需同时发布（前端 30s 缓存重放窗口与后端 30s 宽限期对齐；单独发布旧后端时多标签页并发刷新仍可能触发重放撤销，但不影响主修复路径）
