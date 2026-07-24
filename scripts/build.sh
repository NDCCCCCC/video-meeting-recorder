#!/usr/bin/env bash
# 视频会议录制系统 V2.0 - 多平台构建脚本（Unix: macOS / Linux / Git Bash）
#
# 用法：
#   ./scripts/build.sh                    # 当前宿主平台
#   ./scripts/build.sh all                # 全部 4 个目标（windows/amd64, windows/arm64, linux/amd64, linux/arm64）
#   ./scripts/build.sh linux/amd64        # 指定目标
#   ./scripts/build.sh windows/amd64,linux/arm64  # 多个目标
#   ./scripts/build.sh --no-frontend all  # 跳过前端（前端已构建）
#
# 目标格式: <os>/<arch>  支持：windows/amd64, windows/arm64, linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
# 输出目录: ./bin/

set -euo pipefail

# ---- 颜色 ----
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ---- 路径 ----
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/bin"
EMBED_DIR="$PROJECT_ROOT/internal/frontend/dist"
FRONTEND_DIST="$PROJECT_ROOT/frontend/dist"

cd "$PROJECT_ROOT"

# ---- 参数 ----
BUILD_FRONTEND=1
TARGETS=()

for arg in "$@"; do
    case "$arg" in
        --no-frontend) BUILD_FRONTEND=0 ;;
        --help|-h)
            sed -n '2,12p' "$0"
            exit 0
            ;;
        all)
            TARGETS=(windows/amd64 windows/arm64 linux/amd64 linux/arm64)
            ;;
        *) TARGETS+=("$arg") ;;
    esac
done

# 默认目标：当前平台
if [[ ${#TARGETS[@]} -eq 0 ]]; then
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64) ARCH=amd64 ;;
        aarch64|arm64) ARCH=arm64 ;;
    esac
    TARGETS=("$OS/$ARCH")
fi

# ---- 工具链检查 ----
echo -e "${BLUE}== 工具链检查 ==${NC}"
command -v go >/dev/null 2>&1 || { echo -e "${RED}未找到 go${NC}"; exit 1; }
echo "  go: $(go version)"
if [[ $BUILD_FRONTEND -eq 1 ]]; then
    command -v node >/dev/null 2>&1 || { echo -e "${RED}未找到 node${NC}"; exit 1; }
    command -v npm >/dev/null 2>&1 || { echo -e "${RED}未找到 npm${NC}"; exit 1; }
    echo "  node: $(node --version)"
    echo "  npm: $(npm --version)"
fi
echo

# ---- 前端构建 ----
if [[ $BUILD_FRONTEND -eq 1 ]]; then
    echo -e "${BLUE}== 前端构建 ==${NC}"
    (cd "$PROJECT_ROOT/frontend" && npm run build) || { echo -e "${RED}前端构建失败${NC}"; exit 1; }

    echo -e "${BLUE}== 复制前端到 embed 目录 ==${NC}"
    rm -rf "$EMBED_DIR"
    mkdir -p "$EMBED_DIR"
    cp -r "$FRONTEND_DIST/." "$EMBED_DIR/"
    echo "  已复制到 $EMBED_DIR"
    echo
fi

if [[ ! -f "$EMBED_DIR/index.html" ]]; then
    echo -e "${YELLOW}[警告] $EMBED_DIR/index.html 不存在，编译出的二进制将不包含前端资源${NC}"
fi

# ---- 后端构建 ----
mkdir -p "$OUTPUT_DIR"

build_one() {
    local target="$1"
    local goos="${target%/*}"
    local goarch="${target#*/}"
    local ext=""
    local name="record-v2-${goos}-${goarch}"
    if [[ "$goos" == "windows" ]]; then
        ext=".exe"
    fi
    local out="$OUTPUT_DIR/${name}${ext}"

    echo -e "${BLUE}>> 构建 ${target} -> ${name}${ext}${NC}"
    GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
        go build -trimpath -ldflags "-s -w" \
        -o "$out" ./cmd/server
    echo -e "   ${GREEN}✓ ${out} ($(du -h "$out" | cut -f1))${NC}"
}

echo -e "${BLUE}== 后端构建 ==${NC}"
for t in "${TARGETS[@]}"; do
    build_one "$t"
done
echo

# ---- 复制当前平台配置（仅当目标含 darwin 或当前 OS 时）----
HOST_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
if [[ -f "$PROJECT_ROOT/config.yaml" ]] && [[ " ${TARGETS[*]} " == *" ${HOST_OS}/"* ]] || [[ ${#TARGETS[@]} -eq 1 && "${TARGETS[0]}" == "${HOST_OS}/"* ]]; then
    cp "$PROJECT_ROOT/config.yaml" "$OUTPUT_DIR/"
    echo -e "${GREEN}已复制 config.yaml 到 $OUTPUT_DIR/${NC}"
fi

# ---- 总结 ----
echo
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}  构建完成${NC}"
echo -e "${GREEN}=========================================${NC}"
echo "  输出目录: $OUTPUT_DIR"
ls -lh "$OUTPUT_DIR" | grep -E "record-v2-"
echo
echo "部署时：将对应平台的二进制（连同 ffmpeg / ffprobe）复制到目标服务器后直接运行。"
echo "  示例：scp $OUTPUT_DIR/record-v2-linux-amd64 user@server:/opt/record-v2/"
