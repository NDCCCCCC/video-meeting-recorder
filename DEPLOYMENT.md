# 生产环境部署配置

## 密钥配置

### 1. 生成 SM4 密钥

使用强随机密钥生成器生成 SM4 密钥：

```bash
openssl rand -base64 16
```

或者使用项目提供的脚本：

```bash
go run scripts/gen_sm4_key.go
```

### 2. 设置后端环境变量

导出 SM4 密钥到环境变量：

```bash
export SM4_SECRET="<生成的密钥>"
```

### 3. 设置前端环境变量

将相同的密钥配置到前端环境变量：

```bash
echo "VITE_SM4_SECRET=<相同的密钥>" > frontend/.env.production
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
  -subj "/C=CN/ST=State/L=City/O=Organization/CN=localhost"
```

---

## 数据库初始化

首次启动时，后端会自动创建 SQLite 数据库文件 (`./data/record.db`)。

---

## 环境变量清单

| 变量名 | 用途 | 必需 | 默认值 |
|--------|------|------|--------|
| `SM4_SECRET` | SM4 加密密钥 | 是 | (无默认，必须设置) |
| `VITE_API_URL` | 前端 API 地址 | 否 | (相对路径) |
| `VITE_SM4_SECRET` | 前端 SM4 密钥 | 是 | (必须与后端一致) |

---

## 启动命令

### 后端

```bash
export SM4_SECRET="<your-sm4-secret>"
go run cmd/server/main.go
```

### 前端

```bash
cd frontend
echo "VITE_SM4_SECRET=<your-sm4-secret>" > .env.production
echo "VITE_API_URL=https://your-server:8443" >> .env.production
npm run build
npm run preview
```

---

## 安全检查清单

部署前请确认：

- [ ] SM4 密钥已从默认值更改为强随机密钥
- [ ] 前端环境变量 `VITE_SM4_SECRET` 已正确配置
- [ ] TLS/HTTPS 已启用
- [ ] 密钥已妥善保管，未提交到版本控制系统
- [ ] 前端生产构建包含正确的密钥
- [ ] 防火墙规则正确配置

---

*部署文档版本: 1.0*
*最后更新: 2026-04-24*
