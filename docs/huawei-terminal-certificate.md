# 华为终端证书导入与 TLS 配置运维手册

> **适用版本**: Record V2 主分支（2026-08-07 之后的版本，含 `huawei.tls_server_name` 配置项）
> **触发场景**: 客户端通过 IP 直连华为 TE40 终端时出现
> `x509: certificate signed by unknown authority` 或
> `x509: cannot validate certificate for 10.62.10.3 because it doesn't contain any IP SANs`
> **核心约束**: 不得使用 `huawei.insecure_skip_verify: true`（生产 fail-closed）。

---

## 1. 背景

华为 TE40 终端使用私有 CA（CN=`huawei_ca`）签发 HTTPS 服务证书。客户端以 IP 地址（`10.62.10.3`）直连时，TLS 握手要同时通过两道校验：

| 校验门 | 失败现象 | 修复手段 |
|--------|---------|---------|
| 信任链（Trust Anchor） | `x509: certificate signed by unknown authority` | 导入 `huawei_ca` 自签根到 `huawei.ca_bundle_file` |
| Hostname/SAN | `x509: cannot validate certificate for 10.62.10.3 because it doesn't contain any IP SANs` | `huawei.tls_server_name` 显式设 `vct.tp.huawei.com`（与证书 DNS SAN `*.tp.huawei.com` 匹配）|

实际终端证书 SAN 只有 `DNS:*.tp.huawei.com`，**没有 iPAddress**。因此只导入 CA 不足以通过校验——必须同时显式指定 ServerName。

---

## 2. 终端证书链结构

| 文件 | subject | issuer | 用途 |
|------|---------|--------|------|
| `huawei-10.62.10.3-chain-1.pem` | `CN=vct.tp.huawei.com` | `CN=huawei_ca` | 终端服务证书（leaf） |
| `huawei-10.62.10.3-chain-2.pem` | `CN=huawei_ca` | `CN=huawei_ca` | 自签 trust anchor（root） |

校验命令：

```bash
openssl x509 -in chain-1.pem -noout -subject -issuer -ext subjectAltName
# 预期: subject=CN=vct.tp.huawei.com, issuer=CN=huawei_ca, SAN=DNS:*.tp.huawei.com

openssl x509 -in chain-2.pem -noout -subject -issuer
# 预期: subject==issuer==CN=huawei_ca

openssl verify -CAfile chain-2.pem chain-1.pem
# 预期: chain-1.pem: OK
```

---

## 3. 三种部署方案（按运维改造成本递增）

### 方案 A — 推荐：替换 ca_bundle_file + 配置 tls_server_name（最低改造成本）

**原理**：用自签根（chain-2）替换原 ca_bundle_file 中的 leaf，再显式告诉客户端按 DNS SAN 做 hostname 校验。

#### 步骤

1. **备份原始 CA bundle**
   ```bash
   cd /path/to/record_v2
   cp certs/huawei-10.62.10.3-ca.pem \
      certs/huawei-10.62.10.3-ca.leaf-backup-$(date +%Y%m%d).pem
   ```

2. **替换为自签根**
   ```bash
   cp certs/huawei-10.62.10.3-chain-2.pem certs/huawei-10.62.10.3-ca.pem
   ```

3. **编辑 `config.yaml` 的 `huawei` 段**
   ```yaml
   huawei:
     insecure_skip_verify: false       # 必须保持 false（fail-closed）
     ca_bundle_file: "./certs/huawei-10.62.10.3-ca.pem"
     tls_server_name: "vct.tp.huawei.com"
   ```
   或使用环境变量覆盖：
   ```bash
   export HUAWEI_CA_BUNDLE_FILE="./certs/huawei-10.62.10.3-ca.pem"
   export HUAWEI_TLS_SERVER_NAME="vct.tp.huawei.com"
   ```

4. **重启服务**
   ```bash
   ./scripts/build.sh windows/amd64
   # 把 bin/record-v2-windows-amd64.exe + bin/config.yaml + certs/ 同步到生产
   # 在维护窗口重启服务
   ```

5. **启动日志验证**
   ```bash
   grep "华为 TLS" logs/server.log
   # 预期:
   #   华为 TLS CA bundle 加载成功 ... path=.../huawei-10.62.10.3-ca.pem cert_count=1
   #   华为 TLS hostname ServerName 已配置 ... tls_server_name=vct.tp.huawei.com
   ```

6. **业务验收**：触发"锁定终端"，应返回成功，不再出现 `unknown authority` 或 `IP SANs` 错误。

#### 风险与回滚

- **风险**：替换 ca_bundle_file 后，原 leaf 被覆盖。如未来需 leaf 做手工 verify，需要从备份恢复。
- **回滚**：
  ```bash
  cp certs/huawei-10.62.10.3-ca.leaf-backup-YYYYMMDD.pem \
     certs/huawei-10.62.10.3-ca.pem
  # 同时把 config.yaml 的 tls_server_name 留空,或还原旧二进制
  ```

---

### 方案 B — 不改 PEM，通过内部 DNS 把 hostname 解析到 IP

**原理**：让 dial 地址本身就是 hostname，使 Go x509 verifier 直接用 DNS SAN 匹配，无需显式 ServerName。

#### 步骤

1. **保持 ca_bundle_file 不变**（仍指向原 leaf 或 chain-2 自签根）
2. **运维在内部 DNS 添加 A 记录**
   ```
   vct.tp.huawei.com  →  10.62.10.3
   ```
3. **修改 `config.yaml`**
   ```yaml
   huawei:
     conference_server: "vct.tp.huawei.com"   # 由 IP 改为 hostname
     ca_bundle_file: "./certs/huawei-10.62.10.3-ca.pem"
     tls_server_name: ""                       # 留空,Go 用 dial 地址(此时已是 hostname)
   ```
4. **重启服务**（同方案 A 步骤 4–6）

#### 前置条件

- 内部 DNS / 内网 DNS 服务器可写入 `vct.tp.huawei.com` 的 A 记录
- 应用进程的 DNS resolver 优先指向该 DNS（不能走公网 8.8.8.8 / 1.1.1.1）
- `vSphere / 防火墙策略`允许应用服务器访问 `10.62.10.3:443`（hostname 解析后真实 IP）

#### 风险与回滚

- **风险**：DNS 解析是动态的；DNS 故障 / 缓存污染会立即影响"锁定终端"功能。
- **回滚**：删除 DNS A 记录 + 还原 `conference_server` 为 IP + 还原 `tls_server_name` 为 `vct.tp.huawei.com`。

---

### 方案 C — 联系华为 / 内部 CA 重新签发含 IP SAN 的服务器证书（最稳）

**原理**：让证书 SAN 同时包含 `vct.tp.huawei.com` (DNS) 和 `10.62.10.3` (iPAddress)，代码侧完全无需 `tls_server_name`。

#### 步骤

1. **生成 CSR**
   ```bash
   openssl req -new -newkey rsa:2048 -nodes \
     -subj "/C=CN/ST=GuangDong/O=Huawei/CN=vct.tp.huawei.com" \
     -keyout vct.key -out vct.csr \
     -addext "subjectAltName=DNS:vct.tp.huawei.com,IP:10.62.10.3"
   ```
2. **交给华为 / 内部 CA 签发**（使用现有 `huawei_ca` 或新内部 CA）
3. **部署新 leaf**
   ```bash
   # 替换 chain-1.pem 中的 leaf
   cp <new-leaf>.pem certs/huawei-10.62.10.3-chain-1.pem
   ```
4. **修改 `config.yaml`**
   ```yaml
   huawei:
     conference_server: "10.62.10.3"            # 仍可用 IP
     ca_bundle_file: "./certs/huawei-10.62.10.3-ca.pem"  # 仍是 huawei_ca 自签根
     tls_server_name: ""                        # 留空,Go 用 dial address 命中 IP SAN
   ```
5. **重启服务**（同方案 A 步骤 4–6）

#### 风险与回滚

- **风险**：需要外部流程（华为 / CA 团队），变更周期长（天–周级）。
- **回滚**：从备份恢复原 `chain-1.pem` + 还原 `tls_server_name`。

---

## 4. 方案选择决策表

| 条件 | 推荐方案 |
|------|---------|
| 需要立即修复且无法等待外部流程 | **A** |
| 已有内部 DNS 服务且希望零 PEM 变更 | **B** |
| 长期方案 / 证书即将过期 / 需要更稳的 SAN 覆盖 | **C** |
| 终端未来可能迁移到不同 IP | **A** 或 **C**（B 方案需重新配 DNS） |
| 安全合规要求 SAN 含 IP（部分审计场景） | **C**（A 方案的 SAN 仅含 DNS） |

**多数生产场景下选 A**：改造成本低、与现有部署模板兼容、不依赖外部流程。

---

## 5. 配置参考

### `config.yaml`（gitignored，部署物）

```yaml
huawei:
  conference_server: "10.62.10.3"
  conference_port: 80
  username: "<api_user>"
  password: "<env:HUAWEI_PASSWORD>"
  https: true
  insecure_skip_verify: false                # 必须 false
  api_timeout: "30s"
  session_timeout: "3600s"
  keep_alive_interval: "300s"
  min_tls_version: "1.2"
  ca_bundle_file: "./certs/huawei-10.62.10.3-ca.pem"
  tls_server_name: "vct.tp.huawei.com"        # 与证书 DNS SAN 匹配
```

### 环境变量（优先级高于 config.yaml）

| 变量 | 等价配置键 | 默认 |
|------|------------|------|
| `HUAWEI_INSECURE_SKIP_VERIFY` | `huawei.insecure_skip_verify` | `false`（生产 fail-closed） |
| `HUAWEI_MIN_TLS_VERSION` | `huawei.min_tls_version` | `"1.2"` |
| `HUAWEI_CA_BUNDLE_FILE` | `huawei.ca_bundle_file` | `"./certs/huawei-10.62.10.3-ca.pem"` |
| `HUAWEI_TLS_SERVER_NAME` | `huawei.tls_server_name` | `""`（Go 用 dial 地址） |

`tls_server_name` 留空时，**仅当**证书 SAN 含 `iPAddress:10.62.10.3` 才可通过校验。否则必须填写与 DNS SAN 匹配的字面名（如 `vct.tp.huawei.com`）。

---

## 6. 验证步骤

### 6.1 部署前验证（推荐在维护窗口前离线做）

```bash
# 1. 自签根的 subject / issuer / 有效期
openssl x509 -in certs/huawei-10.62.10.3-ca.pem -noout \
  -subject -issuer -dates -fingerprint -sha256
# 预期: subject=issuer=CN=huawei_ca, notAfter 在未来

# 2. chain 验证（需要 chain-1.pem 存在；生产目录可能缺，临时从终端抓取）
openssl verify -CAfile certs/huawei-10.62.10.3-ca.pem certs/huawei-10.62.10.3-chain-1.pem
# 预期: chain-1.pem: OK
```

### 6.2 TLS 握手在线探测（部署后必做）

```bash
openssl s_client \
  -connect 10.62.10.3:443 \
  -servername vct.tp.huawei.com \
  -verify_hostname vct.tp.huawei.com \
  -CAfile certs/huawei-10.62.10.3-ca.pem \
  -verify_return_error \
  -tls1_2 </dev/null
# 预期:
#   Verify return code: 0 (ok)
#   ---
#   New, TLSv1.2, ...
```

`-verify_hostname vct.tp.huawei.com` 必须与 `-servername` 保持一致；任一缺失都会绕过 hostname 校验，验证无效。

### 6.3 业务验证

```bash
# 通过前端 / API 触发"锁定终端"，然后检查:
tail -f logs/server.log
# 预期:
#   华为 TLS CA bundle 加载成功 path=.../huawei-10.62.10.3-ca.pem cert_count=1
#   华为 TLS hostname ServerName 已配置 tls_server_name=vct.tp.huawei.com
#   创建华为终端客户端成功 ...
#   (业务侧)锁定终端成功

# 反证指纹（不应出现）
grep -E "unknown authority|doesn't contain any IP SANs" logs/server.log
# 预期: 无输出
```

---

## 7. 故障排查

| 现象 | 可能原因 | 修复 |
|------|---------|------|
| `x509: certificate signed by unknown authority` | ca_bundle_file 指向 leaf 而非自签根 | 改用 chain-2.pem 内容（方案 A 步骤 2） |
| `x509: cannot validate certificate for 10.62.10.3 because it doesn't contain any IP SANs` | 未设置 `tls_server_name` 或证书不含 IP SAN | 方案 A：配置 `tls_server_name: "vct.tp.huawei.com"`；方案 C：重签证书 |
| `tls: failed to verify certificate: x509: certificate is valid for *.tp.huawei.com, not 10.62.10.3` | `tls_server_name` 与证书不匹配 / DNS 解析错误 | 检查 `tls_server_name` 字面值与 DNS 解析结果（`nslookup vct.tp.huawei.com`）|
| `x509: certificate has expired or is not yet valid` | 终端证书已过期（自签根 valid 到 2049 通常不会） | 联系华为 / CA 续签（方案 C 路径） |
| `配置文件解析错误: yaml: ...` | ca_bundle_file 路径含中文或 YAML 转义字符 | 用绝对路径或 `file://` URL；删除多余引号 |
| 启动后 grep 不到"华为 TLS"日志 | 二进制未更新 / 服务未重启 / 服务读到的是旧 config | 确认 `bin/` 二号产物 SHA-256；检查 `config.yaml` 是否同步到 `bin/` |

### `openssl verify` 在生产目录失败的常见误判

```bash
openssl verify -CAfile certs/huawei-10.62.10.3-ca.pem certs/huawei-10.62.10.3-chain-1.pem
# 错误: Can't open certs/huawei-10.62.10.3-chain-1.pem
#       No such file or directory
```

这是文件系统 ENOENT，**不是**证书链验证失败。生产部署可能只保留了 `ca.pem`，没保留 `chain-1.pem`。请改用 §6.2 的 `openssl s_client` 在线探测，或临时从终端抓取 leaf：

```bash
# 用 openssl s_client -showcerts 抓取终端返回的完整 PEM chain
echo | openssl s_client -connect 10.62.10.3:443 -servername vct.tp.huawei.com \
  -CAfile certs/huawei-10.62.10.3-ca.pem -showcerts 2>&1 \
  | sed -n '/-----BEGIN CERTIFICATE-----/,/-----END CERTIFICATE-----/p' \
  > /tmp/live-leaf.pem

# 然后离线 verify
openssl verify -CAfile certs/huawei-10.62.10.3-ca.pem /tmp/live-leaf.pem
# 预期: /tmp/live-leaf.pem: OK
```

**注意**：不要把抓取的 live leaf 追加到 `ca_bundle_file`——leaf 与 trust anchor 角色不同，混在一起会让 Go 误把 leaf 当 trust anchor 处理。

---

## 8. 回滚清单

| 步骤 | 命令 / 操作 |
|------|------------|
| 1. 恢复原 ca_bundle_file | `cp certs/huawei-10.62.10.3-ca.leaf-backup-YYYYMMDD.pem certs/huawei-10.62.10.3-ca.pem` |
| 2. 清空 `tls_server_name` | 把 `config.yaml` 的 `huawei.tls_server_name` 改为 `""`，或 `unset HUAWEI_TLS_SERVER_NAME` |
| 3. 回滚二进制 | 从备份还原旧 `server.exe`（如 `server.pre-huawei-tls-20260807.exe`） |
| 4. 重启服务 | 在维护窗口重启 |
| 5. 触发"锁定终端"验证 | 期望失败信息回到先前状态（确认回滚生效） |

---

## 9. 安全约束

- **不得**使用 `insecure_skip_verify: true` 跳过校验：生产环境 SEC-003a fail-closed，配置 true 会触发 `logger.Fatal` 终止进程。
- **不得**把终端 leaf 当 trust anchor 加入 `ca_bundle_file`——必须用 chain-2 自签根。
- **不得**在仓库中提交 `certs/*.pem`（gitignored）——备份文件统一带日期后缀放在 `certs/` 下，避免污染版本控制。
- ca_bundle_file 路径支持环境变量展开（`${VAR:default}`），但推荐使用绝对路径以避免 `go test` / 服务进程工作目录差异导致的 ENOENT。
- 升级 Go 版本时，需回归测试 `TestNewHTTPClient_NoServerName_IPClient_FailsHostnameSAN`：错误文本可能从 `doesn't contain any IP SANs` 变成 `is valid for ..., not ...`，但指纹（包含 `x509`、不包含 `unknown authority`）保持不变。

---

## 10. 参考

- `.planning/debug/resolved/huawei-tls-after-ca-fix.md` — 完整诊断过程、TDD RED/GREEN 证据、撤回记录
- `.planning/debug/resolved/huawei-tls-private-ca.md` — Phase 26 SEC-003a SetCABundle 引入的初始修复
- `internal/huawei/manager.go` — `SetTLSServerName` 实现 + 并发安全约束
- `internal/huawei/client.go` — `NewHTTPClient` ServerName 透传 + tls.Config 装配
- `internal/config/config.go` — `HuaweiConfig.TLSServerName` 字段 + `BindEnv` 注册
- `config.yaml.example` — `huawei.tls_server_name` 注释模板