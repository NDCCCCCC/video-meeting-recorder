# 构建与交叉编译指南

本项目支持 **Windows / Linux / macOS** 多平台构建，**交叉编译** 一条命令完成，无需目标机器安装 Go 环境。

## 1. 工具链要求

| 工具 | 最低版本 | 用途 |
|------|---------|------|
| Go | 1.25.0 | 后端编译（静态链接、纯 Go SQLite 驱动） |
| Node.js | 20+ | 前端构建 |
| npm | 10+ | 依赖管理 |
| OpenSSL | 任意 | 生成 SM4 / TLS 密钥 |

> Go 交叉编译使用 `modernc.org/sqlite`（纯 Go），**无需 CGO**，可避免交叉编译器依赖。

## 2. 快速开始

### 当前平台构建

```bash
# Unix (macOS / Linux / Git Bash)
./scripts/build.sh

# Windows (cmd.exe)
scripts\build.bat
```

默认产出 `bin/record-v2-<os>-<arch>[.exe]`，并在仓库根目录同时嵌入前端。

### 交叉编译（推荐）

```bash
# 编译所有 4 个目标（windows + linux，amd64 + arm64）
./scripts/build.sh all

# 编译指定目标
./scripts/build.sh linux/amd64
./scripts/build.sh windows/arm64,linux/arm64
```

支持的目标列表：

| 目标 | 平台 | 架构 | 输出文件名 |
|------|------|------|------------|
| `windows/amd64` | Windows | x86_64 | `record-v2-windows-amd64.exe` |
| `windows/arm64` | Windows | ARM64 (Surface Pro X 等) | `record-v2-windows-arm64.exe` |
| `linux/amd64` | Linux | x86_64 | `record-v2-linux-amd64` |
| `linux/arm64` | Linux | ARM64 (树莓派4+/5, AWS Graviton) | `record-v2-linux-arm64` |
| `darwin/amd64` | macOS | Intel | `record-v2-darwin-amd64` |
| `darwin/arm64` | macOS | Apple Silicon | `record-v2-darwin-arm64` |

### 跳过前端

如果前端已经构建过，可加快增量构建：

```bash
./scripts/build.sh --no-frontend all
scripts\build.bat --no-frontend windows/amd64
```

## 3. 手动构建（无脚本）

```bash
# 1. 前端构建
cd frontend && npm run build && cd ..

# 2. 复制到 embed 目录
rm -rf internal/frontend/dist
mkdir -p internal/frontend/dist
cp -r frontend/dist/. internal/frontend/dist/

# 3. 编译后端
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -trimpath -ldflags "-s -w" \
    -o bin/record-v2-linux-amd64 ./cmd/server
```

## 4. 输出说明

`bin/` 目录结构示例：

```
bin/
├── .gitkeep
├── record-v2-windows-amd64.exe      # Windows x86_64 主程序
├── record-v2-windows-arm64.exe      # Windows ARM64 主程序
├── record-v2-linux-amd64            # Linux x86_64 主程序（静态链接）
├── record-v2-linux-arm64            # Linux ARM64 主程序（静态链接）
├── config.yaml                      # Windows 配置模板
├── ffmpeg.exe                       # Windows ffmpeg（已包含）
├── ffprobe.exe                      # Windows ffprobe（已包含）
└── certs/                           # TLS 自签名证书（部署时生成）
```

> **Linux/macOS 部署时**：`ffmpeg.exe` / `ffprobe.exe` 不适用，需要把目标平台的 ffmpeg 二进制放入 `bin/` 目录，或在 `config.yaml` 中将 `ffmpeg.path` / `ffprobe_path` 指向系统的 ffmpeg。

## 5. 部署清单

部署一个完整可运行的实例需要：

- [ ] 对应平台的 `record-v2` 二进制
- [ ] `config.yaml`（已用环境变量注入 SM4 密钥、华为会议凭证等）
- [ ] `certs/server.crt` / `certs/server.key`（自签名，部署时重新生成）
- [ ] 对应平台的 `ffmpeg` / `ffprobe`
- [ ] `frontend/.env.production`（在 build 之后替换为生产环境变量）

### Linux 服务器快速部署

```bash
# 1. 上传产物
scp bin/record-v2-linux-amd64 user@server:/opt/record-v2/
scp -r certs/ user@server:/opt/record-v2/

# 2. 在服务器上启动
ssh user@server
export SM4_SECRET="<your-secret>"
export ALIYUN_ACCESS_KEY_ID="<your-key>"
/opt/record-v2/record-v2-linux-amd64
```

## 6. 常见问题

### Q1: 编译报 `cgo` 错误
本项目已用 `modernc.org/sqlite` 替代 CGO 驱动。确保：
- 设置 `CGO_ENABLED=0`
- 升级 Go 至 1.25+

### Q2: 前端 dist/index.html 找不到
`internal/frontend/embed.go` 通过 `//go:embed dist` 嵌入前端资源。编译前必须把 `frontend/dist/` 复制到 `internal/frontend/dist/`，脚本会自动处理。

### Q3: 部署到 ARM 设备时崩溃
确认下载的二进制是 `*-arm64` 而不是 `*-amd64`。Linux ARM 设备上可以用 `uname -m` 验证：`aarch64` = arm64。

### Q4: 编译后大小比预期大
未压缩二进制 ~50MB（包含 antd / charts / hls.js）。如需瘦身：
- 启用 Vite 代码分割（已在 `vite.config.ts` 配置）
- 使用 UPX 压缩（`upx --best --lzma bin/record-v2-linux-amd64`）

---

## 7. viper 环境变量绑定（Phase 17 P0 修复）

历史代码曾配置 viper 的环境变量前缀为 `RECORD`（即查找 `RECORD_*` 环境变量），但部署文档
与 `.env.example` 全部使用无前缀的 `SM4_SECRET` / `HLS_TOKEN_SECRET` 等名称，导致运维设置的
环境变量无法生效。Phase 17 已改为 `BindEnv` 显式映射（见 `internal/config/config.go:bindSecretEnv`）：

| 环境变量 | viper key |
|----------|-----------|
| `SM4_SECRET` | `auth.sm4_secret` |
| `HLS_TOKEN_SECRET` | `auth.hls_token_secret` |
| `HUAWEI_INSECURE_SKIP_VERIFY` | `huawei.insecure_skip_verify` |
| `HUAWEI_MIN_TLS_VERSION` | `huawei.min_tls_version` |

**重要**：上述环境变量名**大小写敏感**（viper 默认行为）。`config.yaml` 内可继续使用
`${SM4_SECRET:}` 等语法作为辅助注入路径，最终由显式 `BindEnv` 覆写。

---

最后更新：2026-07-30
