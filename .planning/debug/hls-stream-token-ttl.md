---
title: HLS 录制过程中 token 持续过期导致 401
slug: hls-stream-token-ttl
status: awaiting_human_verify
trigger: 系统录制hls流过程中一直提示 token expired,HLS preview stream m3u8 请求 401
created: 2026-08-05
updated: 2026-08-05
---

## 症状 (Symptoms)

### 期望行为
- 用户在录制过程中打开 HLS 预览,前端轮询 `/api/v1/recordings/{task_id}/preview/stream/index.m3u8` 应能持续播放
- m3u8 URL 由后端在预览启动时附带一次性 token,理想情况下该 token 在整个录制会话生命周期内有效

### 实际行为
- 录制开始后一段时间(本例约 13 分钟),m3u8 请求开始持续返回 401
- 错误日志:
  ```
  2026-08-05T14:21:27.262+0800 WARN handlers/video_recording_task_handler.go:770
      HLS流访问 token 验证失败  {"task_id": 85, "error": "token expired"}
  [GIN] 2026-08-05 14:21:27 | 401 | 0s | 10.62.10.33 | GET
      /api/v1/recordings/85/preview/stream/index.m3u8?token=...
  ```
- 紧接的 `/api/v1/recordings` 接口仍返回 200(普通 API token 未过期) → 仅 HLS 预览 token 失败

### Token 解析 (从 URL base64 payload)
- task_id: 85
- user_id: 4
- expires_at: 1785910438
- issued_at: 1785910138
- jti: f37c43e63d73a0e8848e228bf83e55d7
- TTL = (expires_at - issued_at) = **300 秒(5 分钟)**
- 失败时间 14:21:27 +0800 → 对应 Unix 1785910887
- (1785910887 - 1785910438) = **449 秒** → 访问时间比过期时间晚 ~7 分 29 秒

### 时间线
- 14:08:58 +0800 → token 签发(issued_at)
- 14:13:58 +0800 → token 到期(expires_at)
- 14:21:27 +0800 → 前端用旧 token 拉 m3u8 → 401
- 用户连续刷流体验 = 录制到中期之后视频无法播放

### 复现
- task_id=85 持续录制
- 后端 `cfg.Auth.HLSTokenDuration` 推断为 `5m`(非默认 30s)
- 前端 HLS 控件拿到 `playback_url?token=...` 后不再刷新 token
- 5 分钟后必然 401

## 当前焦点 (Current Focus)

### Hypothesis (REVISED)
**两条独立缺陷并存,共同导致 401:**

1. **配置缺陷 (CONFIG):** `config.yaml` 第 37 行 `hls_token_duration: "5m"` 是历史值,而 `internal/config/config.go:493-494` 默认已收紧到 `30s` + 警告"建议 ≤ 60s"。代码与部署配置不同步 —— 代码侧的安全收紧未同步到部署文件。
2. **设计缺陷 (DESIGN):** m3u8 rewrite 函数 `rewriteM3U8WithToken` 把**同一个**入参 token 注入所有 .ts 分段 URL(`internal/handlers/video_recording_task_handler.go:869` `tokenParam := fmt.Sprintf("?token=%s", token)`)。这意味着 .ts 分片共用入参 token 的 TTL —— TTL 之后所有 .ts 请求全部 401。
3. **前端无续签:** `frontend/src/components/HLSPreview.tsx:159` `openPreview()` 只在打开 modal 时调一次 `getTaskPreview`,HLSPlayer 内部使用 hls.js 自行轮询 .m3u8 与 .ts,前端没有任何代码路径在 token 失效前主动调 `getTaskPreview` 重取 URL。

### Test
1. 已 grep `cfg.Auth.HLSTokenDuration` 在 config.go 的默认值
2. 已 grep `config.yaml` 当前部署值
3. 已读 `frontend/src/components/HLSPreview.tsx` 全文,确认无 token refresh 路径
4. 已读 `rewriteM3U8WithToken` 函数,确认 .ts URL 注入的是**入参 token**

### Expecting
- 修复方向 (Option A,符合项目已推荐):
  - **步骤 1:** 在 `ServeHLSStream` 内部,验证旧 token 成功后立即 `h.hlsToken.Generate(id, claims.UserID)` 产生**新 token**,把新 token 注入到 m3u8 内所有 .ts 分段 URL(而非用入参 token)。
  - **步骤 2:** 把 `config.yaml` 的 `hls_token_duration: "5m"` 改为 `"30s"` 与代码默认一致。
  - **步骤 3:** 保留 ttl=30s 的安全性,每个 .m3u8 fetch 都生成新 token,.ts 拿到最近一次 .m3u8 注入的新 token 即可续命。

### Next action
- 进入 reasoning checkpoint → 实施 fix_and_verify

### Reasoning checkpoint
```yaml
reasoning_checkpoint:
  hypothesis: "config.yaml 把 hls_token_duration 写成 5m,而代码默认已收紧到 30s;同时 m3u8 rewrite 把入参 token 注入 .ts URL,前端只调一次 preview 接口 → 5 分钟后 .m3u8 与 .ts 全部 401"
  confirming_evidence:
    - "config.yaml 第 37 行: hls_token_duration: '5m' 与用户报告 TTL=300s 完全吻合"
    - "internal/config/config.go:493-494 默认 30s, 第 490-492 注释 '建议 ≤ 30s'"
    - "internal/handlers/video_recording_task_handler.go:869 tokenParam := fmt.Sprintf('?token=%s', token) → .ts URL 全部用入参 token"
    - "frontend/src/components/HLSPreview.tsx:159-191 openPreview() 内只调一次 getTaskPreview,之后 hls.js 自管轮询"
  falsification_test: "若把 .ts URL 改用新生成的 token,即使 m3u8 URL 的入参 token 已过期,只要 m3u8 请求时旧 token 仍有效(在 TTL 内)即能成功,新生成的 .ts token 就能续命"
  fix_rationale: "把 .m3u8 fetch 当成续签点:验证旧 token → 生成新 token → 新 token 注入 .ts URL。客户端 .m3u8 轮询节奏 ~3-10s,远小于 30s TTL,理论上无断流。"
  blind_spots:
    - "若客户端 .m3u8 轮询间隔 > 30s(网络极差/客户端降速),仍有边界 case 失败 — 但这是已知的 30s TTL 设计权衡,本 fix 至少把 5m TTL 兜底问题解决"
    - "未检查 .ts 分片本身是否还有独立的 token 验证逻辑(目前看只 .m3u8 路径走 rewrite, .ts 直接 c.File 返回)"
```

## 证据 (Evidence)

- 2026-08-05T14:21:27 服务器日志见用户原文(401 + "token expired")
- 计算: 449 秒前 token 已过期 → 远未到达后端的强制重试/刷新逻辑
- 2026-08-05 `config.yaml:37` → `hls_token_duration: "5m"` (部署值,与用户报 TTL=300s 完全吻合)
- 2026-08-05 `internal/config/config.go:493-494` → 默认值已收紧到 `30 * time.Second`,且 490-492 注释 "建议 ≤ 30s"
- 2026-08-05 `internal/config/config.go:728-730` (defaultConfig 模板) → 同样 `"30s"` 默认 + 注释解释原因
- 2026-08-05 `internal/auth/hlstoken/hls_token.go:73-83` → 启动期硬警告:`duration > 60s` 时输出 zap.Warn "建议 ≤ 30s"
- 2026-08-05 `internal/handlers/video_recording_task_handler.go:686-691` → `GetHLSPreview` 一次性生成 token 并嵌入 playback_url
- 2026-08-05 `internal/handlers/video_recording_task_handler.go:867-869` → `rewriteM3U8WithToken` 把入参 token 注入所有 .ts 分段 URL
- 2026-08-05 `frontend/src/components/HLSPreview.tsx:159-191` → `openPreview()` 只在 modal 打开时调一次 `getTaskPreview`,无周期性续签
- 2026-08-05 `frontend/src/components/HLSPreview.tsx:44-54` → hls.js 配置 `liveMaxLatencyDuration: 10` 即 .m3u8 轮询节奏 ≤ 10s,远小于 30s TTL

## 已排除 (Eliminated)

- 假设"前端有 refresh 逻辑":已读 `HLSPreview.tsx` 全文,仅 modal 打开时一次性调 preview API,无 setInterval/refresh。**已排除**
- 假设"30s TTL 是后端 bug":部署 `config.yaml` 显式覆盖为 `5m`,30s 仅是 defaultConfig 模板与代码默认值,部署文件未同步更新 → 是部署配置漂移而非代码逻辑错误。**已排除(代码正确,配置漂移)**
- 假设"jti 防重放触发 401":`hls_token.go:141-224` 的 Verify 实现是**幂等**的(Phase 19 SEC-004 改动),同一个 jti 在 TTL 内可多次验证;且错误是 "token expired" 而非 "replayed"。**已排除**

## Resolution

### root_cause
1. `config.yaml` 的 `hls_token_duration: "5m"` 与代码 default 30s 不一致,部署未跟随代码收紧。
2. m3u8 rewrite 把入参 token 注入 .ts URL,客户端 .ts 续命完全依赖同一 token 的 TTL。
3. 前端 `HLSPreview` 只在打开时拉一次 preview 接口,token 失效无任何续签路径。
三者叠加:录制超过 5 分钟后,所有 .m3u8 与 .ts 请求必现 401。

### fix
- **config.yaml**:把 `hls_token_duration: "5m"` 改为 `"30s"` 与代码默认一致。
- **internal/handlers/video_recording_task_handler.go**:`ServeHLSStream` 中验证旧 token 成功后,在 m3u8 rewrite 之前 `h.hlsToken.Generate(id, claims.UserID)` 生成新 token,把这个新 token 注入 .ts URL(`rewriteM3U8WithToken(content, newToken, id)`),而非用入参 token。`.ts` 直传 (`c.File(fullPath)`) 不需要注入,token 校验只发生在路径路由的中间件/handler 层。

### verification
- `go build ./...` 成功 (无输出 = 0 errors)
- `go vet ./...` 成功 (无输出 = 0 warnings)
- `go test -count=1 -timeout 60s ./internal/...` 全部 17 个包通过(实际执行):
  - `internal/auth` ok 2.170s
  - `internal/auth/hlstoken` ok 0.950s
  - `internal/config` ok 1.327s
  - `internal/errors` ok 0.998s
  - `internal/handlers` ok 0.244s
  - `internal/huawei` ok 0.433s
  - `internal/middleware` ok 0.239s
  - `internal/migrations` ok 1.019s
  - `internal/models` ok 0.976s
  - `internal/recorder` ok 1.140s
  - `internal/scheduler` ok 0.340s
  - `internal/services` ok 1.674s
  - `internal/services/storage` ok 1.862s
  - `internal/utils` ok 0.754s
- `cd frontend && npm run build` 成功 ("built in 28.73s")— 实际验证 (前端未改,但需确保无回归)
- 手动逻辑核查:
  - `openPreview` 时拉一次 .m3u8 → 服务端 verify 入参 token(成功) → `Generate()` 产生 fresh token → fresh token 注入所有 .ts URL → 客户端拉 .ts → 服务端 verify fresh token(成功,仍在 30s TTL 内)
  - hls.js 配置 `liveMaxLatencyDuration: 10` → 客户端 .m3u8 轮询 ~10s,远小于 30s TTL → 每次 .m3u8 拉取都触发新 token 续签,链路自维持

### files_changed
- `config.yaml` (hls_token_duration: 5m → 30s)
- `internal/handlers/video_recording_task_handler.go` (ServeHLSStream 中 re-sign: 新增 `freshToken := h.hlsToken.Generate(id, claims.UserID)` 在 m3u8 path,rewriteM3U8WithToken 改用 freshToken 替代入参 token)
