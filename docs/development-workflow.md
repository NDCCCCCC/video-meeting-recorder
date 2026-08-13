# 开发流程

本项目通过 git hooks + 一键脚本实现本地与 CI 一致的 Go 代码质量自动化。
CI（`.github/workflows/ci.yml`）已完善；本文档描述**本地侧**的自动化门禁。

## 首次准备

```bash
./scripts/install-tools.sh      # 安装 golangci-lint + govulncheck（一次性）
./scripts/install-githooks.sh   # 激活 git hooks（设置 core.hooksPath=.githooks）
```

## 自动化门禁

### 提交时（pre-commit）

当暂存区含 `*.go` 文件时，自动执行：

1. `golangci-lint fmt` —— 按 `.golangci.yml`（gofmt + gofumpt + goimports）格式化，
   并把改动**重新加入暂存**（仅限原本已暂存的 `.go` 文件，不会偷偷加入未暂存的改动）。
2. `go vet ./...` —— 编译器级静态检查；失败则拒绝提交。

此外，改动涉及 `internal/errors` / `internal/handlers` / `pkg/response` 时，
自动 `go generate ./internal/errors/...` 同步 `docs/errors.md`。

> 未安装 golangci-lint 时，跳过格式化但仍跑 vet（建议先运行 `./scripts/install-tools.sh`）。

### 推送时（pre-push）

仅当推送内容含 `*.go` 改动时触发（纯文档/前端推送直接放行）：

- `go build ./...`
- `go test ./...`（本地不带 `-race` 提速；CI 带 `-race` 兜底）

任一失败则拒绝推送，避免坏代码推到 GitHub 才被 CI 暴露。

## 一键全量检查

`scripts/check.sh` 依次跑 8 步（等价 CI 的本地版），任一失败即停并汇总：

```bash
bash scripts/check.sh
```

| # | 步骤 | 命令 |
|---|------|------|
| 1 | go mod 校验 | `go mod tidy` 后检查 `go.mod`/`go.sum` 无漂移 |
| 2 | fmt 检查 | `golangci-lint fmt --diff` |
| 3 | vet | `go vet ./...` |
| 4 | lint | `golangci-lint run --timeout=5m` |
| 5 | build | `go build ./...` |
| 6 | test | `go test -race ./...` |
| 7 | 安全 | `govulncheck ./...` |
| 8 | errors.md 同步 | `go generate ./internal/errors/...` 后检查无漂移 |

## 跳过门禁

紧急情况可跳过单次（不要养成习惯）：

```bash
git commit --no-verify
git push --no-verify
```

## 范围与限制

- 上述 hook / `check.sh` **只覆盖根模块**，与 `.github/workflows/ci.yml` 一致。
- `scripts/check_db`、`scripts/fix_migration` 是**独立子 module**（各有自己的 `go.mod`），
  改动后需各自 `cd` 进去单独 `go vet` / `go test`。
- 规则配置见 `.golangci.yml`（基线 0 告警，新增代码须满足全部规则）。
