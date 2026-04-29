---
phase: 13
slug: usb
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-29
---

# Phase 13 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Standard Go testing + Testify |
| **Config file** | No specific config file — tests alongside source code |
| **Quick run command** | `go test ./internal/services -run TestInputConfig -v` |
| **Full suite command** | `go test ./... -v` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/services -run TestInputConfig -v`
- **After every plan wave:** Run `go test ./... -v`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 13-01-01 | 01 | 1 | D-01, D-02 | T-13-01 | Config type validation enforces mutual exclusion | unit | `go test ./internal/models -run TestInputConfig_Validate -v` | ❌ W0 | ⬜ pending |
| 13-01-02 | 01 | 2 | D-04, D-05 | T-13-02 | Required field validation prevents empty configs | integration | `go test ./internal/services -run TestInputConfigService_ValidateConfig -v` | ❌ W0 | ⬜ pending |
| 13-02-01 | 02 | 1 | D-09 | T-13-03 | Database migration preserves data integrity | unit | `go test ./internal/migrations -run Test014_CreateInputConfigs -v` | ❌ W0 | ⬜ pending |
| 13-03-01 | 03 | 1 | D-12 | T-13-04 | API routes use authentication middleware | integration | `go test ./internal/handlers -run TestInputConfigHandler_CreateConfig -v` | ❌ W0 | ⬜ pending |
| 13-04-01 | 04 | 1 | D-07 | T-13-05 | Frontend routes use API client with tokens | integration | `npm test -- --testPathPattern=InputConfigForm` | ❌ W0 | ⬜ pending |
| 13-05-01 | 05 | 1 | D-06 | T-13-06 | Scheduler validates config before recording | unit | `go test ./internal/scheduler -run TestVideoScheduler_InputConfig -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/models/input_config_test.go` — 配置模型验证测试
- [ ] `internal/services/input_config_service_test.go` — 配置服务业务逻辑测试
- [ ] `internal/handlers/input_config_handler_test.go` — API处理器测试
- [ ] `internal/scheduler/input_config_scheduler_test.go` — 调度器适配测试
- [ ] `frontend/src/components/__tests__/InputConfigForm.test.tsx` — 前端表单组件测试

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 华为终端实际连接测试 | D-05 | 需要物理硬件或真实华为终端 | 在开发/测试环境中配置真实华为终端,点击"测试连接"按钮,验证连接成功提示 |
| USB设备实际扫描测试 | D-05 | 需要物理USB设备 | 连接USB采集卡到测试机器,运行设备扫描,验证设备列表显示正确 |
| 流媒体URL连通性测试 | D-05 | 需要可访问的流媒体服务器 | 配置测试用RTMP/RTSP URL,点击"测试连接",验证连接状态和响应时间 |
| 前端路由重定向验证 | D-07 | 需要浏览器环境验证旧URL自动跳转 | 访问 `/system/huawei-configs`,验证自动重定向到 `/system/input-configs` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

---

*Phase: 13-usb*
*Validation strategy created: 2026-04-29*
