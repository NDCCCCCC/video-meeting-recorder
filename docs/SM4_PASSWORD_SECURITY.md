# SM4 密码加密传输安全配置指南

## 配置要求

### 1. 密钥生成

使用强随机密钥生成器生成 SM4 密钥:

```bash
openssl rand -base64 16
```

### 2. 密钥同步

确保前后端使用相同的 SM4 密钥:
- 后端: `config.yaml` 中的 `auth.sm4_secret`
- 前端: `frontend/.env.production` 中的 `VITE_SM4_SECRET`

### 3. 生产环境检查清单

- [ ] SM4 密钥已从默认值更改为强随机密钥
- [ ] 前端环境变量 `VITE_SM4_SECRET` 已正确配置
- [ ] TLS/HTTPS 已启用（现有安全层）
- [ ] 密钥已妥善保管，未提交到版本控制系统
- [ ] 前端生产构建包含正确的密钥

## 技术细节

### 加密模式

- 算法: SM4 (国密)
- 模式: ECB (适合短字符串如密码)
- 密钥长度: 16 字节（从任意长度密钥通过 SHA256 派生）
- 密文格式: Base64 编码

### 传输流程

1. 用户输入明文密码
2. 前端使用 SM4-ECB 加密密码
3. 通过 HTTPS/TLS 发送加密密码
4. 后端解密后使用 bcrypt 验证
5. 返回 Token

### 向后兼容

- 后端自动检测密码是否加密
- 支持明文密码（不推荐，仅用于过渡期）
- 加密检测依据: Base64 格式和长度特征

## 故障排查

### 问题: 登录失败，提示"密码格式错误"

- 检查前后端 SM4 密钥是否一致
- 检查前端环境变量是否正确加载
- 查看后端日志中的解密错误信息

### 问题: 密码未加密

- 检查前端是否正确调用 `encryptPassword`
- 检查 `VITE_SM4_SECRET` 是否为空
- 查看浏览器 Network 面板的请求内容

### 问题: 前端构建后密钥丢失

- 确保 `VITE_SM4_SECRET` 在构建时可用
- 检查 `.env.production` 文件是否正确配置
- 验证构建输出包含正确的环境变量

## 安全最佳实践

1. **密钥轮换**: 定期更换 SM4 密钥（建议每季度）
2. **密钥存储**: 不要在代码中硬编码密钥
3. **环境隔离**: 开发、测试、生产环境使用不同密钥
4. **访问控制**: 限制配置文件的访问权限
5. **审计日志**: 监控密码解密失败的情况

## 相关文件

- 后端配置: `config.yaml`
- 后端解密: `internal/utils/sm4_password.go`
- 后端集成: `internal/auth/service.go`
- 前端工具: `frontend/src/utils/sm4.ts`
- 前端集成: `frontend/src/api/auth.ts`
- 前端配置: `frontend/.env.production`
- 配置示例: `frontend/.env.example`

## 参考资料

- [国密 SM4 算法规范](http://www.gmbz.org.cn/main/viewfile.html?filename=GM%2FT%200002-2012%20%E5%9B%BD%E5%AF%86%E7%AE%97%E6%B3%95%E2%80%94%E5%88%86%E7%BB%84%E5%AF%86%E7%A0%81%E7%AE%97%E6%B3%95.pdf)
- [tjfoc/gmsm 文档](https://github.com/tjfoc/gmsm)
- [sm-crypto 文档](https://github.com/JuneandGreen/sm-crypto)
