# Milestones

## v1.1 文件管理与编辑增强 / 后端安全加固 (Shipped: 2026-08-03)

**Phases completed:** 6 phases (17-22), 25 plans

**Timeline:** 2026-07-30 → 2026-08-03 (5 days)
**Git range:** `cf2d248` (phase 17 plan) → HEAD — 247 commits, 267 files, +35261/-5976 LOC

**Key accomplishments:**

1. **后端代码审查 56 项全量修复**（Phase 17）— 13 HIGH + 18 MEDIUM + 25 LOW 发现全量修复，45 个原子 commit，12 包 `go test -race` 全绿，零回归
2. **凭据静态加密 SM4-GCM + 自动密钥轮换**（Phase 18，SEC-003b）— 华为密码明文→AEAD envelope（`SM4:<version>:<base64(nonce|ciphertext|tag)>`），与浏览器传输密钥族隔离，fail-closed 启动 10 步不变量，操作员轮换手册 + 物理残留边界文档
3. **ctx 全量级联 + jti replay 防御 + error-mapping 三组件**（Phase 19）— 403 处 GORM `.WithContext(ctx)` + ~190 service 方法 ctx 透传；SEC-004 jti replay 修复（TTL sweeper，不加 DB 表）；mapping.go + HandleError + error_mapper.go
4. **HandleError 统一收敛 + sentinel 体系**（Phase 20）— 9 个 handler 的 ad-hoc classify 全量清理（含 `classifyAuthLoginError` 函数删除），zap logger `SentinelField` 4-state（sentinel/BusinessError/ad-hoc），自动生成 `docs/errors.md` + CI sync-check 门禁
5. **v1.1 过程缺口闭环**（Phase 21）— goal-backward retro-verify 重建 phase 17/18/19 的 VERIFICATION.md（5/5 passed）；创建 `REQUIREMENTS.md`（251 行，~80 REQ-ID，0 orphan）；auth_handler.go:57 规范化为 canonical HandleError 模式
6. **审计 tech debt 收尾**（Phase 22）— regenerate `docs/errors.md`（footer=16，CI SYNC_OK）；回填 phase 17/18/19/21 的 Nyquist VALIDATION.md；翻转 phase 20 VALIDATION.md 签核（nyquist_compliant/wave_0_complete→true，6/6 sign-off）

**Audit status:** 重审 `v1.1-MILESTONE-AUDIT-REAUDIT.md` `status: tech_debt`，gaps 全空（requirements 60/60、phases 5/5、integration 5/5、flows 4/4），5/5 phase VERIFICATION passed。原始审计 `v1.1-MILESTONE-AUDIT.md`（gaps_found, immutable）一并归档保留演进历史。

**Known deferred items at close:** 24（全部为 v1.0 历史遗留 planning 元数据：4 debug sessions / 14 quick tasks / 1 seed / 4 uat gaps / 1 verification gap — 非 v1.1 交付物，见 STATE.md Deferred Items）

**Key tech debt deferred:** STYLE-001 全库 %w 迁移（~117 errors.New + ~474 fmt.Errorf）；STYLE-009 Get* rename（124 处）；KMS/Vault 自动注入凭据；真实生产数据 post-audit；jti replay 多实例需 Redis（单实例 5min 窗口风险已接受）。

---
*See .planning/milestones/v1.1-ROADMAP.md for full milestone details.*

---

## v1.0 视频切割与会议转录PPT (Shipped: 2026-05-06)

**Phases completed:** 14 phases, 69 plans, 2 deferred items

**Key accomplishments:**

1. **视频分割功能** - 用户可以在浏览器中点击时间线添加分割标记，FFmpeg 分割视频为多个 MP4 片段
2. **本地/云端转录** - 集成阿里通义听悟，支持本地和云端两种转录模式，自动生成 PPT
3. **PPT 管理和编辑** - PPT 预览、多结果管理、幻灯片合并、重复检测、删除回滚、视频帧捕获插入
4. **AD 域控认证** - 集成 Windows Active Directory 认证，支持白名单模式
5. **USB/流媒体录制** - 重构配置架构，支持 USB 直录和流媒体 (RTMP/RTSP) 录制模式
6. **批量操作** - 文件管理页面支持批量下载（ZIP 打包）和批量转录（任务组模式）
7. **多角色权限** - 用户可以拥有多个角色，权限聚合使用 OR 逻辑
8. **审计日志** - 记录敏感操作供审计查询
9. **UI 增强** - 预览页面缩略图侧边栏、可编辑进度条、PPT 切换下拉、键盘快捷键

**Known deferred items at close:** 2 (UAT 测试延期 - Phase 01-08 UI 交互测试需要人工运行时验证)

---
*See .planning/milestones/v1.0-ROADMAP.md for full milestone details.*
