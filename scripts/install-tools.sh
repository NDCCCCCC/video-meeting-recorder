#!/usr/bin/env bash
# 安装 Go 开发工具链：golangci-lint + govulncheck。
#
# 用法：
#   ./scripts/install-tools.sh
#
# 安装后请确保 "$(go env GOPATH)/bin" 在 PATH 中。

set -euo pipefail

# golangci-lint v2.12.2：需用 go1.25 构建，才能分析 go1.25 代码
# （见 .github/workflows/ci.yml 中 golangci-lint step 的注释）。
echo "安装 golangci-lint v2.12.2 ..."
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2

echo "安装 govulncheck ..."
go install golang.org/x/vuln/cmd/govulncheck@latest

GOBIN="$(go env GOPATH)/bin"
echo ""
echo "✓ 安装完成。请确保以下目录在 PATH 中："
echo "  $GOBIN"
case ":$PATH:" in
  *":$GOBIN:"*) echo "  ✓ 已在 PATH" ;;
  *) echo "  ⚠️ 当前 PATH 未含该目录，请将其加入 shell 配置（.bashrc/.zshrc）或系统环境变量" ;;
esac
