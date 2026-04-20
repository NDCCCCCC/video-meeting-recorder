# Python 依赖管理实现总结

## 已完成的更改

### 1. 项目配置
- ✅ `pyproject.toml` - Python 项目定义和依赖声明
- ✅ `config.yaml` - 新增 `python.prefer_uv` 配置项

### 2. 服务层
- ✅ `internal/services/python_deps.go` - Python 依赖管理器
- ✅ `internal/services/pptx_generator.go` - 支持 uv 运行 Python 脚本

### 3. 脚本
- ✅ `scripts/check_python_deps.py` - 依赖检查脚本
- ✅ `scripts/python-deps/README.md` - 使用文档

### 4. 应用启动
- ✅ `cmd/server/app.go` - 启动时检查 Python 依赖

### 5. 文档
- ✅ `docs/PYTHON_DEPENDENCIES.md` - 完整依赖管理指南

## 配置选项

在 `config.yaml` 中：

```yaml
python:
  prefer_uv: true  # true=使用uv, false=使用系统Python
```

## 运行时行为

### 启动时检查

应用启动时会自动检查 Python 依赖：

```
{"level":"info","msg":"检查Python依赖..."}
{"level":"info","msg":"Python dependencies verified","python_version":"3.13.2","command":"uv"}
```

如果依赖缺失：

```
{"level":"warn","msg":"Python依赖检查失败，PPT功能可能不可用","error":"..."}
```

应用仍会启动，但 PPT 相关功能不可用。

### 执行 PPT 生成

根据 `prefer_uv` 配置：

**使用 uv:**
```bash
uv run python scripts/create_pptx.py output.pptx img1.jpg img2.jpg
```

**使用系统 Python:**
```bash
python3 scripts/create_pptx.py output.pptx img1.jpg img2.jpg
```

## 部署清单

### 新服务器部署

1. **安装 uv（推荐）**
   ```bash
   curl -LsSf https://astral.sh/uv/install.sh | sh
   ```

2. **配置文件** - 设置 `python.prefer_uv: true`

3. **同步依赖**
   ```bash
   uv sync
   ```

4. **启动服务**
   ```bash
   ./server
   ```

### 现有服务器迁移

1. **安装 uv**
2. **设置 `python.prefer_uv: false`**（保持使用系统 Python）
3. **或运行 `uv sync` 并设置为 `true`**

## 故障排查

| 问题 | 解决方案 |
|------|----------|
| `python not found` | 安装 Python 3.8+ |
| `python-pptx not installed` | 运行 `uv sync` 或 `pip install python-pptx Pillow` |
| `uv not found` | 安装 uv 或设置 `prefer_uv: false` |
| 权限错误 | 检查 scripts 目录执行权限 |

## 相关文件

```
record_V2/
├── pyproject.toml                      # Python 依赖定义
├── config.yaml                         # 新增 python.prefer_uv 配置
├── docs/PYTHON_DEPENDENCIES.md         # 完整文档
├── scripts/
│   ├── check_python_deps.py            # 依赖检查脚本
│   ├── python-deps/
│   │   └── README.md                    # 使用指南
│   ├── create_pptx.py                  # PPT 生成（已存在）
│   ├── extract_slides.py               # 幻灯片提取（已存在）
│   └── merge_slides.py                 # 幻灯片合并（已存在）
├── internal/
│   ├── config/config.go                # 新增 PythonConfig
│   └── services/
│       ├── python_deps.go              # 依赖管理器（新）
│       └── pptx_generator.go           # 支持 uv（更新）
└── cmd/server/app.go                   # 启动时检查（更新）
```
