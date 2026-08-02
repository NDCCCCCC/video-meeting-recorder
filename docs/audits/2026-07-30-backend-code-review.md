# Record V2 后端代码审查报告

**审查日期**：2026-07-30
**审查范围**：`internal/` 下 153 个 Go 源文件（13 子目录） + `cmd/server/app.go`
**代码规模**：约 40,271 行 / 336 个 Go 文件
**Go 版本**：1.25.0
**审查方法**：ripgrep 全库模式扫描（Phase 1，4 个并行 Explore agent）+ 10 个关键文件精读（Phase 2，3 个并行 Explore agent）+ 上下文交叉验证
**审查范围声明**：仅审查 `internal/` 与 `cmd/`，不涉及 `frontend/`、`docs/`、`bin/`、`data/`、`.claude/`、`.planning/`、worktrees 中的代码
**审查方式**：**只读不写**（按用户指定）—— 本报告只描述问题，不修改任何代码

---

## 1. 摘要

### 1.1 整体健康度评分

🟡 **黄灯** — 项目整体纪律性较好（godoc 覆盖率 ~96%，无 `init()` 副作用、无 `context.TODO()` 滥用、业务包内无 viper 直读），属于"局部精修"级别而非"需要大规模重构"级别。但存在 **3 个 HIGH 级别的真 bug** 与 **多个攻击面无网际的安全漏洞**，建议在下个迭代立即修复。

### 1.2 问题总数（按类别 × 严重度）

| 类别 | HIGH | MEDIUM | LOW | 小计 |
|------|-----:|-------:|----:|-----:|
| 🐛 Bug / 逻辑错误 | 2 | 4 | 9 | 15 |
| 🔒 安全漏洞 | 4 | 5 | 6 | 15 |
| ⚡ 性能问题 | 5 | 6 | 5 | 16 |
| 🎨 Go 风格 / 架构 | 2（项目级） | 3（局部） | 5 | 10 |
| **合计** | **13** | **18** | **25** | **56** |

### 1.3 Top 5 最严重问题（按爆炸半径排序）

| # | 严重度 | 位置 | 问题 | 爆炸半径 |
|---|--------|------|------|----------|
| 1 | **HIGH** | `internal/config/config.go:259` + `internal/auth/hlstoken/hls_token.go` 全局 | **SM4/HLS Token 配置全链路漏洞**：Viper `SetEnvPrefix("RECORD")` 与文档 `SM4_SECRET` 不匹配 + 硬编码 fallback `change-me-in-production` + 启动不校验 + HLS secret 默认复用 SM4 + 5 分钟 token 无 replay 防护 + token URL 传播 + 密码 DB 明文 + ParseTaskID/ParseUserID 不验签 | **一处泄漏 = 全链路劫持**：伪造任意 access_token / 签 HLS token 看任意录制 / 解 AD 密文 / 任意 user_id 身份 |
| 2 | **HIGH** | `cmd/server/app.go` 装配遗漏 + `auth/service.go:183/199` + `local_auth.go:149/234/250/273` | **审计装配遗漏**：创建 `authService` 后**从未调用** `SetAuditService(auditService)`，导致 6 个 audit 调用点全部因 `auditLogger == nil` 短路 | 登录失败（密码错、用户不存在、解密失败、用户被禁、频率限制、IP 异常）**100% 不被业务层审计** |
| 3 | **HIGH** | `internal/services/video_recording_task_service.go:786-789` | **`RetryTask` 重算 EndTime 真 bug**：先改了 `task.StartTime`，再用 `task.EndTime.Sub(task.StartTime)`，算出 `oldEnd - newStart`（负数或离谱值），而非计划的"保留时长" | 重试功能**静默损坏**，scheduler 据此停止录制时间全错，运维复测"重试"会发现任务时长异常 |
| 4 | **HIGH** | `internal/huawei/manager.go:80-81` + `internal/huawei/client.go:250` + `models.InputConfig.Password` | **华为 TLS 三重弱点**：硬编码 `InsecureSkipVerify: true` + `MinTLSVersion: 0x0301` (TLS 1.0) + Cipher 含 3DES_EDE_CBC_SHA (SWEET32) + 华为密码 DB 明文存储 | 内网 MITM 攻击者直接中间人所有华为云流量，凭据明文截获 |
| 5 | **HIGH** | `internal/services/video_recording_task_service.go:582-590/975-1006` + 全局 403 处 GORM 缺 `WithContext` | **N+1 + 缺 ctx 数据库操作**：批量删除 100 任务可放大到 600+ 次 DB round-trip；`.Find` 18 处无 Limit；客户端断开/超时无法级联到 SQL | 数据量增长后连接池耗尽，HTTP 超时无法取消 SQL，调度器后台任务占用连接 |

### 1.4 与上次审查的关联

- 上次 `2b4b480 docs(quick-260730-dr8): 补 48 个敏感 GET 端点审计` 主要覆盖**权限可访问性**（HTTP 中间件级）
- 本次审查覆盖**代码内部正确性**（bug / 安全 / 性能 / 风格），与上次互补
- 关联发现：dr8 审查的"公开路由依赖 handler 内校验"（如 `/api/v1/files/download/:token`、`/api/v1/recordings/:id/preview/stream/:file`）与本次 `hlstoken.go` 的"token URL 传播 + 无 replay 防护"形成攻击链

---

## 2. 🐛 Bug / 逻辑错误

按严重度从高到低排列。

### 2.1 HIGH 级

#### [BUG-001] `RetryTask` 重算 EndTime 真 bug

- **文件**：`internal/services/video_recording_task_service.go:786-789`
- **类别**：Bug / 数据正确性
- **触发场景**：用户对已结束/失败的任务点击"重试"操作
- **代码片段**：
  ```go
  task.StartTime = newTriggerTime.Add(time.Duration(task.PreJoinMinutes) * time.Minute)  // ①
  task.EndTime = task.StartTime.Add(task.EndTime.Sub(task.StartTime))                    // ②
  ```
- **问题分析**：第 ② 步 `task.EndTime.Sub(task.StartTime)` 读取的是**已被改写的** `StartTime`，结果是 `oldEnd - newStart`（负数或离谱大的值），而非原计划"保留时长"
- **修复建议**：先保存原时长再改 StartTime：
  ```go
  duration := task.EndTime.Sub(task.StartTime)
  task.StartTime = newTriggerTime.Add(time.Duration(task.PreJoinMinutes) * time.Minute)
  task.EndTime = task.StartTime.Add(duration)
  ```

#### [BUG-002] 8 个 fire-and-forget goroutine 缺 recover

- **文件**：
  - `internal/services/transcription_service.go:911` (cleanupOrphanedOSSFiles, 1h ticker)
  - `internal/auth/sm4_token.go:394` (GracePeriod revoke)
  - `internal/services/storage/file_service.go:280` (async delete)
  - `internal/services/video_recording_task_service.go:313,797` (scheduler sync)
  - `internal/services/video_file_service.go:1585` (zip writer)
  - `internal/huawei/client.go:591` (keep-alive)
  - `internal/middleware/apikey.go:30` (usage log)
- **类别**：Bug / 并发安全
- **触发场景**：goroutine 内部任意 panic（nil 指针、map/slice 越界、类型断言失败等）→ Go runtime fatal → 整个服务进程退出
- **注意**：`api_key.go:81` 的 panic 路径已被 `cmd/server/app.go:519` 的 `gin.Recovery()` 兜底（Phase 2 精读确认），不会真杀进程，但仍应避免在业务代码 panic
- **修复建议**：每个 fire-and-forget goroutine 入口加：
  ```go
  defer func() {
      if r := recover(); r != nil {
          logger.Error("xxx goroutine panicked", zap.Any("recover", r), zap.Stack("stack"))
      }
  }()
  ```

### 2.2 MEDIUM 级

#### [BUG-003] `json.Unmarshal` 错误被吞（6 处）

- **文件**：
  - `internal/middleware/audit.go:47`
  - `internal/models/audit_log.go:136,145,154` (尤其严重 — GetOldData/GetNewData/GetDiffData)
  - `internal/models/notification.go:85,107`
- **类别**：Bug / 错误处理
- **触发场景**：DB 列被人工改坏或迁移失败 → JSON 反序列化报错 → 静默返回 nil → 审计回放数据全无但不告警
- **修复建议**：每处加 `if err != nil { logger.Warn("field parse failed", zap.Error(err)); return defaultValue }`

#### [BUG-004] `_ = ` 显式忽略 error（9 处）

- **文件**：
  - `internal/models/api_key.go:92,112` — IP 白名单解析失败会**默认放行所有 IP**（安全降级）
  - `internal/models/role.go:38`, `internal/models/user.go:104`
  - `internal/services/usb_device_scanner.go:121,336`
  - `internal/services/video_recording_task_service.go:410`（`UpdateTask` 时间比较退化分支：错误信息误导）
  - `internal/auth/sm4_token.go:356`
  - `internal/scheduler/video_scheduler.go:459`
- **触发场景**：`json.Unmarshal` / `cmd.Run` / `RevokeUserSessions` 返回 err 直接被吞，配置文件被人工改坏后无日志无告警
- **修复建议**：改为带日志的 err 忽略：`if err != nil { logger.Warn("...", zap.Error(err)) }`

#### [BUG-005] GORM 全部 403 处调用缺 `.WithContext(ctx)`

- **文件**：全库 GORM 调用 `.Find/.First/.Where/.Create/.Update/.Delete/.Save`（403 处均未用）
- **类别**：Bug / 上下文传播
- **触发场景**：service 方法接收 `ctx` 但传给 ORM 时 `context` 失效；高延迟 SQL 执行时客户端已断开但数据库继续跑，连接池耗尽
- **修复建议**：所有 GORM 调用加 `db.WithContext(ctx)`：
  ```go
  s.db.WithContext(ctx).Find(&tasks)
  ```

#### [BUG-006] `time.Sleep` 不可被 ctx 取消（4 处）

- **文件**：
  - `internal/recorder/coordinator.go:267` (reconnect delay, 10s)
  - `internal/huawei/manager.go:203` (1s 轮询)
  - `internal/scheduler/video_scheduler.go:355` (按分钟延迟)
  - `internal/auth/sm4_token.go:395` (GracePeriod)
- **类别**：Bug / 优雅退出
- **触发场景**：进程 graceful shutdown 时这些 goroutine 仍 sleep 满，导致等待 SIGKILL
- **修复建议**：改为 `time.NewTimer` + `select { case <-ctx.Done(): return; case <-time.After(delay): }`

### 2.3 LOW 级（次要）

#### [BUG-007] `defer mu.Unlock()` 顺序问题：0 处

- 全部 24 处 `defer *.Unlock()` 均在 `Lock()` 之后立即 defer，顺序正确。

#### [BUG-008] `http.ListenAndServe` 无 `http.Server` 包装：0 处

- `cmd/server/app.go:1098,1212` 均用 `&http.Server{...}` + `Shutdown(ctx)` + `ErrServerClosed` 判断，优雅退出已正确实现。

#### [BUG-009] `make(chan)` 容量为 0 用作 buffer：0 处

- 所有 channel 都有合理 buffer（conversion 100、transcription 100、splitting 100、audit 1000、notification 1000、updateChan 1、stopCh 0 仅作信号）。

#### [BUG-010] `select` 无 default 阻塞：0 处

- 全部 25 处 select 至少含 `ctx.Done()`、`default` 或 ticker/超时。

#### [BUG-011] `int64(time.Now().UnixNano())` 拼接（5 处）

- `internal/services/frame_capture_service.go:120`
- `internal/services/storage/file_service.go:587,592,597,629`
- 用于文件名，2262 年前不溢出；高并发下 nanosecond 几乎不会冲突，理论风险

#### [BUG-012] `range` 循环里修改 map：0 严重

- `internal/auth/sm4_token.go:103-105` 循环中 `delete(s.tokenCache, token)` —— Go spec 允许删除当前键，安全

#### [BUG-013] `sync.WaitGroup.Add` 在 goroutine 内部：0

- 全部在 `go func()` 之前调用，顺序正确

#### [BUG-014] nil map 写入：0 严重

- 6 处 `var m map[...]` 全部作为 unmarshal 目标，被 `json.Unmarshal` 隐式 make

#### [BUG-015] 注释与代码错位（api_key.go:134）

- `internal/models/api_key.go:134` 注释写"磁化显示密钥，仅保留前 8 位"，但下方代码是 IP 白名单循环（旧函数残留）

#### [BUG-016] `IsIPAllowed` 不处理 IPv6-mapped IPv4

- `internal/models/api_key.go:127-150`：`net.ParseIP("::ffff:192.0.2.1")` 不等于 `192.0.2.1`，白名单含两种写法会漏判

---

## 3. 🔒 安全漏洞

按严重度从高到低排列。

### 3.1 HIGH 级（必须立即修复）

#### [SEC-001] SM4/HLS Token 配置全链路漏洞

- **位置**：
  - `internal/config/config.go:259` `SetEnvPrefix("RECORD")`
  - `internal/config/config.go:360-361,373-374,584,588` 硬编码默认值
  - `internal/auth/hlstoken/hls_token.go` 全局
- **类别**：安全 / 配置 + 加密 + 重放
- **触发链**：
  1. Viper `SetEnvPrefix("RECORD")` 导致查找 `RECORD_AUTH_SM4_SECRET`
  2. 但 `DEPLOYMENT.md` / `.env.example` / `SECURITY.md` / `BUILD.md` / CI yml 全部写成 `SM4_SECRET` / `HLS_TOKEN_SECRET`（无前缀）
  3. 运维按文档设置 `export SM4_SECRET=...` → viper 找不到 → 回退到 `change-me-in-production`
  4. `setDefaults` 启动不校验 secret 有效性
  5. `HLSTokenSecret` 默认复用 `SM4Secret`（config.go:373-374）
  6. `hlstoken.NewHLSToken` 启动不校验 secret 长度
  7. token 无 jti / nonce / replay 防护，5 分钟可无限复用
  8. token 走 URL query（被 CDN/日志记录）
  9. `ParseTaskIDFromToken` / `ParseUserIDFromToken` 不验签
- **攻击场景**：任意攻击者 `git clone` 仓库读源码 → 算 SM4 密钥 + HLS Token 密钥 → (a) 伪造 SM4-GCM access_token 拿下任意用户身份；(b) 签 HLS token 看任意录制；(c) 解密任何 AD 用户密文密码
- **修复建议**（按优先级）：
  1. `SetEnvPrefix("")` 改为空 或显式 `BindEnv("auth.sm4_secret", "SM4_SECRET")`
  2. 删除 `setDefaults` 中所有密钥默认值，改为 `Environment=="production" && Secret==""` 时 `logger.Fatal`
  3. `Load()` 末尾统一 `ValidateSM4Secret` + `len(secret) >= 32` 启动校验
  4. `NewHLSToken` 启动时 `len(secret) < 32` → log.Fatal
  5. 增加 `jti` 字段 + Redis 防重放；或缩短 duration 到 30 秒 + refresh token 机制
  6. `ParseTaskIDFromToken` 调用方必须先 `Verify()`（强制要求签名校验才会暴露 claims）
  7. m3u8 分片 URL 不复用 token，改用 cookie + SameSite=Strict + 短暂 session

#### [SEC-002] 审计装配遗漏：登录失败 100% 不审计

- **位置**：
  - `cmd/server/app.go` 装配序列（行 557 创建 authService、行 578 创建 auditService，**两者间无注入动作**）
  - `internal/auth/service.go:183,199`
  - `internal/auth/local_auth.go:149,234,250,273`
- **类别**：安全 / 可观测性 + 审计
- **触发链**：创建 `authService` 后从未调用 `SetAuditService(auditService)` → `Service.auditLogger` 和 `LocalAuthenticator.auditLogger` 在生产里永远为 nil → 6 个 audit 调用点的 `if a.auditLogger != nil` 全部短路为 false
- **对比**：`apikeyService.SetAuditService(auditService)` ✓（app.go:598 已调用）
- **实际后果**：登录失败（密码错、用户不存在、解密失败、用户被禁、频率限制、IP 异常）**100% 不被业务层审计**。`middleware/audit.go:AuditLogin` 仍会记录一次"登录"，但只有 HTTP 状态码 200/非 200 二分，**没有具体错误原因字段**——是个审计可观测性硬伤
- **修复建议**：在 `cmd/server/app.go` 创建 `authService` 之后加一行：
  ```go
  authService.SetAuditService(auditService)
  ```

#### [SEC-003] 华为 TLS 三重弱点

- **位置**：
  - `internal/huawei/manager.go:80` `InsecureSkipVerify: true` 硬编码
  - `internal/huawei/manager.go:81` `MinTLSVersion: 0x0301` (TLS 1.0) 硬编码
  - `internal/huawei/client.go:250` Cipher 含 3DES_EDE_CBC_SHA
  - `models.InputConfig.Password` 字段无加密（不经过 SM4 解密）
- **类别**：安全 / TLS + 凭据存储
- **触发链**：硬编码三个弱点 + 华为密码 DB 明文存储 + 端口不匹配时 TLS handshake 失败但登录凭据已通过 TCP 发送
- **攻击场景**：内网 MITM 攻击者（ARP 欺骗 / 伪造交换机）→ 服务到华为终端的 HTTPS 流量被降级到 TLS 1.0 RSA key-exchange → 证书校验关闭 → 攻击者直接中间人，拿到 `WEB_RequestCertificateAPI` 提交的用户名+密码明文（client.go:486-489）→ 利用这些凭据登录其他华为终端进行会议接听/录制劫持
- **修复建议**：
  1. `InsecureSkipVerify` 改为从 `cfg.InsecureSkipVerify` 读取，**生产强制 false**
  2. `MinTLSVersion` 改为 `tls.VersionTLS12` 最低
  3. 去除 3DES cipher，保留 ECDHE 套件
  4. 华为密码 DB 存储先经 `DecryptPasswordECB` 解密后再用（与 AD 走同一路径）
  5. `removeClient` / `Close` 内 `Logout` 改成传入 `ctx`，不再用 `context.Background()`

#### [SEC-004] HLS Token 密钥长度未校验 + 5 分钟无限重放

- **位置**：`internal/auth/hlstoken/hls_token.go:100` 签名前无校验；第 37-96 生成/校验
- **类别**：安全 / 加密 + 重放
- **触发链**：`NewHLSToken` 零校验 + HMAC-SHA256 + 短密钥 → `hashcat -m 1450` 离线爆破 → 拿到 token → 5 分钟内无限重放 m3u8 + .ts 分片
- **攻击场景**：用户 A 在浏览器打开 `?token=...` → 浏览器历史 / CDN 边缘日志 / F12 Network / 服务端 `file_handler.go:127` 主动 `zap.String("token", accessToken)` 写到日志文件 → 攻击者拿到 token → (a) 5 分钟内无限 GET m3u8 拿到未来 .ts 分片；(b) 轮询 `GetHLSPreview` 续期 token，永续观看整段录制
- **修复建议**：
  1. `NewHLSToken` 启动时 `len(secret) < 32` → log.Fatal
  2. 增加 `jti` + 服务端 `used_jtis` 表（或 Redis 防重放）
  3. `ParseTaskIDFromToken` 调用方必须先 `Verify()`
  4. HMAC-SHA256 改用 base64.RawURLEncoding

### 3.2 MEDIUM 级

#### [SEC-005] SQL 字符串拼接（1 处）

- **文件**：`internal/migrations/013_add_ad_fields.go:37`
- **类别**：安全 / SQL 注入（迁移代码）
- **代码**：`db.Exec("ALTER TABLE users ADD COLUMN " + field.column + " " + field.typ)`
- **触发场景**：当前迁移代码中 column 名是写死的，无直接风险；**但若未来从配置/请求读取列名，将存在注入风险**
- **修复建议**：迁移代码也用 GORM 的 `db.Migrator().AddColumn` 或白名单校验

#### [SEC-006] MD5 用于文件指纹

- **文件**：`internal/services/storage/file_service.go:510`
- **类别**：安全 / 弱哈希
- **触发场景**：MD5 文件指纹存在碰撞风险——攻击者可构造两个内容不同但 MD5 相同的文件欺骗去重逻辑
- **修复建议**：改用 SHA-256

#### [SEC-007] CORS 通配符（4 处）

- **文件**：
  - `internal/handlers/file_handler.go:141`
  - `internal/handlers/video_file_handler.go:160, 410`
  - `internal/handlers/video_recording_task_handler.go:718`
- **触发场景**：任意网站可跨域读取已签发的 HLS 下载 token，绕过 Same-Origin Policy
- **修复建议**：从配置读取允许的 origin 列表（至少按域名精确匹配）

#### [SEC-008] CSRF 全局缺失

- **文件**：`cmd/server/app.go:700-940` 所有状态变更接口
- **触发场景**：当前用 Authorization Bearer，CSRF 风险低；**但若未来引入 Cookie 认证将立即出现 CSRF 风险**
- **修复建议**：保留配置项 `csrf_enabled: bool`，未来启用

#### [SEC-009] 敏感信息日志（2 处）

- **文件**：
  - `internal/handlers/file_handler.go:127` 完整 token 打日志
  - `internal/huawei/client.go:351` Huawei 完整响应体 `Debug` 级别日志
- **触发场景**：日志被采集后 token 可被任意持有者复用；Huawei 响应可能含敏感元数据
- **修复建议**：`file_handler.go:127` 改为只打印 token 末 4 位；`client.go:351` 改为脱敏后再 Debug

#### [SEC-010] Token 走 URL query（CDN 日志泄露）

- **文件**：`internal/middleware/auth.go:137-138`
- **类别**：安全 / 凭据传播
- **触发场景**：CDN / Nginx / 反向代理 / 浏览器历史会记录完整 token
- **当前缓解**：仅在 `isVideoDownloadEndpoint` 命中时允许 query token（白名单子串匹配）
- **修复建议**：把白名单从"子串匹配"改成精确前缀列表（`strings.HasPrefix` + 集合）；或者考虑用短期 cookie + SameSite=Strict

### 3.3 LOW 级

#### [SEC-011] SHA256 截断派生 SM4 密钥

- **文件**：`internal/auth/sm4_token.go:113` `sha256.Sum256(secret)[:16]`
- **类别**：安全 / 弱派生
- **触发场景**：哈希截断损失熵；虽比明文好，但与 SM4 规范不严格匹配
- **修复建议**：直接 hex decode 32 字符 secret（已与前端 sm4.ts 对齐）

#### [SEC-012] 文件上传未校验 MIME magic bytes

- **文件**：`internal/services/storage/file_service.go:468-500`
- **类别**：安全 / 上传校验
- **触发场景**：扩展名可被伪造（如 `evil.mp4.exe`）；纯扩展名白名单易绕过
- **修复建议**：用 `http.DetectContentType` 校验真实 MIME

#### [SEC-013] SSRF 风险面（4 个出站 URL）

- **文件**：`internal/services/tingwu_client.go` + `internal/huawei/client.go`
- **触发场景**：当前 URL 都是配置固定值，**未发现用户输入直接传入 `http.Get`**，风险面窄
- **修复建议**：未来若添加"通过 URL 拉取转录"功能必须补白名单

#### [SEC-014] 认证旁路：公开路由依赖 handler 内校验

- **文件**：`cmd/server/app.go:840-841, 884, 887`
- **公开路由**：`/api/v1/files/download/:token`、`/api/v1/recordings/:id/preview/stream/:file`、`/api/v1/ppts/:id/slides/:resolution/:filename`
- **触发场景**：Token 泄露即未授权访问；`ServeHLSStream` 暴露 token 在 URL 中易被 CDN/日志记录
- **修复建议**：每条公开路由必须有专门的 `audit` + `warn_unusual_access` 装饰器

#### [SEC-015] `gin.Context.GetString` 无 `, ok` 检查

- **文件**：`internal/handlers/system_handler.go:148`, `internal/middleware/audit.go:62-63,136,180`
- **类别**：安全 / 类型安全
- **触发场景**：用 `GetString` 直接取 `user_id` 未校验 `exists`，得到空串写入审计 → 阻碍攻击溯源
- **修复建议**：所有 `c.Get(...)` 后做类型断言 `if id, ok := v.(uint); ok`

---

## 4. ⚡ 性能问题

按严重度从高到低排列。

### 4.1 HIGH 级

#### [PERF-001] N+1 查询（3 处）+ 3 重 Preload 隐藏 N+1

- **文件**：
  - `internal/services/video_recording_task_service.go:582-590` (DeleteTask)
  - `internal/services/video_recording_task_service.go:975-1006` (BatchDeleteTasks，**比单删更危险**)
  - `internal/services/video_recording_task_service.go:999-1006` (ClearStuckTasks)
  - `internal/handlers/admin_handler.go:273,276`
  - **隐藏 N+1**：96/180/301/438 行的 `Preload("InputConfig").Preload("TaskInputConfigs").Preload("Creator")` —— ListTasks 每页 100 条 → 1 主 + 3×100 = 301 次查询
- **最坏情况**：N=100 任务 × M=5 配置 → 600+ 次 DB round-trip
- **修复建议**：
  ```go
  // DeleteTask: 先 Pluck 再 IN 批量 UPDATE
  ids := []uint{}
  s.db.Model(&models.TaskInputConfig{}).Where("task_id = ?", taskID).Pluck("input_config_id", &ids)
  s.db.Model(&models.InputConfig{}).Where("id IN ?", ids).Update("is_locked", false)
  ```

#### [PERF-002] `.Find` 无 Limit（18 处）

- **文件**（部分列出）：
  - `internal/scheduler/video_scheduler.go:311,360,566,1208`
  - `internal/services/role_service.go:199,233`
  - `internal/services/video_file_service.go:364,1046,1125,1459,1551`
  - `internal/services/video_recording_task_service.go:476,554,584,819,899,964,999`
- **触发场景**：数据量增长后放大 SQL 扫描、网络传输、Go slice 内存
- **修复建议**：管理后台列表加分页（`?page=1&page_size=20`）；后台清理任务保留全量加载但加注释说明

#### [PERF-003] 27 个长 DB 操作缺 `WithContext(ctx)` 超时控制

- 与 [BUG-005] 同一根因但属性能维度。**27 个 `.Find` 均未在调用链附近显式使用 GORM `WithContext`**。
- **触发场景**：数据库拥塞、锁等待、网络异常时，请求/后台任务长期占用连接和 goroutine，拖垮连接池
- **修复建议**：所有 GORM 调用加 `.WithContext(ctx)`；service 层构造时加 `ctx, cancel := context.WithTimeout(parent, 30*time.Second)` 兜底

#### [PERF-004] 锁粒度过粗（3 处）

- **文件**：
  - `internal/recorder/coordinator.go:93-123`（含 `os.MkdirAll` 文件 I/O + `cmd.Start()`）
  - `internal/huawei/manager.go:107-112`（含 `client.Logout()` 出站 HTTP POST）
  - `internal/huawei/manager.go:118-127`（含 `client.Logout()`）
- **触发场景**：慢磁盘、进程启动、网络请求期间，其他无关任务也无法访问服务
- **修复建议**：`StartRecordingWithConfig` 改 "先创建所有资源（无锁）+ 最后一次 `c.mu.Lock(); defer Unlock()` 完成 `processes[processKey] = ...`"

#### [PERF-005] Gin handler 同步做重 IO/批处理（3 处）

- **文件**：
  - `internal/handlers/admin_handler.go:263-278`（管理迁移 handler 同步全量读取并逐条迁移）
  - `internal/handlers/transcription_handler.go:232`（转录 handler 直接批量查询）
  - `internal/handlers/video_recording_task_handler.go:708-735`（HLS handler 同步 Stat、整文件 ReadFile、字符串重写）
- **触发场景**：handler goroutine 被数据库批处理或文件 IO 占据；大迁移和较大 m3u8 增加请求延迟、内存峰值
- **修复建议**：handler 接到请求后立即返回 202 + task_id，异步 worker 处理

### 4.2 MEDIUM 级

#### [PERF-006] goroutine 泄漏（1 处）

- **文件**：`internal/huawei/client.go:591-608`（keep-alive goroutine 依赖传入 context）
- **触发场景**：调用方传入长期存活的 context 且未调用停止逻辑，ticker goroutine 一直保留
- **修复建议**：每个客户端都有明确 Stop/Close，监控 goroutine 数量

#### [PERF-007] 高频分配缺少 `sync.Pool`（4 处）

- **文件**：
  - `internal/middleware/audit.go:43,46`（请求体读取）
  - `internal/services/conversion_service.go:366`
  - `internal/services/snapshot_service.go:295`
  - `internal/services/frame_capture_service.go:128,199`（图片整文件读取）
- **修复建议**：用 `sync.Pool` 复用 buffer

#### [PERF-008] 函数体内正则重复编译（6 处）

- **文件**：
  - `internal/auth/password_validator.go:97,126-129`（密码强度检查一次编译 4 个正则）
  - `internal/config/config.go:204`
- **触发场景**：每次验证都重新解析构建 regexp
- **修复建议**：所有正则提到包级 `var xxxRe = regexp.MustCompile(...)`

#### [PERF-009] `interface{}` 反序列化（6 处）

- **文件**：
  - `internal/middleware/audit.go:41,47`
  - `internal/config/config.go:278,289`
  - `internal/models/audit_log.go:133-153`
  - `internal/services/usb_device_scanner.go:95,167,312,388`
- **触发场景**：大 JSON/YAML 会创建大量 map、装箱值、类型断言
- **修复建议**：可能的话用类型化 struct 替代 `map[string]interface{}`

#### [PERF-010] `coordinator.go:218-247` 锁释放后继续读 process 字段

- **类别**：性能 + 并发安全
- **触发场景**：race detector 必报；`process.ReconnectCount` 自增（270 行）也未受任何锁保护
- **修复建议**：`process.*Reconnect*` 与 `process.Status` 一起包到同一锁内，或用 `atomic.Int32`

#### [PERF-011] `coordinator.go:705` hlsDeleteThreshold 缺配置校验

- `cfg.FFmpeg.HLSListSize == 0` → `hlsDeleteThreshold = 1`；负数 → FFmpeg 行为未定义
- **修复建议**：`buildRecordingCommand` 加配置合法性检查

### 4.3 LOW 级

#### [PERF-012] 循环字符串拼接（1 处）

- `internal/recorder/coordinator.go:488-496` — 实际开销 < 10µs/次，**Phase 1 列 HIGH 是过度**
- **修复建议**：低优先，改用 `strings.Builder`

#### [PERF-013] 同一函数重复 `time.Now()`（1 处）

- `internal/services/conversion_service.go:251,268` — 单次成本不高，但产生不一致时间基准

#### [PERF-014] 无缓冲 channel 误用（3 处）

- `internal/services/audit/audit_log_service.go:138` / `notification/notification_service.go:69` / `rate_limiter.go:45`
- 实际是 close-only 信号，性能风险很低

#### [PERF-015] 数据库连接池配置缺失

- `internal/` 内未发现 `sql.Open` / `gorm.Open` — 连接初始化可能位于目录外
- **建议**：在 `cmd/server/app.go` 启动期确认 `SetMaxOpenConns` / `SetMaxIdleConns` / `SetConnMaxLifetime` 已设置

#### [PERF-016] `time.After` 在 select 中（2 处）

- `internal/services/conversion_service.go:445` + `transcription_service.go:784`
- 频次低（每任务最多几次），timer 在 select 完成后由 GC 回收
- **建议**：低优先，改写时顺手换 `time.NewTimer(delay); defer t.Stop()`

---

## 5. 🎨 Go 风格 / 架构

### 5.1 项目级系统性问题（2 个）

#### [STYLE-001] 错误处理风格不统一

- **现状**：
  - `errors.New("中文")` 占 168 个（业务 153 个）—— 无包装，调用方无法 `errors.Is`
  - `fmt.Errorf` 474 个但仅 59% (278) 用 `%w` 包装
  - 5 处 `err.Error() == "中文"` 在 handler 反向匹配 service 抛出的硬编码字符串（违反 Go 惯例）
  - 13 处直接 `err == gorm.ErrRecordNotFound`
- **已有约定未贯彻**：
  - `internal/errors/errors.go:11-27` 定义了 `ErrNotFound / ErrInvalidInput / ErrUnauthorized / ErrForbidden`
  - `internal/common/interfaces.go:194` 定义了另一套指针型 `BusinessError`
  - `internal/auth/ad_auth.go:21` 有 `ErrADUserNotRegistered` 成功范例
- **修复建议**：从 `internal/errors` 入手统一一处，让所有 service 都 import 它；handler 层用 `errors.Is` 分流回 400/404/500

#### [STYLE-002] `auth → services/audit` 依赖方向（误报"循环"）

- **真实情况**：`services/audit/audit_log_service.go` **未 import `auth`**（grep `auth.` 0 命中），反向依赖不存在，**没有真正的循环 import 风险**
- **Phase 1 把"非对称单向依赖"误报成"循环候选"**（误报纠正）
- **真正的关联问题**：[SEC-002] 审计装配遗漏（`authService.SetAuditService(auditService)` 从未调用）—— 已在 SEC-002 中处理

### 5.2 局部架构问题（MEDIUM，3 个）

#### [STYLE-003] 接口定义在实现方（3-4 处）

- `internal/services/conversion_service.go:20`（ConversionService + FFmpegConversionService 同文件）
- `internal/services/storage/driver.go:11`（StorageDriver，impl 在同子包）
- `internal/auth/ad_config.go:70`（Authenticator，impl 在同包）
- `internal/common/interfaces.go:9`（Service + BaseService 同包）
- **正面对比**：`internal/scheduler/video_scheduler.go` 里的 `SchedulerInterface/ConversionServiceInterface` 是在 scheduler (消费者) 定义，**正确**
- **修复建议**：将接口移到消费方包

#### [STYLE-004] 中间件 `GetUserID` 等零值语义

- `internal/middleware/auth.go:13-62` —— `GetUserID` 等 6 个助手函数无 user_id 时返回零值
- **后果**：userID=0 是合法值，无法与"未认证"区分。叠加 audit 中间件 `c.Next()` 之后读取的时序坑，未授权访问被审计成"user_id=0 的合法用户"，污染所有 user 维度统计
- **修复建议**：助手函数改为返回 `(uint, bool)`，调用方显式区分"未注入"和"user 0"

#### [STYLE-005] 类型断言未守 `ok`

- `internal/middleware/auth.go:15/24/32/40/48` 共 5 处 `c.Get(...)` 后直接 `.()` 不带 `, ok`
- 触发 panic 后 `gin.Recovery()` 兜底但请求 500
- **修复建议**：与 `audit.go:85-94` 已有的 `, ok` 写法对齐

### 5.3 局部小毛病（LOW，5 个）

#### [STYLE-006] `panic` 替代 `error`

- `internal/models/api_key.go:81` — `crypto/rand.Read` 在现代 OS 上几乎不会失败，但模型层 panic 不优雅
- **修复建议**：把签名改为 `(string, error)`，BeforeCreate hook 内处理错误

#### [STYLE-007] `switch` 缺 default

- `internal/middleware/apikey.go:253` — `switch requiredScope { case ScopeRead: ... case ScopeWrite: ... }` 缺 default，未知 scope 静默通过
- **修复建议**：加 `default: hasPermission = false`

#### [STYLE-008] `defer conn.Close()` 缺 nil 防御

- `internal/auth/ad_auth.go:68,390` + `internal/auth/ad_validator.go:44` —— 实际安全（connectAD 失败时先 return），但建议加 `if conn != nil { defer conn.Close() }` 强化防御

#### [STYLE-009] 包名冗余（133 处 Get*）

- 133 个 `GetXxx` 方法名违反 Go 习惯（Go 推崇直接字段）
- 包名冗余：`huawei.HuaweiClient` → `huawei.Client`、`notification.NotificationService` → `notification.Service`
- **修复建议**：低优先，按文件逐步改名

#### [STYLE-010] godoc 缺失（8 处）

- `internal/handlers/split_handler.go:13,20`
- `internal/handlers/admin_handler.go:27`
- `internal/services/splitting_service.go:30,49`
- `internal/services/snapshot_service.go:19,28`
- `internal/auth/ad_validator.go:18`
- **整体覆盖率 ~96%**，显著高于同类项目；仅上述 8 处需补注释

### 5.4 项目优秀之处（应维持）

- ✅ **`init()` 函数做副作用：0 处** — 干净
- ✅ **`context.TODO()` 滥用：0 处** — 干净
- ✅ **业务包内 viper 直读：0 处** — 配置读取隔离良好
- ✅ **XXE / XML 外部实体：0 处** — 未使用 `xml.Unmarshal`
- ✅ **不安全随机数：0 处** — 密码/token/盐值生成均使用 `crypto/rand`
- ✅ **godoc 覆盖率 ~96%** — auth 94%/100%, handlers 89%/96%, services 95%/99%, models 100%/100%, middleware 100%/100%

---

## 6. 附录

### 6.1 审查方法

| 阶段 | 工具 | 覆盖 |
|------|------|------|
| Phase 1 - 模式扫描 | ripgrep + 4 个并行 Explore agent | 全库 153 个 Go 文件，按 bug/security/perf/style 4 个维度 |
| Phase 2 - 关键文件精读 | Read + Grep + 3 个并行 Explore agent | 10 个 HIGH 优先级文件完整阅读 |
| Phase 3 - 汇总输出 | 人工整合 + 分类 | 56 个最终发现 |

### 6.2 优先级修复清单

**P0（必须立即修复，影响安全或数据正确性）**：
1. [SEC-002] `cmd/server/app.go` 加 `authService.SetAuditService(auditService)` 一行
2. [BUG-001] `video_recording_task_service.go:786-789` 修复 RetryTask EndTime 计算
3. [SEC-001/1] `config.go` 改 `SetEnvPrefix` 或显式 BindEnv
4. [SEC-001/2] `config.go` 删除硬编码密钥默认值，生产环境 secret 为空时 logger.Fatal

**P1（应在下个迭代修复）**：
5. [SEC-001/4-7] HLS Token 启动校验 + jti 防重放 + ParseToken 强制验签
6. [SEC-003] 华为 TLS 全部去掉硬编码弱点
7. [SEC-004] HLS Token 长度校验 + replay 防护
8. [PERF-001] N+1 改批量查询/更新
9. [BUG-002] 8 个 fire-and-forget goroutine 加 recover
10. [PERF-003] GORM 全面加 `WithContext(ctx)`

**P2（应长期清理）**：
- [STYLE-001] 错误处理风格统一
- [STYLE-003] 接口定义位置修正
- [PERF-007/008/009] sync.Pool / 正则提到包级 / 类型化 struct
- [BUG-003] 6 处 `_ = json.Unmarshal` 加日志
- [SEC-007] CORS 通配符改白名单

### 6.3 后续建议

1. **建立 lint 规则**（`.golangci.yml`）：
   - `errcheck` 启用（捕获 `_ =` 忽略）
   - `gosec` 启用（捕获 `InsecureSkipVerify` / `math/rand`）
   - `exportloopref` 启用（虽然 Go 1.22+ 自动修复，仍建议显式）
   - `bodyclose` 启用（捕获 HTTP 响应未关闭）
   - `contextcheck` 启用（捕获 ctx 未透传）

2. **CI 集成**：
   - 把审查脚本化（参考 `gsd-code-reviewer`）
   - 每次 PR 触发快速扫描（`rg` 模式），阻断 HIGH 问题合并

3. **测试覆盖**：
   - 当前未审计测试覆盖（除 `go.mod` 修改历史外）
   - 建议下个迭代先补 `auth/`、`services/video_recording_task_service.go`、`huawei/`、`hlstoken/` 的单元测试

4. **配置管理**：
   - 建议引入 `koanf` 或类似强类型配置库，避免 viper 字符串 key 散落各处
   - 启动期配置校验应该集中且强制

5. **审计体系**：
   - 把 `audit` 从 `internal/services/audit` 挪到 `internal/audit` 或 `pkg/audit` 中性包
   - 引入结构化审计 schema（`audit_event_type` 枚举），便于后续 BI 分析

### 6.4 未审查清单（明确边界）

- ❌ 前端代码（`frontend/`，由前端审查负责）
- ❌ 文档（`docs/`、`*.md`）
- ❌ 配置文件（`config.yaml.example`、`.env.example`、CI yml）
- ❌ vendor / 第三方依赖
- ❌ 数据库迁移 SQL 文件（除 `migrations/*.go` 内的 Go 代码）
- ❌ `bin/`、`data/`、`certs/`、`.claude/`、`.planning/`
- ❌ `.claude/worktrees/` 下其他 worktree 中的代码

### 6.5 与已有文档的交叉引用

| 文档 | 关联点 |
|------|--------|
| `SECURITY.md` | [SEC-001] 事件 #1（config.yaml git filter-repo）+ [SEC-002] 装配遗漏 |
| `SHARED_VIEWER_PERMISSIONS_AUDIT.md` | dr8 GET 端点审计 → [SEC-014] 公开路由依赖 handler 内校验 |
| `BUILD.md` / `DEPLOYMENT.md` | [SEC-001/1] 文档与代码不一致（Viper 前缀错配） |
| 上次 commit `2b4b480 docs(quick-260730-dr8): 补 48 个敏感 GET 端点审计` | 权限可访问性 → 本次代码内部正确性 |

---

**审查完成时间**：2026-07-30
**审查方式**：READ-ONLY（未修改任何代码）
**审查结论**：🟡 **黄灯** — 局部精修级别，但有 3 个 HIGH 真 bug 与多个无网际攻击面，建议在下次迭代立即修复 P0 项目。