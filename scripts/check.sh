#!/usr/bin/env bash
# scripts/check.sh — Record V2 一键本地全量检查（等价 CI 的本地版）
#
# 依次跑：mod 校验 / fmt 检查 / vet / golangci-lint / build / test(-race) / govulncheck / errors.md 同步
# 任一步失败即停；退出时打印汇总。用法：
#   bash scripts/check.sh
#
# 仅覆盖根模块（与 .github/workflows/ci.yml 一致）。scripts/check_db、scripts/fix_migration
# 是独立子 module，需各自 cd 进去单独检查。

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

# 测试密钥（见 ci.yml Test step）；允许外部覆盖。
export SM4_SECRET="${SM4_SECRET:-0123456789abcdef0123456789abcdef}"
export HLS_TOKEN_SECRET="${HLS_TOKEN_SECRET:-0123456789abcdef0123456789abcdef}"

# 颜色（非 TTY 退化为无色）
if [ -t 1 ]; then
  G=$'\033[32m'; R=$'\033[31m'; B=$'\033[1m'; X=$'\033[0m'
else
  G=''; R=''; B=''; X=''
fi

MAX=8
TOTAL=0
PASS=0
FAILED_STEP=""
START=$(date +%s)

step() {
  TOTAL=$((TOTAL+1))
  local t0; t0=$(date +%s)
  local label="$1"
  echo ""
  echo "${B}[$TOTAL/$MAX] $label${X}"
  shift
  if "$@"; then
    local t1; t1=$(date +%s)
    echo "${G}  ✓ OK${X} ($((t1-t0))s)"
    PASS=$((PASS+1))
  else
    echo "${R}  ✗ FAIL${X}" >&2
    FAILED_STEP="$label"
    return 1
  fi
}

# --- 单步检查（函数）-------------------------------------------------------

check_mod() {
  go mod tidy || return 1
  if ! git diff --quiet -- go.mod go.sum; then
    git checkout -- go.mod go.sum
    echo "  go.mod/go.sum 与 'go mod tidy' 结果不一致，请本地运行 go mod tidy 并提交" >&2
    return 1
  fi
}

check_errors_md() {
  go generate ./internal/errors/... || return 1
  if ! git diff --quiet docs/errors.md; then
    echo "  docs/errors.md 与 internal/errors 不同步，请运行 go generate ./internal/errors/..." >&2
    return 1
  fi
}

# --- 汇总（任何退出都打印）-------------------------------------------------
summarize() {
  local code=$?
  local end; end=$(date +%s)
  echo ""
  if [ -n "$FAILED_STEP" ]; then
    echo "${R}${B}失败：$FAILED_STEP${X}  ($PASS/$MAX 通过, $((end-START))s)" >&2
  else
    echo "${G}${B}全部通过${X}  ($PASS/$MAX, $((end-START))s)"
  fi
  exit "$code"
}
trap summarize EXIT

# --- 主流程 ----------------------------------------------------------------
echo "${B}Record V2 本地全量检查${X}"

step "go mod 校验"            check_mod
step "fmt 检查 (--diff)"      golangci-lint fmt --diff
step "go vet"                 go vet ./...
step "golangci-lint"          golangci-lint run --timeout=5m
step "go build ./..."         go build ./...
step "go test (-race)"        go test -race ./...
step "govulncheck"            govulncheck ./...
step "errors.md 同步"         check_errors_md
