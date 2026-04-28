---
spike: 002
name: ldaps-security
type: standard
validates: "Given AD服务器配置，当使用LDAPS/StartTLS连接时，则密码传输经过加密且安全"
verdict: VALIDATED
related: [001]
tags: [ldap, ldaps, security, tls, encryption]
---

# Spike 002: ldaps-security

## What This Validates

**Given:** Windows Active Directory服务器配置了LDAP服务
**When:** 客户端使用LDAPS或StartTLS连接并认证用户
**Then:**
- 密码传输经过TLS 1.2+加密
- 数据完整性受到LDAP Signing保护
- 防止中间人攻击和回放攻击
- 支持Channel Binding增强安全性

## Research

### LDAPS vs StartTLS 对比

| 特性 | LDAPS (端口636) | StartTLS (端口389) |
|------|-----------------|-------------------|
| **连接方式** | 立即建立TLS加密 | 先明文握手，后升级TLS |
| **端口** | 636 | 389 |
| **安全性** | ✅ 更高 - 连接即加密 | ⚠️ 较低 - 初始握手可能被拦截 |
| **兼容性** | 所有现代客户端 | 需要客户端支持StartTLS命令 |
| **推荐程度** | ✅ 强烈推荐 | ⚠️ 仅在LDAPS不可用时使用 |

### Microsoft官方安全建议 (2025)

根据[Microsoft Learn](https://learn.microsoft.com/en-us/windows-server/identity/ad-ds/ldap-signing)最新文档：

#### Windows Server 2025 默认安全行为
- ✅ **LDAP Signing**: 新部署默认**强制要求**
- ✅ **Channel Binding**: 默认设置为"When supported"
- ✅ **Client Encryption**: 客户端优先加密连接
- ✅ **Channel Binding Auditing**: 默认启用审计

#### Windows Server 2019 及更早版本
- ⚠️ **LDAP Signing**: 默认可选
- ❌ **Channel Binding**: 默认设置为"Never"
- ⚠️ **Client行为**: 优先加密但可降级到明文

### 安全威胁与防护

| 威胁 | 描述 | LDAP Signing | Channel Binding | LDAPS/StartTLS |
|------|------|--------------|-----------------|----------------|
| **Replay Attacks** | 截获认证票据并重放 | ✅ 防护 | ✅ 防护 | ✅ 防护 |
| **MITM攻击** | 中间人修改数据包 | ✅ 防护 | ✅ 防护 | ✅ 防护 |
| **Session Hijacking** | 劫持加密会话 | ⚠️ 部分防护 | ✅ 完全防护 | ⚠️ 部分防护 |
| **密码嗅探** | 网络窃听密码 | ❌ 无法单独防护 | ❌ 无法单独防护 | ✅ 完全防护 |

### SASL认证机制安全性

| SASL机制 | 安全性 | 说明 |
|----------|--------|------|
| **Kerberos** | ✅ 最高 | 基于票据的认证， mutual authentication |
| **NTLM** | ⚠️ 中等 | 较旧的协议，存在已知弱点 |
| **Digest** | ⚠️ 中等 | 比明文好，但不如Kerberos |
| **Simple Bind** | ❌ 最低 | 明文传输密码，必须配合TLS |

### TLS配置最佳实践

```go
tlsConfig := &tls.Config{
    ServerName:         "ad.example.com",
    InsecureSkipVerify: false, // 生产环境必须false
    MinVersion:         tls.VersionTLS12, // 最低TLS 1.2
    CipherSuites: []uint16{
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_AES_128_GCM_SHA256,    // TLS 1.3
        tls.TLS_AES_256_GCM_SHA384,    // TLS 1.3
    },
}
```

### 密码传输安全保障清单

| 检查项 | 生产环境 | 内部隔离网络 | 验证方法 |
|--------|----------|-------------|----------|
| ✅ 前端SM4加密 | **必须** | **必须** | 检查前端加密逻辑 |
| ✅ HTTPS传输 | **必须** | **必须** | 检查API协议 |
| ✅ LDAPS优先 | 端口636 | 可选 | 检查连接端口 |
| ⚠️ LDAP(389) | **禁止** | 可用(需确认) | 检查配置和环境 |
| ✅ TLS 1.2+ | **必须** | 如启用TLS | 检查TLS版本 |
| ✅ 证书验证 | **必须** | 可选 | 检查InsecureSkipVerify |
| ✅ LDAP Signing | 推荐 | 推荐 | 检查AD策略 |
| ✅ Channel Binding | 推荐 | 可选 | 检查事件日志 |

## How to Run

此Spike为纯研究型，无需运行代码。关键发现已记录在Results部分。

如需验证TLS配置，可参考Spike 001中的代码，确保：
1. 使用`UseTLS: true`连接LDAPS端口636
2. 配置正确的TLS 1.2+最低版本
3. 在生产环境配置企业CA证书

## What to Expect

### 安全连接建立过程

```
1. 客户端 → AD: TCP SYN (端口636)
2. 客户端 ← AD: TCP SYN/ACK
3. 客户端 → AD: TLS ClientHello (TLS 1.3)
4. 客户端 ← AD: TLS ServerHello + 证书
5. 客户端 → AD: TLS Finished (密钥协商完成)
6. 客户端 → AD: LDAP Bind (密码在加密隧道中传输)
7. 客户端 ← AD: Bind Success (认证通过)
```

### Windows事件日志验证

启用LDAP Signing和Channel Binding后，可监控以下事件：

| Event ID | 名称 | 含义 |
|----------|------|------|
| 2889 | Signed LDAP communications | ✅ 成功的签名通信 |
| 2887 | Unsigned LDAP binds detected | ⚠️ 检测到未签名绑定 |
| 3041 | Channel binding successful | ✅ 通道绑定成功 |
| 3039 | Channel binding not supported | ⚠️ 客户端不支持CBT |

## Investigation Trail

### 研究过程 (2025-04-28)

**第1轮 - 文档研究:**
- 查阅Microsoft Learn官方文档
- 对比LDAPS和StartTLS安全性差异
- 了解Windows Server 2025安全默认值

**关键发现:**
1. LDAPS比StartTLS更安全，因为连接建立即加密
2. Windows Server 2025默认强制LDAP Signing
3. Channel Binding提供额外的会话保护层
4. TLS 1.3是Windows Server 2025的标准配置

**第2轮 - 安全威胁分析:**
- 分析Replay Attacks、MITM、Session Hijacking等威胁
- 确认每种防护措施的有效性
- **更新:** 内部隔离网络可使用LDAP(389)，但需前端SM4加密保护

**关键发现:**
1. LDAPS/StartTLS是密码安全的基础
2. LDAP Signing保护数据完整性
3. Channel Binding防止会话劫持
4. **前端SM4加密**为内部LDAP提供了额外保护层

## Results

**Verdict:** ✅ VALIDATED

**关键发现:**
1. ✅ LDAPS(端口636)是安全的，密码通过TLS 1.2+加密传输
2. ✅ StartTLS(端口389)也可用，但安全性略低于LDAPS
3. ✅ Windows Server 2025默认强制LDAP Signing，增强数据完整性
4. ✅ Channel Binding提供会话级别的防护
5. ✅ **前端SM4加密**为所有模式提供基础保护
6. ⚠️ 内部隔离网络可使用LDAP(389)，但需明确风险
7. ⚠️ 生产环境必须配置企业CA证书验证

**安全性评估:**
| 方案 | 密码传输安全 | 数据完整性 | 会话保护 | 适用场景 | 综合评分 |
|------|-------------|-----------|----------|----------|----------|
| LDAPS + Signing + CBT | ✅✅✅ | ✅✅✅ | ✅✅✅ | 生产环境 | **最高** |
| LDAPS + Signing | ✅✅✅ | ✅✅✅ | ⚠️⚠️ | 生产环境 | 高 |
| LDAP(389) + SM4前端 | ✅✅ | ⚠️ | ⚠️ | 内部网络 | 中 |
| StartTLS + Signing | ✅✅✅ | ✅✅✅ | ⚠️⚠️ | 过渡方案 | 中高 |
| LDAPS only | ✅✅✅ | ❌ | ❌ | 基础场景 | 中 |
| LDAP(389)明文 | ❌ | ❌ | ❌ | **禁止** | **不安全** |

**注:** SM4前端加密是指前端使用SM4加密密码，后端解密后再进行LDAP认证。这为内部网络提供了基础保护。

**实施建议:**

**生产环境 (公网/非隔离网络):**
1. ✅ **必须使用LDAPS端口636**
2. ✅ **强制TLS 1.2或更高版本**
3. ✅ **在AD域控上启用LDAP Signing要求**
4. ✅ **启用Channel Binding (Windows Server 2025默认)**
5. ✅ **配置企业CA证书，禁用InsecureSkipVerify**
6. ✅ **前端必须使用SM4加密密码**

**内部隔离网络:**
1. ✅ **优先使用LDAPS端口636** (如果服务器支持)
2. ⚠️ **可使用LDAP端口389** (如果LDAPS不可用)
3. ✅ **前端必须使用SM4加密密码** (核心保护)
4. ⚠️ **配置时需显示安全警告并确认**
5. ✅ **建议在网络层面隔离LDAP流量**

**通用要求:**
1. ✅ **前端SM4加密是所有场景的基础保护**
2. ✅ **HTTPS传输API请求**
3. ✅ **后端解密后再进行LDAP认证**
4. ✅ **所有认证失败记录审计日志**

**对后续Spike的影响:**
- Spike 003 (认证切换架构) 支持端口389和636配置
- Spike 005 (配置验证) 需要验证不同端口的连通性

**配置示例 (基于Spike 001代码):**
```go
config := &ADConfig{
    Server:   "ad.example.com:636",  // LDAPS端口
    BindDN:   "cn=admin,dc=example,dc=com",
    Password: "secure_password",
    BaseDN:   "dc=example,dc=com",
    UseTLS:   true,                   // 必须启用
}

// TLS配置
tlsConfig := &tls.Config{
    ServerName:         "ad.example.com",
    InsecureSkipVerify: false,        // 生产环境必须false
    MinVersion:         tls.VersionTLS12,
}
```

**注意事项:**
1. 测试环境可临时设置InsecureSkipVerify=true，但生产环境禁止
2. Windows Server 2019及更早版本需要手动启用LDAP Signing
3. 升级到Windows Server 2025时注意兼容性测试
