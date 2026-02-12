#!/bin/bash
# 视频会议录制系统 V2.0 - 构建脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 获取脚本所在目录的父目录（项目根目录）
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "========================================"
echo "  视频会议录制系统 V2.0 - 构建脚本"
echo "========================================"
echo ""

# 检测操作系统
OS=$(uname -s)
ARCH=$(uname -m)

# 1. 检查 Node.js
echo "[1/6] 检查 Node.js..."
if ! command -v node &> /dev/null; then
    echo -e "${RED}[错误] 未找到 Node.js，请先安装 Node.js${NC}"
    exit 1
fi
node --version
echo ""

# 2. 检查 Go
echo "[2/6] 检查 Go..."
if ! command -v go &> /dev/null; then
    echo -e "${RED}[错误] 未找到 Go，请先安装 Go${NC}"
    exit 1
fi
go version
echo ""

# 3. 构建前端
echo "[3/6] 构建前端..."
cd "$PROJECT_ROOT/frontend"
npm run build
if [ $? -ne 0 ]; then
    echo -e "${RED}[错误] 前端构建失败${NC}"
    exit 1
fi
echo -e "${GREEN}前端构建完成${NC}"
echo ""

# 4. 复制前端文件到 internal/frontend/dist（用于 go:embed）
echo "[4/6] 准备嵌入文件..."
EMBED_DIR="$PROJECT_ROOT/internal/frontend/dist"
rm -rf "$EMBED_DIR"
mkdir -p "$EMBED_DIR"
cp -r "$PROJECT_ROOT/frontend/dist/"* "$EMBED_DIR/"
echo -e "${GREEN}前端文件已复制到 $EMBED_DIR${NC}"
echo ""

# 5. 构建后端
echo "[5/6] 构建后端..."
cd "$PROJECT_ROOT"

# 设置编译参数
LDFLAGS="-s -w"
OUTPUT_DIR="$PROJECT_ROOT/bin"

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 根据操作系统设置输出文件名
case "$OS" in
    Linux*)
        BINARY_NAME="record-v2"
        ;;
    Darwin*)
        BINARY_NAME="record-v2-mac"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        BINARY_NAME="record-v2.exe"
        ;;
    *)
        BINARY_NAME="record-v2"
        ;;
esac

# 编译
echo "正在编译可执行文件..."
go build -ldflags "$LDFLAGS" -o "$OUTPUT_DIR/$BINARY_NAME" "$PROJECT_ROOT/cmd/server"
if [ $? -ne 0 ]; then
    echo -e "${RED}[错误] 后端构建失败${NC}"
    exit 1
fi
echo -e "${GREEN}后端构建完成${NC}"
echo ""

# 6. 复制配置文件
echo "[6/6] 复制配置文件..."
if [ -f "$PROJECT_ROOT/configs/config.yaml" ]; then
    cp "$PROJECT_ROOT/configs/config.yaml" "$OUTPUT_DIR/"
    echo -e "${GREEN}已复制 config.yaml${NC}"
fi

echo ""
echo "========================================"
echo "  构建完成！"
echo "========================================"
echo ""
echo -e "输出目录: ${GREEN}$OUTPUT_DIR${NC}"
echo -e "可执行文件: ${GREEN}$OUTPUT_DIR/$BINARY_NAME${NC}"
echo ""
echo "使用说明:"
echo "  1. 将 $OUTPUT_DIR 目录复制到目标服务器"
echo "  2. 运行: ./$BINARY_NAME"
echo "  3. 访问 http://localhost:8080 使用系统"
echo ""

# 设置可执行权限（Linux/macOS）
if [[ "$OS" =~ Linux|Darwin ]]; then
    chmod +x "$OUTPUT_DIR/$BINARY_NAME"
    echo -e "${GREEN}已设置可执行权限${NC}"
fi
