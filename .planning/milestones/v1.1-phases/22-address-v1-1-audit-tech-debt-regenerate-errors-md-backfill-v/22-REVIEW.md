# Phase 22 — Code Review: `docs/errors.md`

- **status**: `clean`
- **depth**: standard
- **diff base**: `c6c5771^`
- **files in scope**: `docs/errors.md` (1 file, +4/−4)
- **reviewed at**: 2026-08-03

---

## 1. Scope 判定

Phase 22 是 process-artifact-only phase，未改动 production code。唯一 in-scope 文件
`docs/errors.md` 由 `go generate ./internal/errors/...`（generator: `cmd/error-doc-gen`）
自动生成，因此本次 review **不评判其手写风格**，只验证三件事：

1. 生成内容是否与 `internal/errors/errors.go` + `internal/errors/mapping.go` 一致；
2. audit footer 的 `16` 是否为真实计数；
3. `.github/workflows/ci.yml` 的 sync-check 是否仍然成立（regeneration 是否确定性）。

本次 diff 仅 4 行数值变化（`ErrInternal` 105→108、`INTERNAL_ERROR` 66→67、
`NOT_FOUND` 49→50、ad-hoc count 15→16），属于 stale doc 的重新生成，符合 phase 意图。

---

## 2. 验证结果

### 2.1 Sentinel 清单一致性 — PASS

程序化比对 `errors.go` 中的 sentinel 定义与文档 Sentinel Table：

```
src sentinels: 42  doc rows: 42
diff → 空  >>> SENTINELS MATCH
```

无遗漏、无多余、无拼写漂移。

### 2.2 HTTP status 映射一致性 — PASS

对照 `mapping.go` 的 `MapToHTTPStatus` switch，逐条核验 42 个 sentinel 的 status：

- 404 组（`ErrNotFound` / `ErrTaskNotFound` / `ErrVideoFileNotFound` / `ErrUserNotFound` /
  `ErrRoleNotFound` / `ErrADAccountNotFound` / `ErrPermissionNotFound` / `ErrAPIKeyNotFound` /
  `ErrPPTFileNotFound`）— 9 条全部一致；
- 401 组（7 条）、403 组（7 条）、400 组（2 条）、409 组（6 条）、503 组（4 条）、
  500 组（`ErrFFmpegFailed` / `ErrTranscriptionFailed` / `ErrSplitFailed`）— 全部一致；
- 429：`ErrInsufficientQuota` — 一致；
- **default 落点正确**：`ErrInternal`、`ErrDuplicateRecord`、`ErrForeignKeyConstraint`
  未出现在 switch 的任何 case 中，文档标注 500（default 分支）——这一点容易被误判为
  「漏映射」，实际与 `mapping.go:94-95` 的保守 default 行为完全吻合。

### 2.3 BusinessError code 清单与映射 — PASS

```
src codes: 10  doc rows: 10
diff → 空  >>> CODES MATCH
```

逐条对照 `mapBusinessError`：`NOT_FOUND`→404、`ALREADY_EXISTS`/`TASK_IN_PROGRESS`→409、
`INVALID_INPUT`→400、`UNAUTHORIZED`→401、`FORBIDDEN`→403、`SERVICE_UNAVAILABLE`→503、
`FFMPEG_ERROR`/`INTERNAL_ERROR`→500 均一致。`FOREIGN_KEY_CONSTRAINT` 走 default→500，
文档标注 500，正确。

### 2.4 Audit footer `16` — PASS（独立复算）

未采信 generator 自报值，而是按 `auditAdHocErrors` + `grepCountInSource` 的语义
（两条 pattern `\berr\.Error\(\)` 与 `\berrMsg\s*:=`；排除 `_test.go`；排除
`ShouldBindJSON` block 内的行）在 `internal/handlers/` 上独立复算：

```
replicated adHocCount = 16
```

与文档中的 **16** 精确吻合。（原始匹配 32 行，`ShouldBindJSON` block 排除掉 16 行。）

同时定位了 15→16 的增量来源：`internal/handlers/admin_handler.go:515`
（`updates["error_msg"] = err.Error()`），由上次 regen（`7a72675`）之后的 handler 改动引入。
`cmd/error-doc-gen` 自 `2d7ea72` 起未变更，故此次数值漂移是**真实的代码漂移**，
而非 generator 逻辑变化——文档更新是必要且正确的。

### 2.5 CI sync-check / 确定性 — PASS

```
$ go generate ./internal/errors/...   # exit 0
$ git diff docs/errors.md             # 空
$ git status --porcelain docs/errors.md  # 空
```

重新生成产出与已提交内容 **byte-identical**，`.github/workflows/ci.yml:45-51`
的 `Verify errors doc sync` 步骤将通过（SYNC_OK）。generator 与 errors 包测试同样通过：

```
ok  internal/errors        (cached)
ok  cmd/error-doc-gen      20.946s
```

---

## 3. Findings

**无。** 在 in-scope 文件 `docs/errors.md` 中未发现任何缺陷——内容与 source of truth
完全一致，计数真实，生成可复现。

---

## 4. Observations（非阻塞；根因在 out-of-scope 的 generator source）

以下不是 `docs/errors.md` 的缺陷（文档忠实反映了 generator 输出），但会影响 audit
指标的可解释性，建议在后续触及 `cmd/error-doc-gen` 的 phase 中一并处理。

### OBS-1：ad-hoc 指标存在 false positive，弱化了「target: 0」门禁语义

文档 footer 声明「If this number grows, a handler has regressed to the pre-Phase-20
anti-pattern」。但本次 15→16 的那 **+1 并非 anti-pattern 回归**：

`admin_handler.go:513-518` 是后台迁移 job 把错误文本落库到 `AdminMigrationJob.error_msg`
列，与 HTTP 状态分类无关。

对 16 条命中做人工分类：

| 类别 | 条数 | 示例 |
|------|------|------|
| 真实 ad-hoc classify（硬编码 code + 裸 `err.Error()`） | 11 | `ppt_handler.go` 的 `CodeForbidden` ×9；`auth_handler.go:90/179` |
| 结构化日志 / 落库（非分类） | 2 | `admin_handler.go:515`、`video_recording_task_handler.go:772` (`zap.String`) |
| 入参/解码校验（generator 注释自称应排除） | 3 | `ppt_handler.go:302` (`json.Unmarshal`)、`891`（base64 decode）、`250` |

建议：pattern 增加对 `zap.`/`Updates(`/赋值语境的负向排除，使该指标真正等价于
「remaining classify branches」。

### OBS-2：`ShouldBindJSON` 排除规则覆盖不全

`grepCountInSource` 只识别字面量 `ShouldBindJSON`，但 `ppt_handler.go:297` 用的是
`json.Unmarshal(bodyBytes, &req)` 做等价的请求体绑定校验，未被排除。按 generator 自身注释
（「request-binding validation errors are expected in handlers」），这类应当同样豁免。

### OBS-3（安全，最低优先级）：`ppt_handler.go:248-251` 将原始错误文本写入响应体

批量查询的 partial-failure 分支把 `err.Error()` 放进返回 payload。虽不影响状态码，
但存在内部错误细节外泄的可能。属 production code，不在本 phase 范围内，仅作记录。

**收敛进度参考**：11 条真实 ad-hoc 中有 9 条是 `verifyPPTOwnership` → `CodeForbidden`
的同一模式，可用单次重构（返回 `ErrForbidden` sentinel + `response.HandleError`）一并消除。

---

## 5. 结论

`docs/errors.md` 与 `internal/errors/` 完全同步，42 个 sentinel、10 个 BusinessError code
及其 HTTP status 映射逐条核验无误，audit 计数 16 经独立复算属实，重新生成确定性可复现、
CI sync-check 成立。

**verdict: `clean` — 无需返工，可直接进入 Phase 22 收尾。**
