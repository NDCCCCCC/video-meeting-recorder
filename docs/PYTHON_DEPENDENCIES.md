# Python 依赖管理指南

本文档说明 Record V2 系统中 Python 依赖的管理和配置。

## 概述

系统使用 Python 脚本处理 PPT 相关功能：
- `scripts/create_pptx.py` - 从图片帧生成 PPT
- `scripts/extract_slides.py` - 从 PPT 提取幻灯片图片
- `scripts/merge_slides.py` - 合并多个 PPT 的幻灯片

## Python 依赖

| 包 | 版本 | 用途 |
|---|------|------|
| python-pptx | ^1.0.2 | PowerPoint 文件生成和操作 |
| Pillow | ^10.0.0 | 图片处理 |

## 安装方式

### 方式 1: 使用 uv（推荐）

`uv` 是一个快速的 Python 包管理器。

```bash
# 安装 uv
curl -LsSf https://astral.sh/uv/install.sh | sh

# 或在 Windows 上
powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"

# 同步依赖
uv sync

# 运行脚本
uv run python scripts/create_pptx.py output.pptx image1.jpg image2.jpg
```

### 方式 2: 使用系统 Python

```bash
# 安装依赖
pip install python-pptx Pillow

# 验证安装
python -c "import pptx; print(pptx.__version__)"
```

## 配置

在 `config.yaml` 中配置 Python 依赖管理方式：

```yaml
python:
  prefer_uv: true  # true=使用uv, false=使用系统Python
```

## 部署检查

应用启动时会自动检查 Python 依赖：

```bash
# 检查依赖
go run cmd/server/main.go

# 输出示例
{"level":"info","msg":"Python dependencies verified","python_version":"3.13.2","packages":["pptx","pillow"]}
```

如果依赖缺失，应用会输出错误信息：

```
Error: python not found in PATH
或
Error: python-pptx not installed
```

## 故障排查

### 问题：找不到 python 命令

**解决方案：**
```bash
# Ubuntu/Debian
sudo apt-get install python3 python3-pip

# CentOS/RHEL
sudo yum install python3 python3-pip

# macOS
brew install python@3.13
```

### 问题：python-pptx 导入失败

**解决方案：**
```bash
# 使用 uv
uv sync

# 或使用 pip
pip install python-pptx==1.0.2 Pillow
```

### 问题：uv 命令不存在

**解决方案：**
```bash
# 设置 prefer_uv: false 在 config.yaml 中
# 然后使用系统 Python
pip install -r requirements.txt
```

## CI/CD 集成

### Dockerfile 示例

```dockerfile
FROM python:3.13-slim

# 安装 uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# 复制项目文件
COPY . /app
WORKDIR /app

# 同步依赖
RUN uv sync --frozen

# 运行应用
CMD ["./server"]
```

### GitHub Actions 示例

```yaml
- name: Set up Python
  uses: actions/setup-python@v4
  with:
    python-version: '3.13'

- name: Install uv
  run: curl -LsSf https://astral.sh/uv/install.sh | sh

- name: Install dependencies
  run: uv sync
```

## 开发环境设置

```bash
# 1. 克隆项目
git clone <repo-url>
cd record_V2

# 2. 安装 uv
curl -LsSf https://astral.sh/uv/install.sh | sh

# 3. 同步 Python 依赖
uv sync

# 4. 运行应用
go run cmd/server/main.go
```

## 生产环境部署

### 使用 systemd 服务

```ini
[Service]
Environment="PATH=/usr/local/bin:/usr/bin:/bin"
WorkingDirectory=/opt/record-v2
ExecStartPre=/usr/local/bin/uv sync
ExecStart=/opt/record-v2/server
```

### Docker Compose

```yaml
services:
  app:
    build: .
    volumes:
      - ./data:/app/data
    environment:
      - PYTHON_PREFER_UV=true
```

## 依赖版本锁定

`pyproject.toml` 中定义了依赖版本范围：

```toml
dependencies = [
    "python-pptx>=1.0.2,<2.0.0",
    "Pillow>=10.0.0,<12.0.0",
]
```

`uv.lock` 文件包含精确的版本锁定（自动生成）。

## 更新依赖

```bash
# 更新所有依赖
uv lock --upgrade

# 更新特定包
uv lock --upgrade-package python-pptx

# 重新生成 lockfile
uv sync --reinstall
```

## 安全注意事项

1. **不使用 `pip install` 运行时安装** - 避免在应用运行时动态安装依赖
2. **版本固定** - 生产环境应使用 `uv.lock` 锁定版本
3. **定期更新** - 定期检查并更新依赖以获取安全补丁
4. **审计依赖** - 使用 `uv pip check` 检查依赖冲突

## 相关文件

- `pyproject.toml` - Python 项目定义
- `scripts/check_python_deps.py` - 依赖检查脚本
- `internal/services/python_deps.go` - Go 集成代码
- `internal/services/pptx_generator.go` - PPT 生成服务
