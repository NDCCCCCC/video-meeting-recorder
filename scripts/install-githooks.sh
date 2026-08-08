#!/usr/bin/env bash
# 安装项目 git hooks（pre-commit 自动同步 docs/errors.md）
#
# 用法：
#   ./scripts/install-githooks.sh
#
# 效果：
#   - git config core.hooksPath .githooks
#   - 让 .githooks/pre-commit 自动激活
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

# 确保所有 hook 可执行（从 git checkout 后权限可能丢失）
chmod +x .githooks/*

git config core.hooksPath .githooks

echo "✓ Git hooks 已安装到 .githooks/"
echo "  - pre-commit: docs/errors.md 同步校验（自动 stage 生成结果）"
echo ""
echo "  验证：git config core.hooksPath"
echo "  跳过单次 commit：git commit --no-verify"