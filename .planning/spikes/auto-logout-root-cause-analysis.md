# Spike: 文件管理页面自动登出根本原因分析

## 目标

彻底理清文件管理页面自动登出问题的根本原因，从架构层面提出解决方案。

## 背景

**症状：**
- 停留在文件管理页面时自动登出
- 多次尝试修复（Promise 缓存、冷却机制）后问题仍存在
- 错误日志显示多个并发 401 请求

**已知信息：**
1. 文件管理页面每 5 秒自动刷新数据
2. 页面加载时发起 20+ 并发请求
3. 后端有 token 重用检测机制（SM4-GCM）
4. Token 有效期：access token 2小时，refresh token 7天

## 研究计划

### 阶段 1: 后端 Token 机制分析
- [ ] 分析 SM4 token 服务实现
- [ ] 理解 token 重用检测逻辑
- [ ] 确认什么情况触发 "token reuse detected"

### 阶段 2: 前端请求模式分析
- [ ] 分析文件管理页面的请求时序
- [ ] 确认并发请求的精确数量和时机
- [ ] 验证自动刷新机制的实际行为

### 阶段 3: 架构层面分析
- [ ] 评估当前 token 刷新架构的合理性
- [ ] 分析前后端交互的边界条件
- [ ] 识别设计层面的问题

### 阶段 4: 根本解决方案
- [ ] 设计架构级修复方案
- [ ] 评估方案的可行性和风险
- [ ] 制定实施计划

## 当前状态

**Status:** researching
**Started:** 2026-04-30
**Focus:** 分析后端 token 重用检测机制

## 发现记录

### 🎯 关键发现 #1: 后端 Token 轮换机制的致命缺陷

**位置：** `internal/auth/sm4_token.go:241-276`

**问题：**
```go
// RefreshAccessToken 的执行流程：
1. ValidateRefreshToken(refreshToken)  // 验证 token
2. 检查 token 是否已被撤销（重放检测）
3. RevokeSession(refreshToken)         // 立即撤销！
4. GenerateTokenPair(&user)            // 生成新 token
```

**致命问题：**
- 在步骤 3，refresh token 被**立即撤销**
- 如果在步骤 1-4 之间，前端发送了另一个请求使用同一个 refresh token
- 步骤 2 的重放检测会被触发
- 结果：**所有用户会话被撤销**

### 🎯 关键发现 #2: 网络并发导致的竞态条件

**场景：**
```
前端 Promise 缓存可以防止多次调用 refreshAccessToken()，
但无法防止网络层面的并发！

时间线：
T1: 请求 A 的 HTTP 请求发送（包含旧的 access token）
T2: 请求 A 返回 401
T3: 请求 A 发起刷新请求（POST /auth/refresh，包含 refresh token）
T4: 请求 B 的 HTTP 请求发送（包含旧的 access token）
T5: 请求 B 返回 401
T6: 请求 B 发起刷新请求（POST /auth/refresh，包含同一个 refresh token）
    ↑ 此时请求 A 的刷新可能还在网络传输中！
T7: 后端收到请求 A 的刷新，验证通过，撤销 refresh token
T8: 后端收到请求 B 的刷新，发现 token 已撤销 -> "token reuse detected"!
T9: 所有会话被撤销
```

**关键洞察：**
即使前端代码层面是串行的（Promise 缓存），**HTTP 请求是并发的**！

当多个请求几乎同时返回 401 时：
- 它们都会尝试发起刷新请求
- 即使有 Promise 缓存，HTTP 请求可能已经在网络中了
- 后端会收到多个使用同一个 refresh token 的请求

### 🎯 关键发现 #3: 根本原因是架构不匹配

**当前架构：**
- 后端：OAuth 2.0 Token Rotation 模式
  - 严格的 token 重用检测
  - 一次性使用 refresh token
  - 重用视为安全威胁

- 前端：SPA 的并发请求模式
  - 页面可能同时发起多个请求
  - 401 响应可能同时返回
  - 难以保证完全的串行化

**矛盾：**
后端的安全机制假设**只有一个刷新请求**，但前端的并发模式可能导致**多个刷新请求**！

## 解决方案评估

### 方案 A: 后端增加宽限期（推荐）

**原理：**
不要在验证通过后立即撤销 refresh token，而是：
1. 生成新的 token 对
2. 在数据库中记录 refresh token 的"已使用"状态
3. 如果在**宽限期（如 5 秒）内**收到同一个 token 的刷新请求：
   - 返回相同的 token 对（幂等性）
   - 或返回错误但不视为重放攻击
4. 宽限期过后才真正撤销 token

**优点：**
- ✅ 从根本上解决问题
- ✅ 保持安全性（宽限期很短）
- ✅ 前端无需修改
- ✅ 符合 RFC 6749 的最佳实践

**缺点：**
- ⚠️ 需要修改后端逻辑
- ⚠️ 需要增加数据库字段记录使用时间

### 方案 B: 前端全局请求队列

**原理：**
实现一个全局的请求队列，确保：
1. 任何时候只有一个 token 刷新请求
2. 其他所有请求等待刷新完成
3. 使用更严格的同步机制

**优点：**
- ✅ 前端可控
- ✅ 不需要后端修改

**缺点：**
- ⚠️ 增加前端复杂度
- ⚠️ 仍无法完全防止网络层的并发
- ⚠️ 可能影响性能

### 方案 C: 短期 Token + 自动刷新

**原理：**
- access token 有效期改为 15 分钟
- 在后台自动刷新，不等到 401
- 减少 401 的发生频率

**优点：**
- ✅ 减少并发 401 的概率
- ✅ 更好的用户体验

**缺点：**
- ⚠️ 不能从根本上解决问题
- ⚠️ 增加服务器负载

## 推荐方案：方案 A

**实施步骤：**

1. **修改 Session 模型**，添加字段：
   - `last_used_at` - 最后使用时间
   - `rotation_count` - 轮换次数

2. **修改 RefreshAccessToken 逻辑**：
   ```go
   func (s *SM4TokenService) RefreshAccessToken(refreshToken string) (*TokenPair, error) {
       claims, err := s.ValidateRefreshToken(refreshToken)
       if err != nil {
           return nil, err
       }

       // 检查是否在宽限期内重复使用
       var session models.Session
       if err := s.db.Where("token = ?", refreshToken).First(&session).Error; err == nil {
           if session.LastUsedAt != nil {
               timeSinceLastUse := time.Since(*session.LastUsedAt)
               if timeSinceLastUse < 5*time.Second {
                   // 在宽限期内，返回幂等响应
                   // 查询最近生成的 token 对
                   var newSession models.Session
                   if err := s.db.Where("user_id = ? AND created_at > ?",
                       claims.UserID,
                       time.Now().Add(-5*time.Second),
                   ).Order("created_at DESC").First(&newSession).Error; err == nil {
                       // 返回已生成的新 token（需要缓存）
                       return s.getCachedTokenPair(newSession.ID)
                   }
               }
           }
       }

       // 正常流程：撤销旧 token，生成新 token
       s.RevokeSession(refreshToken)
       return s.GenerateTokenPair(&user)
   }
   ```

3. **添加 Token 缓存**：
   - 在内存中缓存最近生成的 token 对
   - 宽限期内的重复请求返回缓存的 token
   - 5 秒后自动清除缓存

## 实施计划

### 阶段 1: 数据库迁移
- [ ] 在 `sessions` 表添加 `last_used_at` 字段（`datetime`，可为空）
- [ ] 添加索引：`(user_id, token, is_active)`
- [ ] 创建迁移脚本

### 阶段 2: 后端修改
- [ ] 修改 `models.Session` 添加 `LastUsedAt` 字段
- [ ] 添加内存缓存：`map[uint]*TokenPair`（user_id -> token_pair）
- [ ] 添加缓存过期机制（5 秒 TTL）
- [ ] 修改 `RefreshAccessToken` 实现宽限期逻辑
- [ ] 添加单元测试验证幂等性

### 阶段 3: 测试验证
- [ ] 并发刷新测试（模拟 10 个同时 401）
- [ ] 验证宽限期内返回相同 token
- [ ] 验证宽限期外正常轮换
- [ ] 验证仍能检测真正的重放攻击

### 阶段 4: 部署
- [ ] 灰度发布（先测试环境）
- [ ] 监控日志中的 "token reuse detected" 数量
- [ ] 确认自动登出问题解决

## 风险评估

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| 宽限期被滥用 | 低 | 中 | 限制宽限期为 5 秒，记录异常日志 |
| 内存缓存泄漏 | 低 | 中 | 使用带 TTL 的缓存库（如 go-cache） |
| 数据库性能 | 低 | 低 | 已有索引，查询很快 |
| 真实攻击被放过 | 极低 | 高 | 保留 >5 秒的重放检测，监控日志 |

## 验收标准

- [x] 根本原因已识别（网络层并发）
- [x] 解决方案已设计（宽限期 + 幂等性）
- [ ] 实施计划已完成
- [ ] 测试已通过
- [ ] 生产环境问题已解决

## Status

**Current:** designing_solution
**Next:** 开始实施阶段 1（数据库迁移）
