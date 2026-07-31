# 生产环境部署配置

> 完整的构建与交叉编译说明见 [BUILD.md](./BUILD.md)，本文档仅覆盖**部署期**的环境配置。

## 密钥配置

### 1. 生成 SM4 密钥

使用强随机密钥生成器生成 SM4 密钥：

```bash
openssl rand -hex 16
```

或者使用项目提供的脚本：

```bash
go run scripts/gen_sm4_key.go
```

### 2. 设置后端环境变量

```bash
export SM4_SECRET="<生成的密钥>"
export ALIYUN_ACCESS_KEY_ID="<your-key>"
export ALIYUN_ACCESS_KEY_SECRET="<your-secret>"
export TYTW_APP_KEY="<your-tytw-key>"
```

完整可注入的环境变量列表见 `.env.example` 与 `config.yaml.example`。

### 3. 设置前端环境变量

将相同的密钥配置到前端环境变量：

```bash
echo "VITE_SM4_SECRET=<相同的密钥>" > frontend/.env.production
echo "VITE_API_URL=https://your-server:5443" >> frontend/.env.production
```

**重要**: 前后端密钥必须完全一致，否则登录将失败。

---

## TLS/HTTPS 配置

### 内网自签名证书生成

```bash
# 生成私钥
openssl genrsa -out ./certs/server.key 2048

# 生成自签名证书
openssl req -new -x509 -key ./certs/server.key -out ./certs/server.crt -days 365 \
  -subj "/C=CN/ST=State/L=City/O=Organization/CN=your-hostname"
```

> 注意：`certs/*.crt` / `certs/*.key` 已在 `.gitignore` 中忽略，需在部署时手动生成。

---

## 数据库初始化

首次启动时，后端会自动创建 SQLite 数据库文件 (`./data/record.db`)。

---

## 启动命令

### 方式一：使用预编译二进制（推荐）

```bash
# 从 build 目录复制对应平台的二进制
./bin/record-v2-linux-amd64      # Linux
./bin/record-v2-windows-amd64.exe # Windows
```

### 方式二：从源码运行（开发用）

```bash
# 后端（Go 自动识别目录包）
export SM4_SECRET="<your-sm4-secret>"
go run ./cmd/server

# 或者构建后运行
go build -o bin/dev-server ./cmd/server && ./bin/dev-server
```

### 前端开发服务器

```bash
cd frontend
echo "VITE_SM4_SECRET=<your-sm4-secret>" > .env.production
npm run build      # 一次性产出 dist/
# 或
npm run dev        # 开发热更新
```

---

## 环境变量清单

| 变量名 | 用途 | 必需 | 默认值 |
|--------|------|------|--------|
| `SM4_SECRET` | 后端 SM4 加密密钥 | 是 | (无默认，必须设置) |
| `HLS_TOKEN_SECRET` | HLS 直播令牌密钥 | 是 | (无默认) |
| `TYTW_APP_KEY` | 阿里通义听悟 | 否 | (空 = 关闭云端转录) |
| `ALIYUN_ACCESS_KEY_ID` | 阿里云 AK | 否 | (空) |
| `ALIYUN_ACCESS_KEY_SECRET` | 阿里云 SK | 否 | (空) |
| `HUAWEI_CONF_SERVER` | 华为会议服务器 | 否 | (空) |
| `HUAWEI_USERNAME` / `HUAWEI_PASSWORD` | 华为会议账号 | 否 | (空) |
| `VITE_API_URL` | 前端 API 地址 | 否 | (相对路径) |
| `VITE_SM4_SECRET` | 前端 SM4 密钥 | 是 | (必须与后端一致) |

---

## 安全检查清单

部署前请确认：

- [ ] SM4 密钥已从默认值更改为强随机密钥（`openssl rand -hex 16`）
- [ ] 前端环境变量 `VITE_SM4_SECRET` 已正确配置
- [ ] TLS/HTTPS 已启用，证书是部署时新生成的
- [ ] 密钥已妥善保管，未提交到版本控制系统
- [ ] 前端生产构建包含正确的密钥
- [ ] 防火墙规则正确配置
- [ ] 数据库文件（`data/`）已做备份计划

---

## 环境变量与启动校验

下表列出本版本（Phase 17 P0 之后）支持的密钥类环境变量及其启动校验行为：

| 变量名 | 必填 | 校验规则 | 缺失/不合规时的行为 |
|--------|------|----------|----------------------|
| `SM4_SECRET` | 是 | 最小 32 字符 | 生产环境：`logger.Fatal` 终止启动；非生产：Warn |
| `HLS_TOKEN_SECRET` | 是 | 最小 32 字符，且与 `SM4_SECRET` 互不相同 | 生产环境：`logger.Fatal` 终止启动；非生产：Warn |
| `HUAWEI_MIN_TLS_VERSION` | 否 | "1.2"（默认）/ "1.3"；TLS 1.0 强制提升为 1.2 | 非法值归一化为 TLS 1.2 |
| `HUAWEI_INSECURE_SKIP_VERIFY` | 否 | 默认 `false`；生产环境强制 `false` | 生产环境为 `true` 时：`logger.Fatal` 终止启动 |

### 启动校验行为

- **生产环境** (`server.environment == "production"`): 若 `SM4_SECRET` 或 `HLS_TOKEN_SECRET` 长度 < 32 字符、或两者相同、或 `HUAWEI_INSECURE_SKIP_VERIFY=true`，进程以 `logger.Fatal` 退出（不可降级为 warn）。
- **非生产环境**: 仅打印 `Warn` 级别日志，不阻止启动，便于本地开发与测试。
- **HLS Token 防重放**: 每次签发的 token 含一次性 `jti`；同一 jti 在进程生命周期内只能验证通过一次（重启后旧 token 失效一次后即作废）。

---

## 凭据密钥配置与轮换（Phase 18 operator runbook）

本节描述 operator 在生产环境**设置**与**轮换** `CREDENTIAL_SM4_*` 密钥的完整流程。
代码实现：`internal/services/credential_encryptor.go`、`cmd/server/app.go:Initialize()`。

### 关键概念

| 概念 | 说明 |
|------|------|
| `current` 密钥 | 当前用于加密新写入凭据的密钥；envelope 标记 `SM4:<current_ver>:<base64>` |
| `previous` 密钥 | 轮换过渡期保留的旧密钥，用于解密历史 envelope；过渡期结束后移除 |
| envelope version | 与密钥版本绑定的版本号（`v1` / `v2` / `v3` …），由 `CREDENTIAL_SM4_VERSION` 控制 |
| `RotateIfNeeded` | 启动期一次性把 previous 版本 envelope 改写为 current 版本的内部过程 |

### 三组密钥族分离

| 环境变量 | 用途 | 与凭据加密的关系 |
|---|---|---|
| `SM4_SECRET` | 浏览器 ↔ 后端 transport（SM4-ECB） | **必须独立**于 `CREDENTIAL_SM4_SECRET` |
| `HLS_TOKEN_SECRET` | HLS URL 签名 | **必须独立**于 `CREDENTIAL_SM4_SECRET` |
| `CREDENTIAL_SM4_SECRET` | **at-rest 凭据加密**（SM4-GCM） | 与上面两族互不相同 |

### 首次部署：设置凭据密钥

```bash
# 1. 生成三组独立的强随机密钥（每组至少 32 字符）
SM4_SECRET=$(openssl rand -hex 16)              # 32 chars (16 bytes hex)
HLS_TOKEN_SECRET=$(openssl rand -hex 16)
CREDENTIAL_SM4_SECRET=$(openssl rand -hex 16)
CREDENTIAL_SM4_VERSION=v1

# 2. 写入部署环境（systemd / k8s Secret / .env）
# ⚠️ 不要 commit 到仓库；使用部署平台的 secret 注入机制
export SM4_SECRET HLS_TOKEN_SECRET
export CREDENTIAL_SM4_SECRET CREDENTIAL_SM4_VERSION

# 3. 备份 DB（轮换前必须）
cp data/record.db data/record.db.backup-$(date +%Y%m%d-%H%M%S)

# 4. 第一次启动 —— MigratePlaintextToGCM 自动升级历史 plaintext / base64-stub
./bin/record-v2-linux-amd64

# 5. 启动日志关键观察点（operator 必须确认）：
#    INFO CredentialEncryptor 构造成功  current_version=v1 has_previous=false
#    INFO credential version counts  stage=after_migrate
#       column=input_configs.password by_version__v1=N1 ...
#       column=input_configs.stream_password by_version__v1=N2 ...
#       column=system_settings[auth.ad.password] by_version__v1=N3 ...
#    INFO credential version counts  stage=after_invariant （必须非空）
#    → N1+N2+N3 等于受保护凭据总数；non_envelope_rows=0；unknown_version_rows=0
```

### 密钥轮换（v1 → v2）

> 触发时机：定期（建议 90 天）、怀疑泄露、运维人员离职、加密算法升级。

```bash
# ═══════════════════════════════════════════════════════════
# 阶段 A：备份 + 准备新密钥
# ═══════════════════════════════════════════════════════════

# A1. 完整备份 DB（含 WAL）
cp data/record.db data/record.db.pre-rotate-v1to2-$(date +%Y%m%d-%H%M%S)
cp data/record.db-wal data/record.db-wal.pre-rotate-v1to2-$(date +%Y%m%d-%H%M%S) 2>/dev/null || true
sqlite3 data/record.db "PRAGMA wal_checkpoint(TRUNCATE);"   # 强制 WAL 落盘

# A2. 生成新的 current 密钥（同时保留旧密钥作为 previous）
NEW_CRED=$(openssl rand -hex 16)
# CREDENTIAL_SM4_SECRET（旧）从现有环境读取；确保它至少 32 字符
OLD_CRED="$CREDENTIAL_SM4_SECRET"

# A3. 暂存新环境变量（未部署到生产）
export CREDENTIAL_SM4_VERSION=v2
export CREDENTIAL_SM4_SECRET="$NEW_CRED"
export CREDENTIAL_SM4_PREVIOUS_VERSION=v1
export CREDENTIAL_SM4_PREVIOUS_SECRET="$OLD_CRED"

# ═══════════════════════════════════════════════════════════
# 阶段 B：滚动重启（确保所有旧实例停止后再启新实例）
# ═══════════════════════════════════════════════════════════

# B1. 停止旧实例（避免并发写导致 race）
systemctl stop record-v2 || pkill -f record-v2

# B2. 验证端口已释放
ss -lnt | grep :5443 || echo "port 5443 free"

# B3. 启动新实例（带 v2 + previous=v1）
./bin/record-v2-linux-amd64 &

# B4. 观察启动日志（10 秒内必须看到三条 LogVersionCounts）
sleep 5
journalctl -u record-v2 -n 50 | grep -E "credential version counts|凭据轮换完成"
# 期望：
#   INFO 凭据轮换完成  rotated=N  from=v1 to=v2
#   INFO credential version counts  stage=after_migrate   by_version__v1=N+M
#   INFO credential version counts  stage=after_rotate    by_version__v1=0 by_version__v2=N+M
#   INFO credential version counts  stage=after_invariant by_version__v1=0 by_version__v2=N+M

# B5. 功能 smoke（任一选一）
#    - curl https://server/api/v1/system/stats（返回 200 即基本 OK）
#    - 创建一个临时 input-config，确认 password 入库是 SM4:v2:... envelope

# ═══════════════════════════════════════════════════════════
# 阶段 C：移除 previous 密钥（轮换收尾）
# ═══════════════════════════════════════════════════════════

# C1. 确认 after_invariant 阶段 v1=0（v1 envelope 已全部归零）
#    若 v1>0，说明轮换未完成 —— 不要移除 previous，重启再跑一次

# C2. 解除 previous 环境变量
unset CREDENTIAL_SM4_PREVIOUS_VERSION
unset CREDENTIAL_SM4_PREVIOUS_SECRET

# C3. 再次重启（让应用只带 v2 启动，验证 v1 envelope 在没有 previous 密钥时不可解密）
systemctl restart record-v2
sleep 5
journalctl -u record-v2 -n 50 | grep -E "凭据 invariant scan 失败"
# 期望：无任何 invariant scan 失败日志

# C4. 备份 v1 时代的 DB（保留 N 个月符合合规要求）
mv data/record.db.pre-rotate-v1to2-* /backup/credential-rotation/

# C5. 安全处置旧密钥（v1 secret）
#    - 删除环境变量副本
#    - 删除部署平台 secret 历史版本
#    - 在密钥管理系统（AWS KMS / HashiCorp Vault）标记 v1 为 revoked
```

### 紧急回滚（轮换后 invariant 失败 / 业务异常）

```bash
# 紧急回滚路径：恢复轮换前的环境 + DB 备份
systemctl stop record-v2

# 还原 DB（从阶段 A1 备份）
cp data/record.db.pre-rotate-v1to2-* data/record.db
cp data/record.db-wal.pre-rotate-v1to2-* data/record.db-wal 2>/dev/null || true

# 还原环境变量（恢复 CREDENTIAL_SM4_VERSION=v1 + 移除 previous）
unset CREDENTIAL_SM4_PREVIOUS_VERSION
unset CREDENTIAL_SM4_PREVIOUS_SECRET
export CREDENTIAL_SM4_VERSION=v1
export CREDENTIAL_SM4_SECRET="$OLD_CRED"

# 启动
./bin/record-v2-linux-amd64 &
journalctl -u record-v2 -n 50 | grep "CredentialEncryptor 构造成功"
# 期望：current_version=v1 has_previous=false
```

### 备份与密钥保留策略

| 资产 | 保留期限 | 处置方式 |
|------|---------|---------|
| 轮换前 DB 备份 | ≥ 当前合规要求（建议 1 年） | 加密归档到对象存储，密钥独立保管 |
| 旧 `CREDENTIAL_SM4_SECRET`（v1）| 至少 1 个轮换周期（即到下次轮换前） | 在密钥管理系统标记 revoked；不要立即销毁以防需要回滚 |
| 新 `CREDENTIAL_SM4_SECRET`（v2）| 直至下次轮换 | 与 SM4_SECRET / HLS_TOKEN_SECRET 隔离存储 |
| WAL 文件（`.db-wal`）| 同 DB 备份 | 不可单独保留——必须与 `.db` 同步备份 |

### 监控指标（推荐接入 Prometheus / Loki）

Wave 4 新增的可观测信号：

- **每阶段 envelope 行数**：`credential_version_counts{column, stage, version}`（结构化日志字段）
- **轮换进度**：`after_rotate by_version__<previous> = 0` 是阶段性 OK 标志
- **invariant 失败计数**：任何一次启动 invariant scan 失败 → 高优告警
- **未知 version 计数**：`unknown_version_rows > 0` → 数据治理告警（可能存在历史遗留未迁移 envelope）

---

## 凭据存储的物理残留（physical remanence）

SQLite at-rest 加密只保护**在线逻辑视图**——一旦攻击者拿到物理介质（磁盘镜像、
备份磁带、文件系统快照、SSD 控制器缓存），残留风险来自以下渠道：

### 1. SQLite WAL（Write-Ahead Log）

- **风险**：`data/record.db-wal` 是 SQLite 在 `journal_mode=WAL` 下的预写日志。
  最近未 checkpoint 的写操作（包括 envelope 更新）会以**明文或旧 envelope**形式残留在 WAL 段。
- **缓解**：
  - 部署层定期 `PRAGMA wal_checkpoint(TRUNCATE)`（推荐每小时或备份前）；
  - 备份时**必须同时复制 `.db` 和 `.db-wal`**（详见 operator runbook 阶段 A1）；
  - 删除旧 DB 文件时使用 `srm` / `shred` 等工具，不要仅 `rm`。

### 2. SQLite VACUUM 与 free pages

- **风险**：删除的行只是标记 tombstone，未真正擦除；VACUUM 前，攻击者可能
  从 free page 恢复历史 envelope（甚至更早的 plaintext）。
- **缓解**：
  - 凭据表的 `password` / `stream_password` 列是 `TEXT`——每次 UPDATE 都写入
    新 page（SQLite 默认 append-only），旧 page 进入 free list；
  - 强烈建议每季度 `VACUUM` 一次（应用空闲时段），让 free page 重新分配；
  - VACUUM 后再次备份即为"重写"备份，残留面大幅缩小。

### 3. 文件系统快照（snapshot / LVM / ZFS）

- **风险**：LVM snapshot、ZFS snapshot、btrfs snapshot 都是 copy-on-write——一旦
  attach 给攻击者，原 DB 文件的所有历史 page 都在；
- **缓解**：
  - 快照保留期限对齐合规要求（建议 ≤ 30 天）；
  - 删除快照时使用文件系统厂商提供的"安全删除"（如 `zfs destroy -R`）；
  - 不要把含凭据的快照发到开发环境。

### 4. 备份介质退役（tape / SSD / cloud archive）

- **风险**：退役磁盘 / 磁带 / 云归档（Glacier / Coldline）仍含原 DB 文件；
  SSD 控制器缓存可能保留历史块。
- **缓解**：
  - **加密备份**：备份到对象存储前用服务端加密（SSE-KMS）+ 客户端加密
    （age / gpg 与备份密钥独立）；
  - SSD：使用厂商的 Secure Erase（hdparm / NVMe format with secure erase）；
  - 磁带：物理消磁或物理销毁；
  - 云归档：删除前显式调用 `s3 delete-objects` + 等待对象存储 tombstone 过期。

### 5. 内存残留（运行时 plaintext）

- **风险**：DB 文件是密文，但**运行时内存**必然含解密后的 plaintext——
  `/proc/<pid>/maps` + core dump 可能 dump 出凭据。
- **缓解**：
  - 启用 Linux `vm.mmap_min_addr=65536` + 内核 Yama `kernel.yama.ptrace_scope=3`
    限制跨进程 ptrace；
  - 关闭 core dump（`ulimit -c 0`）；如果必须保留，core dump 路径用 tmpfs；
  - Go runtime 在 GC 时会清零栈，但**不会主动清零堆**——敏感变量用完即覆盖：
    ```go
    for i := range plaintextBytes {
        plaintextBytes[i] = 0
    }
    ```
    （当前 CredentialEncryptor.Decrypt 返回 string 会立即复制，调用方负责处理。）

### 6. swap 与 hibernation

- **风险**：Linux swap 可能把内存中的 plaintext 换出到磁盘；
  hibernation（`/sys/power/disk`）在断电前把 RAM dump 到 swap。
- **缓解**：
  - 生产服务器：关闭 swap（`swapoff -a`）或使用加密 swap（`cryptsetup`）；
  - BIOS 禁用 hibernation / S3 sleep；
  - 容器化部署：关闭 swap（默认 kubelet 不挂载 swap）。

### 7. 监控与告警

- 备份文件大小异常缩小 → 可能未经授权的 VACUUM / 文件截断；
- WAL 文件长时间未 checkpoint → 可能存在写入阻塞或异常进程；
- SSD SMART 数据擦除计数异常 → 可能存在外部擦除尝试。

---

*部署文档版本: 1.3*
*最后更新: 2026-07-31*
