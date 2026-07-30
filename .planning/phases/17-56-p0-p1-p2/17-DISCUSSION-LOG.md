# Phase 17: 后端代码审查 56 个发现修复 - P0/P1/P2 全量 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-30
**Phase:** 17-56-p0-p1-p2
**Areas discussed:** Scope chunking, Test discipline, Commit strategy, Breaking changes

---

## Scope chunking

| Option | Description | Selected |
|--------|-------------|----------|
| 一次完成全量 56 个 | 按 P0→P1→P2 顺序在一个 phase 内全部修复（与 Phase 16 D-01.2 一致） | ✓ |
| 拆成 P0 only / P1+P2 两个 phase | 本 phase 只做 P0（13 个 HIGH），开新 phase 做 P1+P2 | |
| 拆成 P0+P1 / P2 两个 phase | 本 phase 做 31 个 HIGH+MEDIUM，开新 phase 做 25 个 LOW 清理 | |

**User's choice:** 一次完成全量 56 个（与 Phase 16 D-01.2 一致）
**Notes:** 用户明确希望"一次完成全量改、不分阶段"，与 Phase 16 的视觉重塑决策保持节奏一致。

---

## Test discipline

| Option | Description | Selected |
|--------|-------------|----------|
| 每个修复都加单元测试 | P0 修复必须有单测（13 处必做），P1 至少加，P2 跳过 | ✓ |
| 只修不改测试（纯代码修复） | 所有 56 个修复都不加新测试 | |
| 只对 P0 HIGH + 真 bug 加测试（11 个） | 只对真 bug + 高风险安全修复加单测 | |

**User's choice:** 每个修复都加单元测试
**Notes:** P0 (13) 全部必做；P1 (18) 至少每个加一个；P2 (25) 跳过。审计已指出测试覆盖稀疏，per-fix 测试是增量补覆盖率。

---

## Commit strategy

| Option | Description | Selected |
|--------|-------------|----------|
| 按 P0/P1/P2 分组 3 个 mega commit | 每个 tier 一个 commit，commit message 列出所有引用 finding | ✓ |
| 每个发现一个原子提交（56 commits） | SEC-002 一个 commit、BUG-001 一个 commit 等 | |
| 按模块/文件分组（10-15 commits） | config.go 一个 commit、auth/ 一个 commit 等 | |

**User's choice:** 按 P0/P1/P2 分组 3 个 mega commit
**Notes:** 保持 PR 紧凑，main 历史干净。每个 mega commit 的 commit message body 必须列出所有引用的 finding ID。

---

## Breaking changes

| Option | Description | Selected |
|--------|-------------|----------|
| 内置迁移 + 启动校验 + 部署文档更新 | 启动时 secret 为空 logger.Fatal；更新 DEPLOYMENT.md / BUILD.md / .env.example | ✓ |
| 仅代码修复 + CI 阻断，不更新部署文档 | 只改 internal/ + cmd/，让运维从 panic 自行查日志 | |
| 保留兼容开关（如 SM4_SECRET_FALLBACK_ENABLED） | 保留旧 env var 名 + 默认值但打印 deprecation warning | |

**User's choice:** 内置迁移 + 启动校验 + 部署文档更新
**Notes:** 同步更新 DEPLOYMENT.md / BUILD.md / SECURITY.md / .env.example。HMAC 编码变更保留向后解析兼容（新签发用 RawURLEncoding，Verify 接受两种编码）。不保留 deprecated 兼容开关。

---

## Claude's Discretion

- BUG-006（time.Sleep → time.NewTimer + select）的具体重构写法
- PERF-007/008/009（sync.Pool、正则包级、类型化 struct）的局部实现选择
- STYLE-009 包名冗余清理（133 处 Get*）— 默认跳过以减少 PR 噪音
- STYLE-010 godoc 缺失 8 处的注释措辞（中文/英文）
- 中间件类型断言 `, ok` 守卫的具体错误返回值

## Deferred Ideas

- STYLE-009 包名冗余清理（133 处 Get*）— 影响面过大且与审计核心修复无关
- 引入 `koanf` 替代 viper（审计 6.3 节建议）— 范围外的依赖迁移
- audit 包从 `internal/services/audit` 挪到 `internal/audit`（审计 6.3 节建议）— 影响 import 链
- 403 处 GORM 全库加 `WithContext`（PERF-003 全集）— 本次仅修改/新增处加
- 引入 `golangci-lint` + `errcheck`/`gosec` 规则（审计 6.3 节建议）— 工具链改造
- 测试覆盖稀疏问题全面提升 — 本次 per-fix 增量补；全面覆盖率提升独立 phase
- HMAC jti 服务端 `used_jtis` 表（Redis 或 DB）— 需要架构决策