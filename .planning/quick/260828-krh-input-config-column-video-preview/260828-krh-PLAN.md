---
title: 输入配置列表为每一项添加视频画面预览（按需 ffmpeg 单帧抓取）
type: quick-fix
status: in-progress
created: 2026-08-28
plan: 260828-krh-01
phase: quick-260828-krh
wave: 1
depends_on: []
autonomous: false
files_modified:
  - internal/services/input_preview_service.go
  - internal/services/input_preview_service_test.go
  - internal/handlers/input_config_handler.go
  - cmd/server/app.go
  - frontend/src/api/input-config.ts
  - frontend/src/api/__tests__/input-config-preview.test.ts
  - frontend/src/pages/system/input-configs/components/InputPreviewCell.tsx
  - frontend/src/pages/system/input-configs/index.tsx
user_setup: []

must_haves:
  truths:
    - "用户在输入配置列表对某一行点击『预览』后，该行显示该输入源的画面帧（USB 摄像头 / RTSP/RTMP/SRT/HLS 流 / 华为终端 RTSP）"
    - "画面加载后可通过刷新按钮手动更新，或开启『自动』开关周期性更新（默认关闭，间隔 10s）"
    - "配置了不可达源或未配置任何可预览源时，单元格显示明确错误提示而不是无限转圈或空白"
    - "预览请求走前端 authedFetch（复用 401 单飞状态机）与后端 api 组 MultiAuth 中间件，无匿名访问"
    - "页面打开不为所有行自动拉流：只有用户主动点击的行才发起抓帧，后端全局并发上限 2 + 单次 10s 超时"
  artifacts:
    - path: internal/services/input_preview_service.go
      provides: InputPreviewService —— 源解析（stream/usb/huawei 三类）+ ffmpeg 单帧抓取 + 并发信号量 + 超时
    - path: internal/services/input_preview_service_test.go
      provides: 源解析 / argv 构建 / NotFound / 无源 / 真实 ffmpeg lavfi 出 JPEG 共 5 组测试
    - path: internal/handlers/input_config_handler.go
      provides: GetConfigPreview handler（返回 image/jpeg 字节 + Cache-Control: no-store）
    - path: frontend/src/api/input-config.ts
      provides: getInputConfigPreview(id) → Blob（走 authedFetch）
    - path: frontend/src/pages/system/input-configs/components/InputPreviewCell.tsx
      provides: 懒加载预览单元格组件（idle/loading/ok/error 四态 + 手动刷新 + 自动刷新开关）
    - path: frontend/src/pages/system/input-configs/index.tsx
      provides: 列表新增『预览』列
  key_links:
    - from: frontend/src/pages/system/input-configs/components/InputPreviewCell.tsx
      to: GET /api/v1/input-configs/:id/preview (cmd/server/app.go inputConfigs 组)
      via: authedFetch → blob → URL.createObjectURL → <img>
      pattern: "input-configs/.*\\/preview|authedFetch"
    - from: internal/handlers/input_config_handler.go (GetConfigPreview)
      to: services.InputPreviewService.CapturePreview
      via: handler 构造函数第 5 参注入 + c.Data image/jpeg
      pattern: "CapturePreview|image/jpeg"
    - from: internal/services/input_preview_service.go
      to: a.config.FFmpeg.Path (internal/config/config.go:608，默认 ./bin/ffmpeg)
      via: NewInputPreviewService(a.db, a.logger, a.config.FFmpeg.Path)
      pattern: "FFmpeg\\.Path"
---

# 输入配置列视频画面预览

## 现状诊断（代码证据）

| # | 事实 | 证据 |
|---|------|------|
| F-1 | 输入配置页为 antd Table，当前 7 列（ID/配置名称/配置类型/描述/状态/关联任务/创建时间/操作），无任何预览能力 | `frontend/src/pages/system/input-configs/index.tsx:210-308`（columns useMemo） |
| F-2 | 输入项数据结构 `InputConfig`：`config_type: 'usb' \| 'stream'`，可预览源三态——`StreamEnabled && StreamURL`（protocol: rtmp/rtsp/srt/hls）、`USBCameraDevice`（backend: dshow/v4l2/avfoundation）、`HuaweiEnabled && Server` | `frontend/src/types/input-config.ts:16-63`、`internal/models/input_config.go:27-72` |
| F-3 | 权威的"配置 → 媒体源"映射已有先例，预览必须镜像同一优先级：stream 优先 → USB → （华为终端独立 RTSP 默认格式 `rtsp://{server}:554/stream`） | `internal/recorder/coordinator.go:522-590`（buildRecordingInput）、`internal/huawei/client.go:1074-1078`（GetRTSPStreamURL） |
| F-4 | ffmpeg 输入 argv 先例：rtsp → `-rtsp_transport tcp -i URL`；rtmp/srt/hls → `-i URL`；dshow → `-f dshow ... -i video=<name>`；v4l2 → `-f v4l2 -i <dev>` | `internal/recorder/coordinator.go:949-1001`（buildRTSPArgs/buildStreamArgs）、`:841-894`（buildUSBVideoArgs） |
| F-5 | 无任何"从直播源抓帧"能力：FrameCaptureService 只接受本地视频文件（扩展名白名单 + ffprobe 时长校验），RTSP/设备源不适用，需新写一个小的抓帧服务 | `internal/services/frame_capture_service.go:44-66`（扩展名白名单）、`:125-153`（CaptureFrameToBytes 临时文件模式，可镜像） |
| F-6 | 后端二进制图片响应先例：`c.DataFromReader`；本任务用更简单的 `c.Data(code, "image/jpeg", bytes)` | `internal/handlers/file_handler.go:146`、`internal/handlers/video_file_handler.go:554` |
| F-7 | 认证：`api := a.router.Group("/api/v1")` + `api.Use(middleware.MultiAuth(...))`，组内路由自动要求 SM4 Token / API Key；auditOp 是逐路由可选装饰器，且有"只读端点刻意跳过审计"先例 | `cmd/server/app.go:932-933`、`:1017-1027`（inputConfigs 组）、`:902-903`（validate-password 跳过审计注释） |
| F-8 | 前端二进制/带认证下载先例：`authedFetch`（复用 401 单飞/缓存重放状态机）+ `URL.createObjectURL`；`API_BASE_URL = import.meta.env.VITE_API_URL \|\| ''` | `frontend/src/api/apiClient.ts:458-469`、`frontend/src/api/video-file.ts:50-64` |
| F-9 | 技术栈：antd ^6.5.3 + React ^19.2.8 + vite 7 + vitest 4（单元测试无 npm script，直接 `npx vitest run`）；前端请求统一走 apiClient（quick 260828-j2a 已收口） | `frontend/package.json:24,29,58-59` |
| F-10 | 哨兵错误齐备，无需新增（新增哨兵会触发 docs/errors.md 重新生成 + CI sync 门禁，quick 任务应避免）：无源 → `ErrInvalidInput`、配置不存在 → `apperrors.NotFound(...)`（包装 ErrNotFound）、ffmpeg 失败 → `ErrFFmpegFailed` | `internal/errors/errors.go:14,32`、`internal/errors/mapping.go:228-233` |

**技术选型**：按需 ffmpeg 单帧抓取（JPEG 直接回响应体）+ 前端懒加载（点击才抓）+ 可选 10s 轮询。
不引入 MJPEG 长连接 / WebSocket / HLS 转推——项目无任何长连接先例，且"确认源是否正确"只需要低频单帧；抓帧进程单帧即退出，无持续流负载。

---

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: 后端按需抓帧能力（InputPreviewService + GetConfigPreview + 路由 + 测试）</name>
  <files>internal/services/input_preview_service.go, internal/services/input_preview_service_test.go, internal/handlers/input_config_handler.go, cmd/server/app.go</files>
  <behavior>
    - Test 1（源解析优先级）TestResolvePreviewSource：stream（StreamEnabled=true+StreamURL=rtsp://x）→ kind=stream；仅 USB（USBCameraDevice=video=Cam）→ kind=usb 且 backend/name 取自配置；stream+USB 同时配置 → stream 优先（镜像 buildRecordingInput :543-585 顺序）；仅华为（HuaweiEnabled=true+Server=10.0.0.5）→ kind=huawei 且 url=`rtsp://10.0.0.5:554/stream`；三者皆无 → error 且 errors.Is(err, apperrors.ErrInvalidInput)
    - Test 2（argv 构建）TestBuildPreviewArgs：rtsp → 参数含 ["-rtsp_transport","tcp","-i",url]；rtmp/srt/hls → ["-i",url]；未知协议 → ErrInvalidInput；dshow 且 USBCameraName 非空 → `-i video=<name>`（name 为空时回退 device 本身，镜像 buildUSBVideoArgs :870-880）；v4l2 → ["-f","v4l2","-i",dev]；所有 case 尾部含 ["-frames:v","1","-vf","scale=640:-2","-q:v","5","-y",<output>]
    - Test 3（配置不存在）TestCapturePreview_ConfigNotFound：db 无该 id → errors.Is(err, ErrNotFound)
    - Test 4（无可预览源）TestCapturePreview_NoSource：配置存在但三源皆空 → errors.Is(err, ErrInvalidInput)
    - Test 5（真实 ffmpeg 出图）TestCaptureffmpeg_ProducesJPEG：lavfi 源 `testsrc=duration=1:size=320x240:rate=1` 走真实 ./bin/ffmpeg，断言返回非空字节且以 JPEG magic（0xFF 0xD8）开头；`./bin/ffmpeg` 不存在时 t.Skip（不假失败）
  </behavior>
  <action>
    1. 新建 internal/services/input_preview_service.go（import 别名参照 input_config_service.go:13 的 `apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"`）：
       - struct `InputPreviewService{db *gorm.DB; logger *zap.Logger; ffmpegPath string; sema chan struct{}}`；`NewInputPreviewService(db *gorm.DB, logger *zap.Logger, ffmpegPath string) *InputPreviewService`，sema 容量 2（全局并发上限，防多行同时抓帧堆积 ffmpeg 进程；与 config.FFmpeg.MaxProcesses=5 的语义独立，取更保守值）。
       - `type previewSource struct { kind string /* stream|usb|huawei|lavfi */; protocol, url, device, backend, name string }`（lavfi kind 仅供测试注入）。
       - `resolveSource(cfg *models.InputConfig) (previewSource, error)`：按 F-3 优先级解析；华为 url 用 `fmt.Sprintf("rtsp://%s:554/stream", cfg.Server)`（对齐 huawei/client.go:1077 默认格式，不发起华为 API 会话——预览场景不必登录终端管理面）。无可预览源时返回 `fmt.Errorf("输入配置 %d 未配置可预览的源 (stream/usb/huawei): %w", cfg.ID, apperrors.ErrInvalidInput)`。
       - `buildArgs(src previewSource, outputPath string) ([]string, error)`：输入段按 F-4 镜像 coordinator 的协议/后端分支（stream: rtmp→`-i`、rtsp→`-rtsp_transport tcp -i`、srt/hls→`-i`；huawei/lavfi 分别按各自输入段；usb: dshow/v4l2/avfoundation，dshow 额外 `-video_size 1280x720 -framerate 15` 降低抓帧开销）。公共尾部固定 `-frames:v 1 -vf scale=640:-2 -q:v 5 -y <outputPath>`（缩到 640 宽控制响应体大小）。StreamURL 照 coordinator 先例原样作为 `-i` 参数（录制路径同样不注入 stream_username/password），设备名/URL 只作为 argv 元素，不经 shell。
       - `CapturePreview(ctx context.Context, configID uint) ([]byte, error)`：`s.db.WithContext(ctx).First(&models.InputConfig{}, configID)`，gorm.ErrRecordNotFound 用 `apperrors.NotFound("input config", configID)` 包装（mapping.go:228 helper）；resolveSource → 信号量获取（`select { case s.sema <- struct{}{}: ... case <-ctx.Done(): ... }`，release 用 defer）→ `capture(ctx, src)`。
       - `capture(ctx, src)`：`context.WithTimeout(ctx, 10*time.Second)`（死源 RTSP 连接挂起由 CommandContext kill 兜底）；临时输出文件 `os.CreateTemp("", "input_preview_*.jpg")`（镜像 frame_capture_service.go:125-153 的临时文件→读字节→删除模式）；`exec.CommandContext(ctx, s.ffmpegPath, args...)` + stderr strings.Builder；非零退出 → `fmt.Errorf("预览抓帧失败: %w: %w, stderr: %s", apperrors.ErrFFmpegFailed, err, stderr.String())`（项目 `%w: %w` 双包装风格，参照 input_config_service.go:368）；成功后读文件（空文件按失败处理）并删除临时文件。
    2. internal/handlers/input_config_handler.go：struct 增字段 `previewService *services.InputPreviewService`，NewInputConfigHandler 增第 5 参；新增 `GetConfigPreview`（@Router /api/v1/input-configs/{id}/preview [get]）：id 解析照 GetConfig :82-88 的 ParseUint 写法；成功路径 `c.Header("Cache-Control", "no-store")` + `c.Data(http.StatusOK, "image/jpeg", jpeg)`（不走 GinSuccess JSON 信封——前端 `<img>` 直接消费字节）；失败路径 `h.logger.Error(..., response.SentinelField(err))` + `response.HandleError(c, err)`。
    3. cmd/server/app.go：:746 附近构造 `previewService := services.NewInputPreviewService(a.db, a.logger, a.config.FFmpeg.Path)`（路径默认 ./bin/ffmpeg，config.go:608）；:869 NewInputConfigHandler 调用点补第 5 参；inputConfigs 组（:1019-1027）新增 `inputConfigs.GET("/:id/preview", a.handlers.InputConfig.GetConfigPreview)`，并加注释说明刻意跳过 auditOp：预览是高频轮询的只读媒体端点，挂审计会污染日志（先例 :902-903 validate-password）；认证由 api 组 MultiAuth（:933）强制。
    4. 新建 internal/services/input_preview_service_test.go：db 用 sqlite 内存库 + AutoMigrate(&models.InputConfig{})（参照包内既有 *_test.go 基建）；Test 5 用 lavfi 源调 `capture`，`./bin/ffmpeg` 不存在则 t.Skip。项目约束：所有 go 命令必须 `CGO_ENABLED=0`。
  </action>
  <verify>
    <automated>cd "D:/code/ClaudeCode/record_V2" && CGO_ENABLED=0 go test ./internal/services/ -run "TestResolvePreviewSource|TestBuildPreviewArgs|TestCapturePreview|TestCaptureffmpeg" -count=1 -v && CGO_ENABLED=0 go vet ./... && CGO_ENABLED=0 go build ./...</automated>
  </verify>
  <done>5 组测试全绿（lavfi 用例真实出 JPEG）；GET /api/v1/input-configs/:id/preview 已注册且走 api 组认证；无新增 go.mod 依赖；CGO_ENABLED=0 下 test/vet/build 全过</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: 前端预览列（getInputConfigPreview + InputPreviewCell 组件 + 表格列）</name>
  <files>frontend/src/api/input-config.ts, frontend/src/api/__tests__/input-config-preview.test.ts, frontend/src/pages/system/input-configs/components/InputPreviewCell.tsx, frontend/src/pages/system/input-configs/index.tsx</files>
  <behavior>
    - Test 1（API 成功路径）：mock authedFetch 返回 ok+Blob → getInputConfigPreview(7) 请求 `http://localhost/api/v1/input-configs/7/preview`（API_BASE_URL 拼接），resolve 为该 Blob
    - Test 2（API 错误路径）：mock 返回 500 + JSON body `{message: "预览抓帧失败"}` → reject 且 error.message 含该文案
    - Test 3（多 id 路由）：id=1 与 id=2 → 两次请求 URL 分别以 /1/preview 与 /2/preview 结尾
  </behavior>
  <action>
    1. frontend/src/api/input-config.ts 顶部补 `import { apiRequest, authedFetch } from './apiClient'` 与 `const API_BASE_URL = import.meta.env.VITE_API_URL || ''`（照 video-file.ts:12 先例）；新增导出 `getInputConfigPreview(id: number): Promise<Blob>`：`const res = await authedFetch(`${'$'}{API_BASE_URL}/api/v1/input-configs/${'$'}{id}/preview`)`；`!res.ok` 时解析 JSON body 取 `message` 抛 `new Error(...)`（照 downloadVideoFile :50-64 的二进制响应处理风格，错误 body 是 JSON 信封需单独读）；成功 `return res.blob()`。调用方负责 revoke objectURL，注释说明。
    2. 新建 frontend/src/pages/system/input-configs/components/InputPreviewCell.tsx（antd v6 + React 19 函数组件，行内 style 风格与所在页面一致，不引入新依赖）：
       - props `{ config: InputConfig }`；状态 `status: 'idle' | 'loading' | 'ok' | 'error'` + `url?: string` + `errorMsg?: string` + `autoRefresh: boolean`（默认 false）+ `inFlightRef` 防重入。
       - `fetchFrame()`：inFlight 时直接 return（轮询重叠保护）→ setInput loading → `getInputConfigPreview(config.id)` → `URL.createObjectURL(blob)`（创建前 revoke 旧 url）→ ok；catch → error + 记录 message。
       - idle 视图：虚线边框占位框（高约 54px）+ `<Button size="small" type="link" icon={<PlayCircleOutlined />}>预览</Button>`（懒加载入口，页面打开不拉流）。
       - ok 视图：`<img src={url} alt={`${'$'}{config.name} 预览`} style={{ height: 54, borderRadius: 4, display: 'block' }} />` + 刷新按钮（`<ReloadOutlined />`，点击 fetchFrame）+ `<Switch size="small" checked={autoRefresh} onChange={...} />` 表示自动刷新。
       - error 视图：占位框内红色简短文案（如"无法获取画面"）+ `<Tooltip title={errorMsg}>` 展示完整错误 + 重试按钮。
       - 自动刷新：`useEffect` 中当 `autoRefresh && status === 'ok'` 时 `setInterval(fetchFrame, 10_000)`，依赖变化或卸载时 clearInterval；`useEffect` 卸载清理 revokeObjectURL（避免内存泄漏）。
    3. frontend/src/pages/system/input-configs/index.tsx：import InputPreviewCell；columns useMemo（:210-308）在"配置类型"列之后插入 `{ title: '预览', key: 'preview', width: 150, render: (_, record) => <InputPreviewCell config={record} /> }`；`scroll={{ x: 1200 }}` 调整为 `x: 1360`。
    4. 新建 frontend/src/api/__tests__/input-config-preview.test.ts：`vi.mock('../apiClient', ...)` 桩掉 authedFetch（照 apiClient.test.ts 的 vitest 4 写法），覆盖上述 3 个 behavior；不 mock antd（纯 api 模块无 UI 副作用）。
  </action>
  <verify>
    <automated>cd frontend && npx vitest run src/api/__tests__/input-config-preview.test.ts && npx tsc -b --noEmit && npx eslint src/pages/system/input-configs/components/InputPreviewCell.tsx src/api/input-config.ts src/api/__tests__/input-config-preview.test.ts --max-warnings 0</automated>
  </verify>
  <done>预览列出现在表格中；组件默认不发起任何请求（点击才抓）；自动刷新默认关、有重叠保护与卸载清理；vitest + tsc + eslint 全绿；零新增 npm 依赖</done>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <name>Task 3: 真实源人工验收（画面帧 / 错误提示 / 轮询行为）</name>
  <files>无新增文件（人工验收任务，不修改代码）</files>
  <action>向用户呈现 what-built 与 how-to-verify 步骤并暂停等待结果；用户描述的任何问题按 quick-fix 流程回炉修复后重新验收，不得自行标记通过。</action>
  <what-built>输入配置管理页新增「预览」列：点击后后端按需用 ffmpeg 从该行配置的源（USB 摄像头 / RTSP/RTMP/SRT/HLS 流 / 华为终端默认 RTSP）抓取单帧 JPEG 返回并显示，支持手动刷新与 10s 自动刷新。Task 1/2 的自动化门禁（go test/vet/build + vitest/tsc/eslint）已全部通过。</what-built>
  <how-to-verify>
    1. 重启后端与前端：`CGO_ENABLED=0 go build -o server.exe ./cmd/server` 后启动 server.exe；另开终端 `cd frontend && npm run dev`。
    2. 用 AD 域账号登录（本机 auth.mode=AD，admin/admin123 不可用）。
    3. 打开 系统管理 → 输入配置管理，确认表格出现「预览」列，且页面刚打开时 Network 面板没有任何 /preview 请求（懒加载验证）。
    4. 对一行配置了真实源的配置点击「预览」→ 约 5-10s 内该行显示画面帧；点击刷新按钮画面更新。
    5. 开启该行「自动」开关 → Network 面板每 10s 出现一次 /preview 请求且画面随之更新；切走页面或刷新后轮询停止。
    6. 对一行源不可达（如错误 RTSP 地址）或未配置源的配置点击「预览」→ 约 10s 内显示错误提示，页面不卡死，其他行仍可正常预览（并发上限 2 的排队行为）。
  </how-to-verify>
  <verify>Task 1 与 Task 2 的 verify 命令已全部通过后才能进入本 checkpoint；本任务本身为人工验收，无自动化命令</verify>
  <done>用户回复 approved，或描述的问题已回炉修复后重新验收通过</done>
  <resume-signal>Type "approved" or describe issues</resume-signal>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| browser → API | 前端携带 Bearer token 访问 /api/v1/input-configs/:id/preview，响应为内网摄像头/会议流的画面帧 |
| API → 内网源 | 后端 ffmpeg 以 argv 形式连接用户配置的 stream_url / 华为终端 / 本机 USB 设备 |
| 用户输入 → 进程 | stream_url、设备名、协议字段进入 ffmpeg argv（不经 shell） |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-KRH-01 | Spoofing / Information Disclosure | GET /input-configs/:id/preview | mitigate | 路由注册在 api 组内，`middleware.MultiAuth`（app.go:933）强制 SM4 Token / API Key 认证；不注册到公开路由；前端走 authedFetch 携带 Bearer。画面含内网会议内容，绝不设免认证路径 |
| T-KRH-02 | DoS | InputPreviewService 抓帧 | mitigate | 全局信号量上限 2 + 单次 `context.WithTimeout(10s)` + `exec.CommandContext` kill（死源 RTSP 连接挂起不会堆积进程）；`-frames:v 1` 保证进程单帧即退出；前端懒加载 + inFlight 防重入，页面打开零请求 |
| T-KRH-03 | Tampering | stream_url / 设备名 / id 参数 | mitigate | id 经 `strconv.ParseUint` 后走 GORM 参数化 `First(&cfg, id)`；URL/设备名仅作为 exec argv 元素（无 shell，无拼接命令）；协议白名单分支（rtmp/rtsp/srt/hls），未知协议拒绝（ErrInvalidInput）；与既有录制路径 coordinator.go 的输入处理同源 |
| T-KRH-04 | Repudiation / 日志污染 | auditOp 缺省 | accept(有意降级) | 预览为高频只读媒体端点，逐次挂 auditOp 会灌爆审计日志——沿用 app.go:902-903 validate-password「只读端点刻意跳过审计」先例；对配置本身的增删改查仍全量审计（既有路由不变） |
| T-KRH-05 | Information Disclosure | 错误日志 | mitigate | ffmpeg stderr 与错误日志不包含凭据（stream_password 不进 URL，与录制路径一致，buildStreamArgs 只用 `-i URL`）；响应体仅画面字节 |
| T-KRH-SC | Tampering | npm / go 依赖安装 | mitigate | 零新增依赖：前端无新 npm 包，后端零 go.mod 变更，无供应链引入面 |
</threat_model>

<verification>
- `cd "D:/code/ClaudeCode/record_V2" && CGO_ENABLED=0 go test ./internal/services/ -run "TestResolvePreviewSource|TestBuildPreviewArgs|TestCapturePreview|TestCaptureffmpeg" -count=1` 通过（含 lavfi 真实出 JPEG 用例）
- `CGO_ENABLED=0 go vet ./... && CGO_ENABLED=0 go build ./...` 通过
- `cd frontend && npx vitest run && npx tsc -b --noEmit && npx eslint src/pages/system/input-configs src/api/input-config.ts --max-warnings 0` 通过
- grep 证实无新增裸 fetch：`grep -rn "authedFetch\|apiRequest" frontend/src/api/input-config.ts` 命中 preview 实现；后端无新增哨兵错误（`git diff --stat` 不含 internal/errors/ 与 docs/errors.md）
- 人工验证（Task 3）：真实源出画面、不可达源出错误提示、无并发拉流与轮询残留
</verification>

<success_criteria>
1. 输入配置列表每一行可按需查看该输入源的画面帧（USB / RTSP/RTMP/SRT/HLS / 华为终端三类源均覆盖）
2. 预览是用户主动触发的单帧抓取：页面打开零请求、并发上限 2、单次 10s 超时——多项列表不产生持续流负载
3. 源不可达 / 未配置源 / 配置不存在分别给出可辨识的错误提示，页面不卡死
4. 预览链路完整走既有认证：前端 authedFetch（401 状态机）→ 后端 api 组 MultiAuth
5. CGO_ENABLED=0 下 go test/vet/build 与前端 vitest/tsc/eslint 全绿，零新增依赖，零新增哨兵错误
</success_criteria>

<output>
完成后在 `.planning/quick/260828-krh-input-config-column-video-preview/` 创建 SUMMARY，并更新 `.planning/STATE.md` 的 Quick Tasks Completed 表。
</output>
