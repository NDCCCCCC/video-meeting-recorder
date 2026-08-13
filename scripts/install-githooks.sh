#!/usr/bin/env bash
# 安装项目 git hooks（pre-commit + pre-push）。
#
# 效果：
#   - git config core.hooksPath .githooks
#   - .githooks/pre-commit：docs/errors.md 同步 + golangci-lint fmt 自动格式化 + go vet 拦截
#   - .githooks/pre-push  ：仅当推送含 *.go 改动时跑 go build + go test（本地门禁）
#
# 用法：
#   ./scripts/install-githooks.sh
#
# 卸载：
#   git config --unset core.hooksPath

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

if [[ ! -d .githooks ]]; then
  echo "❌ .githooks 目录不存在（项目根目录）" >&2
  exit 1
fi

# 工具检测（缺失仅告警，不阻断安装）
echo "工具检测："
if command -v go >/dev/null 2>&1; then
  echo "  ✓ $(go version)"
else
  echo "  ⚠️ go 未安装（hook 需要它）" >&2
fi
if command -v golangci-lint >/dev/null 2>&1; then
  echo "  ✓ golangci-lint 已安装"
else
  echo "  ⚠️ golangci-lint 未安装 → pre-commit 将跳过自动格式化；运行 ./scripts/install-tools.sh 安装" >&2
fi
echo ""

# 确保所有 hook 可执行（从 git checkout 后权限可能丢失）
chmod +x .githooks/*

git config core.hooksPath .githooks

echo "✓ Git hooks 已安装到 .githooks/"
echo "  - pre-commit：docs/errors.md 同步 + golangci-lint fmt + go vet"
echo "  - pre-push  ：含 *.go 改动时跑 go build + go test"
echo ""
echo "  验证：git config core.hooksPath"
echo "  跳过：git commit --no-verify / git push --no-verify"
