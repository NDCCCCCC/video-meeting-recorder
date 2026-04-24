# Phase 01: SM4 密码加密 - Context

**Gathered:** 2026-04-24
**Status:** Ready for planning

---

## Phase Boundary

实现应用层的密码国密 SM4 加密传输功能，在现有 TLS 传输层加密基础上增加额外安全保护。

---

## Implementation Decisions

### 技术栈
- **后端**: Go 1.24 + Gin，已有 SM4-GCM 实现 (`internal/auth/sm4_token.go`)
- **前端**: React 19 + TypeScript + Ant Design 6
- **加密库**: `github.com/tjfoc/gmsm/sm4` (Go 端已使用)

### 核心决策

#### 1. 前端 SM4 加密库选择
- **必须**: 支持浏览器环境的 SM4 加密库
- **候选**: `crypto-sm` (国密 JS 库) 或 `sm-crypto` 
- **要求**: 与后端 `tjfoc/gmsm` 兼容的 SM4-ECB 或 SM4-CBC 模式

#### 2. 加密范围
- **仅加密密码字段**: 用户登录时的 `password` 参数
- **不加密**: username (用户名不需要加密传输)
- **不改变**: 后端密码验证逻辑 (bcrypt 对比)

#### 3. 传输格式
```
# 当前 (明文)
POST /api/v1/auth/login
{ "username": "admin", "password": "admin123" }

# 目标 (加密)
POST /api/v1/auth/login
{ "username": "admin", "password": "SM4_ENCRYPTED_BASE64_STRING" }
```

#### 4. 后端解密流程
1. 接收登录请求
2. 使用 SM4 密钥解密 `password` 字段
3. 调用现有的 `CheckPassword()` 验证
4. 返回 Token

#### 5. 密钥管理
- **使用现有**: `config.yaml` 中的 `auth.sm4_secret`
- **派生方式**: SHA256(secret)[:16] (与 Token 服务相同)
- **前端获取**: 通过环境变量或 API 端点获取加密密钥

#### 6. 兼容性保证
- **向后兼容**: 登录接口同时支持明文和加密密码
- **检测机制**: 通过密码格式判断 (Base64 长度) 或添加版本标识
- **迁移策略**: 逐步过渡，最终要求加密传输

### Claude's Discretion

- 前端加密库的具体选择和安装方式
- 密钥分发的安全机制 (API 端点 vs 环境变量)
- 错误处理和用户提示文案
- 单元测试覆盖范围

---

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 现有 SM4 实现
- `internal/auth/sm4_token.go` — SM4-GCM Token 服务，密钥派生函数 `deriveSM4Key()`
- `scripts/gen_sm4_key.go` — SM4 密钥生成示例

### 认证流程
- `internal/auth/service.go` — 登录服务 `Login()` 方法
- `internal/handlers/auth_handler.go` — 登录处理器
- `internal/models/user.go` — 密码验证 `CheckPassword()`

### 前端登录
- `frontend/src/pages/auth/Login.tsx` — 登录页面
- `frontend/src/api/auth.ts` — 登录 API 调用
- `frontend/src/types/auth.ts` — 类型定义

### 配置
- `config.yaml` — SM4 密钥配置 `auth.sm4_secret`
- `frontend/.env.production` — 生产环境配置

---

## Specific Ideas

### 加密模式
- 推荐 SM4-ECB (简单，适合短字符串如密码)
- 或 SM4-CBC (需要 IV，更安全但更复杂)

### 前端集成点
```typescript
// frontend/src/utils/sm4.ts (新文件)
export function encryptPassword(password: string, key: string): string
export function decryptPassword(encrypted: string, key: string): string
```

### 后端修改点
```go
// internal/auth/service.go
func (s *Service) decryptPasswordIfEncrypted(password string) (string, error)
```

---

## Deferred Ideas

- 修改密码时的加密传输 (后续阶段)
- 其他敏感字段的加密传输
- 密钥轮换机制

---

*Phase: 01-sm4*
*Context gathered: 2026-04-24*
