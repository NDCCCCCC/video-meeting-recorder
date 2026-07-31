# Phase 18 Summary: 凭据静态加密 + 密钥轮换 (SM4-GCM at-rest)

**Phase:** 18
**Subsystem:** backend (security / data layer)
**Tags:** security, encryption, migration, refactor
**Date:** 2026-07-31
**Base HEAD:** `e294ae9` (phase 17 final)
**Final HEAD:** `bd84fe2`

## 概览

按用户锁定决策引入 **SM4-GCM at-rest 加密** + **envelope 版本化** + **密钥轮换** 三件套，
覆盖范围：`input_configs.password` / `input_configs.stream_password` /
`system_settings[auth.ad.password]` / `system_settings[auth.ad]` JSON。

按 P0 纪律，所有 5 个原子 commit（w1a/w1b/w1c/w2/w3）独立提交，未 squash。

---

## Commits

| Wave | Commit | 说明 |
|------|--------|------|
| W1a | `e6315ce` | feat(18-w1a): SM4-GCM 静态加密 envelope + PKCS7 padding + tamper 检测 |
| W1b | `1dbb3b0` | feat(18-w1b): AuthConfig.CredentialSM4* 字段 + BindEnv + ValidateCredentialSM4Config |
| W1c | `edaa4ae` | feat(18-w1c): CredentialEncryptor service + input_config 列宽 + 集成测试 |
| W2  | `558f723` | feat(18-w2): encrypt-on-write + decrypt-on-read 接入业务层 |
| W3  | `bd84fe2` | feat(18-w3): fail-closed 启动 + 列宽扩展 + 集成测试 + 文档 |

---

## 核心决策（用户锁定 + 实现遵循）

### 算法
- **SM4-GCM** (NIST SP 800-38D)：12B nonce + 16B tag，PKCS#7 补到 16B 边界（gmsm v1.4.1 限制）。
- **envelope 格式**：`SM4:<version>:<base64(nonce_12B | ciphertext | tag_16B)>`
- 解析失败 / 未知 version / tag 校验失败 → 立即报错（**永不静默跳过**）

### 密钥族分离
| 环境变量 | 用途 |
|---|---|
| `SM4_SECRET` | 浏览器传输 SM4-ECB（与前端 sm-crypto） |
| `HLS_TOKEN_SECRET` | HLS URL 签名 |
| `CREDENTIAL_SM4_SECRET` | **at-rest 凭据加密**（Phase 18 新增） |

三组密钥**必须互不相同**。

### 启动期 fail-closed（10 步）
1. LoadConfig + ValidateCredentialSM4Config（main.go）
2. initDatabase（AutoMigrate + ALTER COLUMN 扩 password 列到 TEXT）
3. 构造 CredentialEncryptor
4. MigratePlaintextToGCM（事务内：plaintext/base64-stub → envelope；剥离 auth.ad JSON password；Unscoped 含软删除）
5. 第一次 InvariantScan（必须 0 失败）
6. RotateIfNeeded（previous → current）
7. 第二次 InvariantScan（轮换后必须全 current）
8. initRouter → checkPythonDependencies → initHandlers → registerRoutes → registerServices
9. 启动 HTTP（任何上一步失败 → 不进入）

### 死代码删除
- `internal/services/config_service.go:109-123` 的 `s.encryptPassword` / `s.decryptPassword`
  base64-stub **完全删除**（替换为 CredentialEncryptor 真实 GCM 加密）
- `huawei_configs` legacy 表保留（不在本 phase 范围；future cleanup）

---

## 改动文件清单

### 新增
- `internal/services/credential_encryptor.go` — CredentialEncryptor service
- `internal/services/credential_encryptor_test.go` — 21 个测试函数
- `internal/services/config_service_test.go` — 6 个测试函数
- `cmd/server/phase18_integration_test.go` — 6 个端到端集成测试

### 修改
- `internal/utils/sm4_password.go` — 新增 EncryptGCM/DecryptGCM/ParseCredentialEnvelope/EncodeCredentialEnvelope + PKCS#7 padding
- `internal/utils/sm4_password_test.go` — 9 个新测试函数（23 个总数）
- `internal/config/config.go` — AuthConfig 增 4 个 CredentialSM4* 字段 + BindEnv + ValidateCredentialSM4Config
- `internal/config/config_test.go` — 3 个新测试函数（合法 + 12 个非法路径 + BindEnv）
- `internal/models/input_config.go` — Password / StreamPassword 从 varchar(100) 改 type:text
- `internal/services/input_config_service.go` — 注入 encryptor；CreateConfig/UpdateConfig 加密；GetConfigByID 解密
- `internal/services/input_config_service_test.go` — 重写为 6 个 Phase 18 测试函数
- `internal/services/config_service.go` — 重写：删除 base64-stub；使用 authADForDB DTO 剥离 JSON password；SaveAuthConfig/LoadAuthConfig 走 encryptor
- `internal/handlers/admin_handler.go` — NewAdminHandler 增 encryptor；MigrateInputConfigs INSERT 前加密
- `cmd/server/app.go` — MinimalApp 加 credentialEncryptor 字段；Initialize() 10 步 fail-closed；widenPasswordColumns 在 migrateDatabase 内
- `cmd/server/main.go` — 新增 cfg.ValidateCredentialSM4Config() 启动校验

### 文档
- `.env.example` — 新增 CREDENTIAL_SM4_* 4 个示例 + 注释
- `SECURITY.md` — 新增「凭据静态加密 (Phase 18)」章节（算法 / envelope / 密钥族 / fail-closed / 威胁模型 / 轮换）

---

## 测试覆盖

| 包 | 测试函数 | 状态 |
|---|---|---|
| `internal/utils` | 23 个（含 9 个 Phase 18 新增） | PASS |
| `internal/config` | 13 个（含 3 个 Phase 18 新增） | PASS |
| `internal/services` | 33+ 个（含 33 个 Phase 18 新增） | PASS |
| `cmd/server` | 6 个 Phase 18 集成测试 + 既有 auth/cors 测试 | PASS |

**Phase 18 新增测试函数（总计 51 个）**：
- sm4_password_test.go: TestEncryptGCM_DecryptGCM_RoundTrip / TestEncryptGCM_NonceIsRandom / TestDecryptGCM_WrongKey / TestDecryptGCM_TamperedCiphertext / TestDecryptGCM_TruncatedCiphertext / TestDecryptGCM_InvalidKeyLength / TestParseCredentialEnvelope_Success / TestParseCredentialEnvelope_MissingPrefix / TestParseCredentialEnvelope_MissingVersion / TestParseCredentialEnvelope_EmptyPayload / TestParseCredentialEnvelope_InvalidBase64 / TestEncodeCredentialEnvelope_EmptyVersion / TestEncodeCredentialEnvelope_EmptyPayload / TestCredentialEnvelopeVersion_Constant
- config_test.go: TestValidateCredentialSM4Config_AcceptValid / TestValidateCredentialSM4Config_Reject / TestBindEnvCredentialSM4
- credential_encryptor_test.go: TestNewCredentialEncryptor / TestCredentialEncryptor_EncryptDecrypt / TestMigratePlaintextToGCM_PlaintextRows / TestMigratePlaintextToGCM_Base64Legacy / TestMigratePlaintextToGCM_Idempotent / TestMigratePlaintextToGCM_SoftDeleted / TestMigratePlaintextToGCM_AuthADPassword / TestMigratePlaintextToGCM_AuthADStripPassword / TestInvariantScan_AllEnvelopePass / TestInvariantScan_PlaintextRowFails / TestInvariantScan_UnknownVersionFails / TestInvariantScan_TamperedCiphertextFails / TestInvariantScan_ADJSONPasswordFails / TestRotateIfNeeded_NoPrevious_NoOp / TestRotateIfNeeded_RewritesPreviousVersion / TestRotateIfNeeded_Idempotent
- input_config_service_test.go: TestInputConfigService_CreateConfig_EncryptsPasswords / TestInputConfigService_CreateConfig_NilEncryptor_PassesThrough / TestInputConfigService_CreateConfig_EmptyPasswords_StaysEmpty / TestInputConfigService_UpdateConfig_EncryptsOnlyChanged / TestInputConfigService_GetConfigByID_DecryptionFailureFails / TestInputConfigService_UpdateConfig_StreamPassword_Encryption
- config_service_test.go: TestConfigService_SaveAuthConfig_StripsPasswordFromADJSON / TestConfigService_SaveAuthConfig_PasswordStoredAsEnvelope / TestConfigService_SaveAuthConfig_NilEncryptor_Fails / TestConfigService_LoadAuthConfig_DecryptsPassword / TestConfigService_LoadAuthConfig_NilEncryptor_Fails / TestConfigService_SaveAuthConfig_EmptyPassword_NoEnvelopeWritten
- phase18_integration_test.go: TestPhase18_StartupFlow_PlaintextRowsMigratedToEnvelope / TestPhase18_InvariantScan_FailsOnPlaintextRow / TestPhase18_InvariantScan_FailsOnAuthADPasswordField / TestPhase18_RotateIfNeeded_FromPrevToCurrent / TestPhase18_Initialize_FailClosedWhenInvariantFails / TestPhase18_FullStartup_WithLegacyData

---

## 验证结果

- ✅ `go build ./...` silent（无错误）
- ✅ `go vet ./...` silent
- ✅ `gofmt -l` 全部 touched files clean
- ✅ `go test ./...` 全部 PASS（既有测试 + 51 个新增）
- ✅ 全项目零回归（既有 config / utils / services / handlers / cmd / scheduler / recorder / migrations / models / middleware / storage / huawei 等包全部绿）
- ✅ Wave 4 out-of-scope（DEPLOYMENT.md operator runbook / 多轮 rotation 测试）由独立 agent 处理

---

## 与计划差异 / 设计调整

1. **gmsm Sm4GCM nonce 拷贝**：发现 gmsm v1.4.1 `Sm4GCM` 在 96-bit IV 时通过 `IV = append(IV, 0001)` 就地修改 nonce slice 的 backing array；若直接传入 envelope 的 nonce 切片会污染 ciphertext 段。EncryptGCM / DecryptGCM 内部统一先 copy 再传给 gmsm。已加注释说明。
2. **PKCS#7 padding**：gmsm v1.4.1 `GCMDecrypt` 仅支持块对齐密文（`P = make([]byte, BlockSize*n)`）。任意长度明文需先补到 16B 边界。tag 仍覆盖完整 plaintext（包括 padding 字节），tamper detection 不受影响。
3. **migrations/017_alter_* 未创建**：计划明确指出 `runCustomMigrations()` 永远不会被调用（dormant），所以列宽扩展直接放在 `cmd/server/app.go:widenPasswordColumns()` 内由 Initialize() 主动执行。不创建 dead migration 文件。
4. **EncryptGCM 拷贝 + sanity check**：原计划提到 "constant-time tag 比对"，实现时发现 gmsm Sm4GCM 解密路径第二个返回值 `_T` 就是基于输入 ciphertext 复算的 expected tag，所以我们的 `DecryptGCM` 用它做常量时间比对即可（不需要重新 encrypt）。
5. **SECURITY.md 章节独立成块**：与已有 SECRET 校验 / HLS Token 安全 / TLS 最低版本 三节并列；新增「凭据静态加密 (Phase 18)」放在文件末尾（与 恢复操作 之后）。

---

## 不在本 phase 范围（明确不做）

- `models.FileShare.Password`（独立 phase）
- `models.APIKey.Key` / `models.Session.Token` / `models.UploadedFile.AccessToken` / `models.FileShare.ShareToken`（lookup-by-value）
- User passwords（bcrypt hash）
- YAML / cloud secrets
- Legacy `huawei_configs` 表
- `StreamURL` URL-embedded credentials（独立 phase）
- dormant migration registry（`runCustomMigrations()`、`013_add_ad_fields.go` 等）
- 前端 transport（`frontend/src/utils/sm4.ts` + `utils/sm4_password.go` ECB helpers）
- Wave 4: Operator runbook in DEPLOYMENT.md
- Wave 4: Rotation-specific tests (v1→v2→v3 repeated)
- Wave 4: Per-site/version counts logging
- Wave 4: WAL/vacuum/backup-retirement documentation

---

## 后续 agent 接力点（Wave 4）

Wave 4 agent 接手时建议从以下文件开始：
1. `DEPLOYMENT.md` — 新增「凭据密钥配置与轮换」operator runbook 章节
2. `cmd/server/app.go` — 在 `MigratePlaintextToGCM` / `RotateIfNeeded` 后增加 `logPerSiteVersionCounts(ctx, db)` 调用
3. `internal/services/credential_encryptor.go` — 新增 rotation 端到端测试（v1→v2→v3 重复轮换）
4. `docs/audits/2026-08-XX-phase18-post-audit.md`（可选）— 验证 fail-closed 启动在真实生产数据上的行为