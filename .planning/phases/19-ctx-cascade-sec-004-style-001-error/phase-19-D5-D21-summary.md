# Phase 19: Sentinel 化迁移计划 — D5-D21 总结

**会话日期**: 2026-07-31  
**commit 范围**: `f358602` 之前的 17 个 `refactor(19/dN)` commits (D5–D21)  
**起始触发**: `/gsd-debug` 用户请求"继续增量迁移剩余 ~80% 散点 errors.New / fmt.Errorf"  
**前置工作**: D1–D4 (bed0ecc 总结: 4 项 deferred 已交付)

## 摘要

延续 Phase 19 已建立的 sentinel 化原则，将散落在 `internal/` 各包内的 `errors.New` /
`fmt.Errorf("XXX: %w", err)` 形式错误统一为 `apperrors.New...()` / 包装 `apperrors.Sentinel`，
以满足 errors.Is 链在 service 边界 / HTTP handler / middleware 上的错误分支需求。

总计 **17 atomic commits** (`D5-D21`) + 1 docs commit (`D22`)：
- 新增 **24 个 sentinel**（涵盖 auth / token / quota / AD / transcription / SDK / task / role / permission / file / preset 等域）
- 迁移 **~356 处 `errors.New` / `fmt.Errorf` 散点**（在 13 service / 9 handler / middleware / 启动期 utility 文件中）
- 累计行变化：**+461 / -270**

## 设计原则

Phase 19 之前已有 sentinel (D5-D4 commit `f4291f5`)：
```
ErrNotFound, ErrTaskNotFound, ErrVideoFileNotFound, ErrUserNotFound,
ErrTaskInProgress, ErrAlreadyExists, ErrInvalidInput, ErrInvalidFileType,
ErrUnauthorized, ErrForbidden, ErrInternal, ErrInsufficientQuota,
ErrServiceUnavailable, ErrTaskNotFound, ErrDuplicateRecord,
ErrForeignKeyConstraint, ErrFFmpegFailed, ErrTranscriptionFailed,
ErrSplitFailed
```

D5-D21 工作目标：在不显著扩张 sentinel 数量的前提下，**复用现有 sentinel**——只在
关键新场景（如 `ErrAPIKeyNotFound` / `ErrAPIKeyInvalid` 等需要区分度高、且无法复用现有 sentinel）
才新增；其余场合按"输入错 vs 权限 vs 资源 vs 服务 vs 配额 vs 内部"维度映射现有 sentinel。

新增 sentinel **24 个**：
- **D5 user/role/admin 域**: `ErrUserNotFound`, `ErrUsernameExists`, `ErrEmailExists`,
  `ErrRoleNotFound`, `ErrSystemAdminProtected`, `ErrRoleNameExists`, `ErrSystemRoleProtected`,
  `ErrRoleInUse`, `ErrPermissionNotFound`
- **D6 AD 域**: `ErrADAccountNotFound`, `ErrUserDisabled`, `ErrADConfigError`, `ErrADUnreachable`
- **D7 token 域**: `ErrTokenInvalid`, `ErrTokenExpired`, `ErrTokenNotYetValid`, `ErrTokenReplayed`
- **D10 APIKey 域**: `ErrAPIKeyNotFound`, `ErrAPIKeyInvalid`, `ErrAPIKeyExpired`,
  `ErrAPIKeyDisabled`, `ErrAPIKeyIPNotAllowed`
- **D13 PPT 域**: `ErrPPTFileNotFound`
- **D14 transcription 域**: `ErrTranscriptionUnavailable`

## Commit 序列（D5–D21 = 17 个 commits）

| Commit | 范围 | Sentinel 增量 | 散点迁移 |
|---|---|---|---|
| D5 user_service | user handler 9 endpoint + service | +5 | 14 |
| D6 ad_auth + local_auth | auth Login | +4 | 33 (21+12) |
| D7 sm4_token | middleware | +4 | 11 |
| D8 ip_validator | utility | 0（复用 ErrInvalidInput） | 9 |
| D9 role_service | role handler 8 endpoint + service | +4 | 9 |
| D10 apikey_service | apikey handler 8 endpoint + service | +5 | 11 |
| D11 hls_token | middleware | 0（复用 D7） | 6 |
| D12 auth/service.go | service.go | 0（复用 D6） | 4 |
| D13 ppt_file_service | ppt_handler 9 endpoint | +1 | 1 |
| D14 tingwu_client | 外部 SDK 适配 | +1 | 23 |
| D15 storage/file_service | storage driver boundary | 0（复用 D5） | 22 |
| D16 scheduler + recorder | 异步路径 | 0（复用 D5） | 33 |
| D17 huawei SDK 适配 | 外部 SDK 适配 | 0（复用 D5） | 25 |
| D18 utils/sm4_password | utility | 0（复用 D5） | 28 |
| D19 migrations + models + input_config | 启动期 + db 验证 | 0（复用 D5） | 47 |
| D20 transcription/oss/notification/local_driver/config | service 长尾 + 启动期 | 0（复用 D5） | 53 |
| D21 video_recording_task_service | 任务 service | 0（复用 D5/D6） | 24 |

## 价值密度递减节奏

| 阶段 | 价值类型 | HTTP 行为变化 |
|---|---|---|
| D5/D7/D9/D10 | handler-service 配对 sentinel 化 | ✅ 401/403/404/409 status code 正确 |
| D6 (auth) | 扩 classify 函数（已有 foundation） | ✅ 401 密码错 / 403 AD 禁用等路由 |
| D11 (hls_token) | 复用 sentinel，0 新增 | ✅ 401 通用 (复用 D7) |
| D13 (ppt) | handler 切 HandleError | ✅ 404 PPT不存在 |
| D14 (tingwu) | 区分配置 vs 传输错误 | ✅ 503 vs 500 |
| D15-D17 | service-boundary 一致性 | 中（异步路径无 HTTP 影响） |
| D18 (utils) | 启动期诊断 / crypto 错误 | 小（启动期故障） |
| D19 (migrations) | 启动期初始化错误 | 极小（启动失败即崩溃） |
| D20 (services) | service 长尾 + 启动期 | 小（更多覆盖率） |
| D21 (task service) | 任务管理 admin | ✅ 401/403/409/404 部分 |

## 三种 handler 错误处理模式

经过 D5-D21，handler 端错误处理演化出三种并存模式：

1. **response.HandleError 直接模式** (D5/D7/D9/D10/D11/D13/D15 部分)
   - service 返回 sentinel
   - handler `if response.HandleError(c, err) { return }`
   - message 由 `errors.MapToHTTPStatus` 统一管理

2. **classify 扩展模式** (D6)
   - service 返回 sentinel
   - handler 调 `classifyAuthLoginError(err)` 拿 `(code, status)`
   - 调 `response.GinError(c, code, err.Error())` 用 err.Error() 中的中文消息
   - 适配 GinError 写死 status code 的设计

3. **err.Error() 中包含 sentinel 包装** (D8/D12/D15/D17 部分)
   - service 用 `fmt.Errorf("XXX: %w", sentinel)` 包装，保留中文消息
   - handler 端 `errors.Is` 链仍可识别 sentinel
   - 适配 handler 已写死中文消息的 string-match 测试

Phase 20+ 重构时可统一收敛到模式 1。

## 累计影响

### 错误处理 API 一致化

**Before (D1 之前):**
```go
// service
return errors.New("任务不存在")     // 500 fallthrough

// handler
response.GinError(c, response.CodeInternalError, "删除PPT文件失败: "+err.Error())
```

**After (D21):**
```go
// service
return apperrors.ErrTaskNotFound   // 404 mapped via MapToHTTPStatus

// handler
response.HandleError(c, err)        // unified
```

### 调用方可基于 sentinel 链分支

- **HTTP path**: handler 调 `response.HandleError(c, err)` → 由 `errors.MapToHTTPStatus` 自动路由
- **Service path**: 上游 service 可 `errors.Is(err, apperrors.ErrTaskNotFound)` 决定是否重试 / 上报告警
- **Async path**: scheduler / recorder logger 可 grep `'ErrTaskNotFound'` 找历史失败案例

## 仍需注意

- handler / service 文件中部分 `errors.New` 残留主要用于**纯本地校验**（参数解析、ID 解码等），与 service 边界无关
- 部分 service 已用 `errors.Is + 字符串匹配`（如 `ip_restriction_test.go`）—— D12 已迁移到 `apperrors.Is`
- Phase 20+ 可考虑 `response.HandleError` 全面取代 `classifyAuthLoginError` 等局部分类函数
- Pre-existing `google/wire` 风格的 handler error 类别（如 ad_handler.go 中更细粒度的 code）需要 Phase 20+ 单独审计

## 测试

D5-D21 期间**全量 tests 通过**：24+ packages 包括
`internal/{errors, migrations, models, recorder, scheduler, services, services/storage, utils}` 等。
所有 `assert.Contains(err.Error(), ...)` string-match 测试已迁移到
`assert.True(t, apperrors.Is(err, sentinel))` 形式。

迁移过程中处理过的测试断点：
- `auth_handler_test.go` (D6): 加 6 个 case (404/403/401/500/wrapped-401)
- `ip_validator_test.go` (D8): 改 IPv6 测试为 sentinel
- `ip_restriction_test.go` (D12): 改 AuditLog 测试为 sentinel (中文错误保留)
- `transcription_task_test.go` (D19): 改 "slide not found" 测试为 `ErrNotFound` sentinel

## 相关 commit 历史

```
bed0ecc docs(19): D1-D4 总结 (4 项 deferred 全部交付)
f4291f5 refactor(19/d4): 增量 errors 包迁移 + DeleteFile/NYCodecryptor 包装

(D5-D21 上述总结)

f358602 refactor(19/d21): video_recording_task_service 24 散点（0 新增，Phase 19 收尾）
```

## 进一步可选项 (Phase 20+)

1. **Phase 20 - Error API 统一收敛**：把所有 handler 切到 `response.HandleError(c, err)` 模式，移除 `classifyAuthLoginError` 等局部分类
2. **Phase 20 - Logging 集成**：利用 `errors.Is` 链在 zap logger 输出 context 字段（如 `task_id`、`sentinel_type`），便于运维排查
3. **Phase 20 - Sentinel 文档**：自动从 errors 包生成 sentinel 列表（导到 README / docs）
4. **Phase 20 - Typed error 区分**：对高频 typed error（如 `apperrors.BusinessError` vs `apperrors.Sentinel` vs ad-hoc）引入明确的 kind 字段

详见：每个 `refactor(19/dN)` commit message 包含完整 commit-level 摘要。
