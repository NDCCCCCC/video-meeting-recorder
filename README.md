# Record V2

视频切割和会议转录PPT系统

## 功能

- 视频多点分割
- 本地/云端转录（阿里通义听悟）
- PPT自动生成和编辑
- Windows AD域控认证
- USB/流媒体录制支持
- 批量操作

## 多平台支持

后端使用纯 Go 实现（`modernc.org/sqlite`，无需 CGO），支持以下平台交叉编译：

| 平台 | 架构 | 输出 |
|------|------|------|
| Windows | amd64 / arm64 | `record-v2-windows-{amd64,arm64}.exe` |
| Linux | amd64 / arm64 | `record-v2-linux-{amd64,arm64}` |
| macOS | amd64 / arm64 | `record-v2-darwin-{amd64,arm64}` |

快速构建：

```bash
./scripts/build.sh all        # 4 个目标（含前端）
./scripts/build.sh linux/arm64 # 单个目标
```

详见 [BUILD.md](./BUILD.md)。

## 部署

详见 [DEPLOYMENT.md](./DEPLOYMENT.md)。关键点：

- 真实密钥通过**环境变量**注入，不要写进 `config.yaml`
- TLS 证书部署时重新生成
- 前端生产构建需要同步 SM4 密钥

## 安全

详见 [SECURITY.md](./SECURITY.md)。

## Windows AD域控认证

系统支持Windows Active Directory域控认证，允许企业用户使用AD账号登录。

### 配置要求

- Windows Server 2008+ Active Directory
- LDAP端口389或LDAPS端口636
- AD管理员账号

### 快速配置

1. 使用管理员账号登录
2. 导航到"系统设置 > 域控认证"
3. 选择"AD域控认证"模式
4. 配置AD服务器信息并测试连接
5. 保存配置

### 安全注意事项

- ⚠️ **生产环境必须使用LDAPS端口636**，LDAP端口389存在明文传输风险
- AD管理员密码使用环境变量存储

详细配置说明：[Windows AD域控认证管理员指南](./.planning/phases/12-windows-ad/12-DOCS.md)
