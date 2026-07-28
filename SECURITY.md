# SECURITY.md

Record V2 项目安全策略与上传前检查清单。

## ⚠️ 历史密钥事件

### 事件 #2: Release v2.0.0 二进制泄露密钥 (2026-07-24)

**发现**: 远端 release v2.0.0 二进制资产 (`record-v2-windows-amd64.exe` 等 4 个) 通过 Go embed 嵌入了前端构建产物，而前端的 `frontend/.env.production` 在 build 时被 Vite 注入，含有真实 SM4 密钥和内网 IP。

**影响**:
- 任何下载 release 资产的人都能获取 SM4 密钥
- 知道内网 API 地址（10.62.0.123）

**响应**:
1. 重新生成 SM4 密钥（`openssl rand -hex 16`）
2. 重新构建前端 + 4 平台二进制
3. 通过 GitHub API 删除旧 release 资产，上传新资产
4. **旧密钥已废止，所有部署需更新 `config.yaml` 与 `frontend/.env.production`**

**新密钥**: 通过环境变量注入，仓库中**永远不存真实值**。

### 事件 #1: 初始清理 (2026-07-24)

发现以下文件曾被提交进 git 历史，已通过 `git-filter-repo` 全部清除：

| 文件 | 暴露内容 | 风险 |
|------|---------|------|
| `.env` | 真实阿里通义听悟 APP_KEY | 高 |
| `config.yaml` | 真实 SM4 加密密钥 | 高 |
| `frontend/.env.development` | 真实 SM4 密钥 | 高 |
| `frontend/.env.production` | SM4 密钥 + 内网 IP | 高 |
| `certs/server.crt`, `certs/server.key` | TLS 私钥 | 中 |
| `scripts/test_sm4_token.go` 等 | 调试脚本 | 低 |
| `.claude/worktrees/...` | agent 临时工作副本 | 低 |

**重要**：所有上述文件的本地副本**未删除**（仍可用于本地开发），但仓库中的所有历史记录已通过 filter-repo 重写。向远端推送前请按本文下方的"推送前清单"逐项确认。

## 受保护的文件

以下文件类型应通过 `.gitignore` 阻止提交：

```
.env                # 后端环境变量
.env.*.local        # 本地覆盖配置
config.yaml         # 后端配置文件（含密钥）
frontend/.env.development
frontend/.env.production
certs/*.crt *.pem *.key  # TLS 证书与私钥
.claude/worktrees/      # agent 临时工作树
```

`.env.example` 与 `config.yaml.example` 始终被跟踪（仅占位）。

## 推送前清单

任何新增/修改后推送远端前请逐项确认：

- [ ] `git status` 不应包含 `.env` / `config.yaml` / `frontend/.env.*`
- [ ] `git log -p HEAD` 无新增密钥字符串
- [ ] 仅提交 `*.example` 模板和源代码
- [ ] 真实密钥通过环境变量或部署脚本注入，不要写入仓库
- [ ] 新的内网 IP、桶名、域名不应出现在仓库中
- [ ] TLS 私钥只放在部署机上，不进仓库

可用验证命令：

```bash
# 检查工作区是否有敏感文件
git ls-files | grep -iE "(\\.env$|config\\.yaml$|certs/|\\.key$|\\.crt$)"

# 检查历史中是否还有敏感关键字（输出应为空）
git rev-list --all --objects | grep -E "(TYTW_APP_KEY|sm4_secret|<redacted-old-sm4-key>)"

# 检查源代码中是否有硬编码密钥
grep -rIn -E "(VITE_SM4_SECRET|sm4_secret|TYTW_APP_KEY)" --include="*.go" --include="*.ts" \
  --include="*.tsx" --include="*.vue" \
  --exclude-dir=node_modules --exclude-dir=.git --exclude-dir=dist
```

## 部署时的密钥注入

推荐通过环境变量注入真实密钥，而非将 `.env` / `config.yaml` 复制到服务器：

```bash
# 后端：使用环境变量
export SM4_SECRET="$(openssl rand -hex 16)"
export ALIYUN_ACCESS_KEY_ID="..."
export ALIYUN_OSS_BUCKET="..."
export HUAWEI_CONF_SERVER="..."
./record-v2.exe

# 前端：构建时注入
echo "VITE_SM4_SECRET=$SM4_SECRET" > frontend/.env.production
cd frontend && npm run build
```

部署文档：`DEPLOYMENT.md`

## 恢复操作

如果未来需要恢复某个被 filter-repo 删除的文件内容（误删），可从原始备份恢复：

```bash
# 本地保留了过滤前的 reflog 和分支 BACKUP_PRE_CLEANUP（如有）
git reflog | head -20
git show <old-commit-sha>:.env
```

## 报告安全问题

发现新的安全风险，请勿在公开 issue 中暴露细节，联系仓库管理员。

---

最后更新：2026-07-24
