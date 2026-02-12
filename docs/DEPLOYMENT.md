# 部署说明

## 概述

本系统使用 Go 的 `embed` 功能将前端静态文件嵌入到后端二进制文件中，生成单个可执行文件即可运行整个应用。

## 首次运行 - 自动初始化

程序首次运行时会**自动创建**以下内容：

1. **配置文件** `config.yaml` 或 `configs/config.yaml`
   - 包含默认配置，开箱即用
   - 端口：8080
   - 数据库：`./data/record.db`

2. **目录结构**
   ```
   data/
   ├── record.db          # SQLite 数据库
   ├── recordings/        # 录制文件存储
   ├── hls/              # HLS 预览文件
   ├── temp/             # 临时文件
   └── files/            # 文件上传存储
   logs/
   └── server.log         # 应用日志
   ```

3. **默认用户**（数据库自动初始化）
   - 用户名：`admin`
   - 密码：`admin123`
   - 角色：系统管理员

## 构建步骤

### Windows 系统

运行完整的构建脚本：

```batch
cd scripts
build-windows.bat
```

或者分步构建：

```batch
# 1. 构建前端
cd frontend
npm run build
cd ..

# 2. 快速构建后端
scripts\build-backend.bat
```

### Linux/macOS 系统

运行构建脚本：

```bash
cd scripts
chmod +x build.sh
./build.sh
```

## 构建输出

构建成功后，输出目录为 `bin/`：

```
bin/
├── record-v2.exe          # Windows 可执行文件
├── record-v2             # Linux 可执行文件
├── record-v2-mac         # macOS 可执行文件
└── config.yaml           # 配置文件（可选，程序会自动生成）
```

## 部署

### 单机部署

1. 将 `bin/` 目录中的可执行文件复制到目标位置
2. 直接运行程序
3. 访问 `http://localhost:8080` 使用系统

**Windows:**
```batch
record-v2.exe
```

**Linux/macOS:**
```bash
chmod +x record-v2
./record-v2
```

### 服务部署（Windows）

使用 NSSM 将程序注册为 Windows 服务：

```batch
nssm install RecordV2 "C:\path\to\record-v2.exe"
nssm set RecordV2 AppDirectory "C:\path\to"
nssm set RecordV2 DisplayName "视频会议录制系统 V2"
nssm start RecordV2
```

### 服务部署（Linux）

创建 systemd 服务文件 `/etc/systemd/system/record-v2.service`：

```ini
[Unit]
Description=Record V2 Service
After=network.target

[Service]
Type=simple
User=record
Group=record
WorkingDirectory=/opt/record-v2
ExecStart=/opt/record-v2/record-v2
Restart=on-failure
RestartSec=5s

# 硬件资源限制
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```

安装并启动服务：

```bash
# 创建用户
sudo useradd -r -s /bin/false record

# 创建目录
sudo mkdir -p /opt/record-v2
sudo chown record:record /opt/record-v2

# 复制程序
sudo cp record-v2 /opt/record-v2/
sudo chown record:record /opt/record-v2/record-v2

# 安装服务
sudo systemctl daemon-reload
sudo systemctl enable record-v2
sudo systemctl start record-v2

# 查看状态
sudo systemctl status record-v2
```

## 配置

程序首次运行会自动创建默认配置文件。如需修改：

编辑 `config.yaml`：

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  path: "./data/record.db"

storage:
  recordings_path: "./data/recordings"
  hls_path: "./data/hls"
```

**重要配置项：**

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.port` | 服务端口 | 8080 |
| `auth.jwt_secret` | JWT 密钥（生产环境必须修改） | 随机生成 |
| `storage.recordings_path` | 录制文件存储路径 | ./data/recordings |
| `logging.level` | 日志级别 | info |

## 注意事项

1. **前端构建**: 修改前端代码后必须重新运行 `npm run build`
2. **嵌入文件**: 修改前端后，构建脚本会自动将文件复制到 `internal/frontend/dist`
3. **配置文件**: 首次运行自动创建，可根据需要修改
4. **端口占用**: 默认端口 8080 被占用时会自动使用 8081
5. **数据安全**: 生产环境务必修改 `jwt_secret` 和默认管理员密码

## 故障排查

### 程序无法启动

1. 检查日志文件 `logs/server.log`
2. 确认端口 8080 未被占用
3. 检查文件系统写入权限

### 无法访问前端

1. 确认前端已构建（运行 `npm run build`）
2. 查看日志中是否有 "前端静态文件未找到" 警告
3. 尝试直接访问 http://localhost:8080

### 数据库错误

1. 检查 `data/` 目录写入权限
2. 删除 `data/record.db` 重新初始化（注意：会丢失数据）
