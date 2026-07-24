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

*部署文档版本: 1.1*
*最后更新: 2026-07-24*
