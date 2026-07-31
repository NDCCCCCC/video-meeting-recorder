# SECURITY.md

Record V2 项目安全策略与上传前检查清单。

## ⚠️ 历史密钥事件

### 事件 #2: Release v2.0.0 二进制泄露密钥 (2026-07-24)

**发现**: 远端 release v2.0.0 二进制资产 (`record-v2-windows-amd64.exe` 等 4 个) 通过 Go embed 嵌入了前端构建产物，而前端的 `frontend/.env.production` 在 build 时被 Vite 注入，含有真实 SM4 密钥和内网 IP。

**影响**:
- 任何下载 release 资产的人都能获取 SM4 密钥
- 知道内网 API 地址（10.62.0.123）

**响应**:
1. 重新生成 SM4 密钥（`openssl rand -hex 16`）
2. 重新构建前端 + 4 平台二进制
3. 通过 GitHub API 删除旧 release 资产，上传新资产
4. **旧密钥已废止，所有部署需更新 `config.yaml` 与 `frontend/.env.production`**

**新密钥**: 通过环境变量注入，仓库中**永远不存真实值**。

### 事件 #1: 初始清理 (2026-07-24)

发现以下文件曾被提交进 git 历史，已通过 `git-filter-repo` 全部清除：

| 文件 | 暴露内容 | 风险 |
|------|---------|------|
| `.env` | 真实阿里通义听悟 APP_KEY | 高 |
| `config.yaml` | 真实 SM4 加密密钥 | 高 |
| `frontend/.env.development` | 真实 SM4 密钥 | 高 |
| `frontend/.env.production` | SM4 密钥 + 内网 IP | 高 |
| `certs/server.crt`, `certs/server.key` | TLS 私钥 | 中 |
| `scripts/test_sm4_token.go` 等 | 调试脚本 | 低 |
| `.claude/worktrees/...` | agent 临时工作副本 | 低 |

**重要**：所有上述文件的本地副本**未删除**（仍可用于本地开发），但仓库中的所有历史记录已通过 filter-repo 重写。向远端推送前请按本文下方的"推送前清单"逐项确认。

## 受保护的文件

以下文件类型应通过 `.gitignore` 阻止提交：

```
.env                # 后端环境变量
.env.*.local        # 本地覆盖配置
config.yaml         # 后端配置文件（含密钥）
frontend/.env.development
frontend/.env.production
certs/*.crt *.pem *.key  # TLS 证书与私钥
.claude/worktrees/      # agent 临时工作树
```

`.env.example` 与 `config.yaml.example` 始终被跟踪（仅占位）。

## 推送前清单

任何新增/修改后推送远端前请逐项确认：

- [ ] `git status` 不应包含 `.env` / `config.yaml` / `frontend/.env.*`
- [ ] `git log -p HEAD` 无新增密钥字符串
- [ ] 仅提交 `*.example` 模板和源代码
- [ ] 真实密钥通过环境变量或部署脚本注入，不要写入仓库
- [ ] 新的内网 IP、桶名、域名不应出现在仓库中
- [ ] TLS 私钥只放在部署机上，不进仓库

可用验证命令：

```bash
# 检查工作区是否有敏感文件
git ls-files | grep -iE "(\\.env$|config\\.yaml$|certs/|\\.key$|\\.crt$)"

# 检查历史中是否还有敏感关键字（输出应为空）
git rev-list --all --objects | grep -E "(TYTW_APP_KEY|sm4_secret|<redacted-old-sm4-key>)"

# 检查源代码中是否有硬编码密钥
grep -rIn -E "(VITE_SM4_SECRET|sm4_secret|TYTW_APP_KEY)" --include="*.go" --include="*.ts" \
  --include="*.tsx" --include="*.vue" \
  --exclude-dir=node_modules --exclude-dir=.git --exclude-dir=dist
```

## 部署时的密钥注入

推荐通过环境变量注入真实密钥，而非将 `.env` / `config.yaml` 复制到服务器：

```bash
# 后端：使用环境变量
export SM4_SECRET="$(openssl rand -hex 16)"
export ALIYUN_ACCESS_KEY_ID="..."
export ALIYUN_OSS_BUCKET="..."
export HUAWEI_CONF_SERVER="..."
./record-v2.exe

# 前端：构建时注入
echo "VITE_SM4_SECRET=$SM4_SECRET" > frontend/.env.production
cd frontend && npm run build
```

部署文档：`DEPLOYMENT.md`

## SECRET 校验

生产环境启动时（`server.environment == "production"`），后端对关键密钥执行强制校验：

- `SM4_SECRET` 长度 ≥ 32 字符；
- `HLS_TOKEN_SECRET` 长度 ≥ 32 字符，且与 `SM4_SECRET` 互不相同。

若任一校验失败，进程以 `logger.Fatal` 退出（不可降级为 warn）。非生产环境仅打印 `Warn` 级别日志，不阻止启动。详见 `internal/config/config.go:ValidateProductionSecrets`。

历史上代码曾存在 `change-me-in-production` 硬编码 fallback，已在 Phase 17 中删除——不再有"忘记配置也能启动"的危险路径。

## HLS Token 安全

HLS Token（`internal/auth/hlstoken/hls_token.go`）当前的安全保证：

- **密钥长度**：构造函数 `NewHLSToken` 强制密钥 ≥ 32 字符，否则 `panic`。
- **HMAC 编码**：签发使用 `base64.RawURLEncoding`（无 padding）。`Verify()` 同时接受 `RawURLEncoding` / `URLEncoding` / `StdEncoding` 三种签名编码，保证**重启后旧 token 仍可验证一次**（D-03.3 向后兼容承诺）。
- **防重放**：每次签发的 token 含 16 字节随机 `jti` 字段；同一 jti 在进程生命周期内只能验证通过一次，第二次返回 `ErrTokenReplayed`。局限：进程重启后记录清零；服务端持久化 `used_jtis`（Redis/DB）列入下个独立 phase。

## TLS 最低版本

华为会议系统 TLS 客户端（`internal/huawei/manager.go`）：

- **`MinTLSVersion` 默认 `tls.VersionTLS12`**（TLS 1.2 强制最低）；
- **`InsecureSkipVerify` 默认 `false`**（证书校验启用）；
- **生产环境 `HUAWEI_INSECURE_SKIP_VERIFY=true` → `logger.Fatal`**（defense-in-depth）；
- **密码套件**：`CipherSuites` 已剔除基于 3DES 的弱套件（SWEET32 攻击面），优先 ECDHE 前向保密套件（AES-GCM、CHACHA20），并保留 RSA-AES 兼容华为老设备；
- **入站 HTTP 调用**：`removeClient` 已透传调用方 `ctx`，不再使用 `context.Background()`，可被优雅退出级联。

历史残留的 `MinTLSVersion: 0x0301`（TLS 1.0）、`InsecureSkipVerify: true`、3DES cipher 硬编码均已在 Phase 17 中移除。

## 恢复操作

如果未来需要恢复某个被 filter-repo 删除的文件内容（误删），可从原始备份恢复：

```bash
# 本地保留了过滤前的 reflog 和分支 BACKUP_PRE_CLEANUP（如有）
git reflog | head -20
git show <old-commit-sha>:.env
```

## 凭据静态加密 (Phase 18)

Phase 18 引入 at-rest 凭据加密，与现有 SM4-ECB 传输加密严格分离：

### 算法选择

- **SM4-GCM (NIST SP 800-38D)** —— SM4 分组密码 + GCM 认证加密模式 (AEAD)。
- 关键参数：12 字节 nonce（96 位，按 NIST 800-38D 推荐）+ 16 字节 tag（128 位）。
- 输入明文先用 PKCS#7 补到 16 字节边界（gmsm v1.4.1 GCM 解密路径仅支持块对齐的密文）。
- tag 由 gmsm 解密分支计算 expected tag，调用方做**常量时间比对**（防 timing attack）。

### Envelope 格式

```
SM4:<version>:<base64(nonce_12B | ciphertext | tag_16B)>
```

- `version` 段（如 `v1`、`v2`）用于密钥轮换；
- 解析失败 / 未知 version / tag 校验失败 → 立即报错（**永不静默跳过**）。

### 密钥族分离

| 用途 | 环境变量 | 用途范围 |
|---|---|---|
| 浏览器传输加密 | `SM4_SECRET` | 前端 `sm-crypto` SM4-ECB → 前后端 password transport |
| HLS Token 签名 | `HLS_TOKEN_SECRET` | HLS URL 短期签名密钥 |
| **凭据静态加密** | `CREDENTIAL_SM4_SECRET` | SQLite 中 `input_configs.password / stream_password` 等 at-rest 字段 |

三组密钥**必须互不相同**，缺失或过短（< 32 字符）将触发生产环境 `logger.Fatal` 终止启动。

### 启动期 fail-closed 流程

`cmd/server/app.go:Initialize()` 严格按 10 步执行，任何一步失败立即返回 error、不进入 HTTP serve：

1. `LoadConfig` + `ValidateCredentialSM4Config`（main.go）
2. `initDatabase`（AutoMigrate + ALTER COLUMN 扩 password 列到 TEXT）
3. 构造 `CredentialEncryptor`（含 current + optional previous 密钥）
4. `MigratePlaintextToGCM`（事务内：plaintext/base64-stub → envelope；剥离 auth.ad JSON 的 password 字段；Unscoped 包含软删除行）
5. 第一次 `InvariantScan`（所有密文为合法 envelope 且可解密；auth.ad JSON 不含 password）
6. `RotateIfNeeded`（previous 版本 envelope → current 版本）
7. 第二次 `InvariantScan`（确保轮换后全部为 current）
8. `initRouter` → `checkPythonDependencies` → `initHandlers` → `registerRoutes` → `registerServices`
9. 启动 HTTP

### 威胁模型

| 边界 | 在范围内 | 不在范围内 |
|---|---|---|
| **应用进程内** | DB 连接 / SQL 注入 / 数据备份导出 | 前端浏览器 JS 注入 |
| **DB 文件**（at-rest） | **保护**：DB 文件泄露（含 backup）不会暴露明文凭据 | DB 内已解密运行中内存 |
| **磁盘快照 / WAL** | **保护**：SQLite `.db` + `.db-wal` 拿走后仍为密文 | WAL checkpoint 期间（部分明文可能在 WAL 段；备份策略见 DEPLOYMENT.md） |
| **网络传输** | 前端 → 后端 password transport 由 SM4-ECB 单独保护 | TLS 终止（部署侧责任） |

### 轮换（密钥族升级）

1. operator 设置 `CREDENTIAL_SM4_VERSION=v2`、`CREDENTIAL_SM4_SECRET=<new>`、
   `CREDENTIAL_SM4_PREVIOUS_VERSION=v1`、`CREDENTIAL_SM4_PREVIOUS_SECRET=<old>`。
2. 启动后 `RotateIfNeeded` 自动把所有 v1 envelope 改写成 v2。
3. 下次启动移除 `*_PREVIOUS_*` 字段即可完成轮换。

详见 `internal/services/credential_encryptor.go`、`internal/utils/sm4_password.go`。

## 报告安全问题

发现新的安全风险，请勿在公开 issue 中暴露细节，联系仓库管理员。

---

最后更新：2026-07-24
