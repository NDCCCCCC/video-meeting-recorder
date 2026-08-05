---
slug: errors-doc-sync-ci-fail
status: fixed
trigger: chore(planning): 在 STATE.md 中记录里程碑重置 #27 CI — Verify errors doc sync 失败
created: 2026-08-05
updated: 2026-08-05
---

# Errors Doc Sync CI Failure - Debug Session

## Symptoms

**Expected Behavior:**
- `Verify errors doc sync` CI step passes on `chore(planning): 在 STATE.md 中记录里程碑重置` PR #27。
- `docs/errors.md` 的 `Sentinel Table` call-site count 与 `internal/errors` 当前源码计数一致。

**Actual Behavior:**
```
Run go generate ./internal/errors/...
Error: docs/errors.md is out of sync with internal/errors — run 'go generate ./internal/errors/...' and commit the result
diff --git a/docs/errors.md b/docs/errors.md
@@ -25,7 +25,7 @@
...
-| ErrInternal | Sentinel | 500 | 108 |
+| ErrInternal | Sentinel | 500 | 110 |
Error: Process completed with exit code 1.
```

## Current Focus

**hypothesis:** ✅ CONFIRMED — `docs/errors.md` 滞后于源码: 最近合入的 commit(主要是 `fix(hls): re-sign token` 链路)新增了 2 处 `ErrInternal` 用法但未重新生成 doc。

**next_action:** apply fix — `go generate ./internal/errors/...` 然后提交 `docs/errors.md`。
**test:** 重新触发 `Backend (Go 1.25)` → `Verify errors doc sync`, 预期通过。
**expecting:** 唯一变化是 `ErrInternal` 计数 108 → 110,无 sentinel 表新增/删除。
**reasoning_checkpoint:** null
**tdd_checkpoint:** null

## Evidence

- **2026-08-05 doc-sync 机制**:`internal/errors/errors.go:3`
  ```go
  //go:generate go run ../../cmd/error-doc-gen -errors-file errors.go -mapping-file mapping.go -output ../../docs/errors.md -repo-root ../..
  ```
  `cmd/error-doc-gen` 扫描所有 `.go` 文件,统计每个 sentinel 的 call-site 数量,生成 markdown 表。

- **CI 验证逻辑**:`.github/workflows/ci.yml` 中 `Verify errors doc sync` step 跑 `go generate`,然后 `git diff --exit-code docs/errors.md` 检查无差异即通过。

- **唯一差异行**:仅 `ErrInternal` 计数 108 → 110。其它 sentinel 名 + Kind + HTTP Status 全部稳定。无新 sentinel, 无 sentinel 删除。

- **call-site 增量来源(高概率)**:近期 `fix(hls): re-sign token on .m3u8 fetch to prevent .ts 401` (`e9a3788`) 链路新增 handler 错误返回路径, 加 `internal/huawei/client.go` 等若干文件新增 `ErrInternal` 用法,共 +2。

## Eliminated

- ~~CI 配置错误~~: 配置正确,只是 doc 没 regen。
- ~~新 sentinel 引入~~: doc 表只动了 ErrInternal 计数,38 个其它 sentinel 名稳定。
- ~~mapping.go 改动: doc 表 HTTP Status 列无任何变化。

## Resolution

**root_cause:** 常规代码改动(新增 `ErrInternal` call-sites)未随之 regen `docs/errors.md`,导致 doc 滞后于 src。CI 同步校验生效,触发失败。

**specialist_hint:** golang-pro / ci

**fix:** 执行 `go generate ./internal/errors/...` 将生成的 markdown 表写回 `docs/errors.md` 并 commit。

**affected_files:**
- `docs/errors.md` (1 行变化, ErrInternal count)

**fix:** ✅ APPLIED — commit `7b530dc fix(docs): regenerate errors.md — ErrInternal 108→110`

**fix_details:**
```diff
diff --git a/docs/errors.md b/docs/errors.md
index 85f3e9d..c678654 100644
--- a/docs/errors.md
+++ b/docs/errors.md
@@ -25,7 +25,7 @@
 | ErrForbidden | Sentinel | 403 | 24 |
 | ErrForeignKeyConstraint | Sentinel | 500 | 15 |
 | ErrInsufficientQuota | Sentinel | 429 | 12 |
-| ErrInternal | Sentinel | 500 | 108 |
+| ErrInternal | Sentinel | 500 | 110 |
 | ErrInvalidFileType | Sentinel | 400 | 5 |
 | ErrInvalidInput | Sentinel | 400 | 141 |
 | ErrNotFound | Sentinel | 404 | 31 |
```

**verification:**
1. `git diff docs/errors.md` 确认仅 ErrInternal 计数变化 108 → 110。
2. CI 重新跑 `Backend (Go 1.25)` → `Verify errors doc sync`, 预期通过。
3. `git show 7b530dc` 确认只改了 `docs/errors.md`。

## Why This Pattern Recurs

CI 设计本身就是为了"防止 doc 漂移"。但缺乏"代码改完自动 regen doc"的 guard。

**预防措施(可选 follow-up, 不在本次 fix 范围):**
- 加 pre-commit hook 自动跑 `go generate ./internal/errors/...`(同时 staged 改动自动覆盖 doc)。
- 或在编辑 `internal/errors` 触发的 PR 上, GitHub Action 提示 reviewer 是否 regen 过。

**为什么不需要立即引入**:本次 fix 是机械 regen,长期看每次新增 ErrInternal 用法都"忘了 regen"说明这是个高频且手工的步骤,值得 pre-commit hook 化。但单独属于 enhancement work。

## Notes

- 与历史 CI 全绿里程碑(见 memory: `project-ci-fully-green-2026-08.md`)中"重生 errors.md"是同类 fix,符合既定模式。
- 本次 fix 沿用 `[Commit boundary separation]` 原则: 单独一个 fix(docs) commit, 不与 Phase 工作或无关的 frontend dev session 文件混在一起(working tree 另有 3 个未跟踪的 frontend 文件未随本次提交)。
