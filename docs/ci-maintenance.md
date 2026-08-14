# CI/CD 维护注意事项

> 适用对象：所有改 `.github/workflows/ci.yml`、`.golangci.yml`、`frontend/package.json`、`go.mod` 的人。

最后更新：2026-08-04（CI 全面修绿后沉淀）

---

## 1. 4 个根因速查（不要重复踩）

历史上让 `ci-main` 长期失败的因素，**新代码/PR 改动时请自查是否触及**：

| 触发条件 | 失败 step | 错误信息关键词 | 历史 commit |
|----------|-----------|----------------|-------------|
| Backend 在 CI fresh runner 跑 `go vet` | `Vet` | `pattern dist: no matching files found`（embed.FS） | `787a39a` |
| Frontend `npm ci` + antd v6 + pro-layout | `Install dependencies` | `ERESOLVE ... peer antd@"^4.24.15 \|\| ^5.11.2"` | `e35319f` |
| `internal/errors/` 改了但 docs/errors.md 没重生 | `Verify errors doc sync` | `docs/errors.md is out of sync` | `9a97c30` |
| golangci-lint v2 + action v6 | `golangci-lint` | `invalid version string 'v2.12.2', you must update to golangci-lint-action v7` | `e6689d6` |

## 2. CI 工具链版本约束

| 组件 | 必须版本 | 原因 |
|------|----------|------|
| `go`（go.mod） | `1.25.x` | 项目目标版本 |
| `golangci-lint` | `>= v2.12.2`（需 go1.25 构建） | 旧版（v1.x / v2.7.x）会报 Go 语言版本过低 |
| `golangci/golangci-lint-action` | `>= v7`（v2 需 v7+） | v6 不支持 v2 lint |
| `node` | `24` | `actions/setup-node@v4` 默认 |
| `.golangci.yml` schema | **v2** | v1 schema 与 v2 lint 不兼容 |
| `frontend/antd` | `^6` | 不要回退 v5（pro-layout 是死路） |

> ⚠️ 改其中任何一项前，**先读历史 commit `e6689d6`** 确认同步配套。

## 3. jobs 间依赖关系

```
frontend  ──┐
            ├──> backend ──> cross-build ──┐
            │                              ├──> ci-status
            └──────────────────────────────┘
```

- **backend** has `needs: frontend`：要 frontend-dist artifact（embed.FS）。
- **cross-build** 有自己的 npm ci + frontend build（不依赖 frontend job）—— 故意冗余，避免单点失败。
- **ci-status** 仅做 gate check：所有上游 job success 它才 success。

修改依赖图前请确认新 job 真的需要上游 artifact。

## 4. 调试 CI 失败的正确顺序

不要看 GitHub run 顶部的 "Annotations 1 error and 1 warning" 盲猜，按这个顺序：

1. **拉 jobs/steps 结论**：
   ```bash
   curl -s "https://api.github.com/repos/<owner>/<repo>/actions/runs/<run_id>/jobs" \
     | python -c "import sys,json; [print(f'{s[\"conclusion\"]:>8}  {s[\"name\"]}') for s in json.load(sys.stdin)['jobs']]"
   ```
2. **拉具体 check-run 的 annotation（公开仓库免 auth）**：
   ```bash
   curl -s "https://api.github.com/repos/<owner>/<repo>/check-runs/<job_id>/annotations"
   ```
3. **本地复现**：
   - 后端：`golangci-lint run --timeout=5m ./...`、`go vet ./...`、`go test -race ./...`
   - 前端：`cd frontend && npm ci && npm run lint && npm run build`
   - **别相信"本地绿 = CI 绿"**——action wrapper 版本、缓存、fresh runner 状态都可能差

## 5. 改 internal/errors/ 后必做

```bash
go generate ./internal/errors/...
git diff docs/errors.md
# 如果有 diff，必须一并提交，否则 CI step#8 "Verify errors doc sync" 会挂
```

## 6. 不要做的蠢事

- ❌ `version: latest` 任何 CI 工具
- ❌ `npm ci` 失败改 `npm install`（绕过 lockfile 检查）
- ❌ 单独升级 antd v6 → 降 v5（pro-layout 是死路，回退 = 项目技术债倒退）
- ❌ 用 `git commit --no-verify` 绕过 husky 等本地检查而不修问题
- ❌ 把 ci-status 加到 branch protection required check（它是聚合 job，单独选会误判）

---

**附**：完整 4 commit 修复链 + 教训见 `.claude/memory/project-ci-fully-green-2026-08.md`。