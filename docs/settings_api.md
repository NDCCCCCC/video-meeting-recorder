# 系统设置管理 API 文档

## 概述

系统设置管理模块提供了完整的系统配置管理功能，包括服务器配置、数据库配置、存储配置、日志配置、FFmpeg配置、华为会议系统配置以及用户配置管理。

## 目录结构

```
internal/services/settings/
├── settings_service.go      # 核心设置服务
├── huawei_service.go       # 华为会议系统配置服务
└── user_config_service.go  # 用户配置服务

internal/handlers/
└── settings_handler.go      # 设置 API 处理器
```

## API 端点列表

### 服务器配置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/settings/server` | 获取服务器配置 |
| PUT | `/api/v1/settings/server` | 更新服务器配置 |

**请求示例 (PUT):**
```json
{
  "host": "0.0.0.0",
  "port": 8080,
  "environment": "production",
  "read_timeout": 30,
  "write_timeout": 30
}
```

### 数据库配置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/settings/database` | 获取数据库配置 |
| PUT | `/api/v1/settings/database` | 更新数据库配置 |

**注意:** 数据库配置变更需要重启服务才能生效。

### 存储配置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/settings/storage` | 获取存储配置 |
| PUT | `/api/v1/settings/storage` | 更新存储配置 |

### 日志配置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/settings/logging` | 获取日志配置 |
| PUT | `/api/v1/settings/logging` | 更新日志配置 |

### FFmpeg配置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/settings/ffmpeg` | 获取FFmpeg配置 |
| PUT | `/api/v1/settings/ffmpeg` | 更新FFmpeg配置 |

### 华为会议系统配置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/settings/huawei` | 获取华为配置（密码已隐藏） |
| PUT | `/api/v1/settings/huawei` | 更新华为配置 |
| POST | `/api/v1/settings/huawei/test` | 测试华为会议系统连接 |

**测试连接请求示例:**
```json
{
  "server": "192.168.1.100",
  "port": 80,
  "username": "admin",
  "password": "password",
  "use_https": false
}
```

**测试连接响应示例:**
```json
{
  "code": 0,
  "message": "操作成功",
  "data": {
    "success": true,
    "message": "连接成功",
    "response_time": "50ms",
    "details": {
      "status_code": 200,
      "server": "Huawei-Server"
    }
  }
}
```

### 用户配置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/settings/user/summary` | 获取用户配置摘要 |
| GET | `/api/v1/settings/user/permissions` | 获取权限树（按资源分组） |
| PUT | `/api/v1/settings/user/roles/:id/permissions` | 更新角色权限 |
| GET | `/api/v1/settings/user/me` | 获取当前用户信息 |

### 系统信息

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/settings/system/info` | 获取系统信息 |
| GET | `/api/v1/settings/auth` | 获取认证配置（不包含敏感信息） |
| GET | `/api/v1/settings/notification` | 获取通知配置（不包含敏感信息） |
| GET | `/api/v1/settings/audit` | 获取审计配置信息 |

## 配置持久化

配置变更会自动保存到配置文件（默认为 `./configs/config.yaml`）。

配置保存流程：
1. 验证输入参数
2. 更新内存中的配置
3. 序列化为 YAML 格式
4. 写入临时文件
5. 原子操作重命名（确保数据安全）

## 安全特性

### 敏感信息处理
- 华为配置中的密码在 GET 请求中会自动隐藏（显示为 `******`）
- 认证配置中的 JWT 密钥不会通过 API 返回
- 通知配置中的邮箱密码、短信密钥等敏感信息不会返回

### 数据脱敏
所有配置响应都经过了脱敏处理，确保敏感信息不会通过 API 泄露。

## 审计日志

配置变更操作可以与审计日志系统集成，记录：
- 配置变更前的值
- 配置变更后的值
- 变更的用户信息
- 变更时间和 IP 地址

## 配置验证

### 服务器配置验证
- 端口范围: 1-65535
- 环境类型: development, test, production
- 超时范围: 1-300 秒

### 数据库配置验证
- 驱动类型: sqlite, mysql, postgres
- Journal 模式: WAL, TRUNCATE, PERSIST, MEMORY, OFF
- 同步模式: OFF, NORMAL, FULL, EXTRA

### FFmpeg 配置验证
- 编解码器: h264, h265, hevc, vp8, vp9
- 输出格式: mp4, mkv, avi

## 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 请求参数错误 |
| 1002 | 未授权访问 |
| 1005 | 服务器内部错误 |

## 使用示例

### 1. 获取服务器配置

```bash
curl -X GET http://localhost:8080/api/v1/settings/server \
  -H "Authorization: Bearer <token>"
```

### 2. 更新服务器配置

```bash
curl -X PUT http://localhost:8080/api/v1/settings/server \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "port": 8081,
    "environment": "production"
  }'
```

### 3. 测试华为会议连接

```bash
curl -X POST http://localhost:8080/api/v1/settings/huawei/test \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "server": "192.168.1.100",
    "port": 80,
    "username": "admin",
    "password": "password",
    "use_https": false
  }'
```

## 扩展性

### 添加新的配置项

1. 在 `internal/config/config.go` 中定义配置结构
2. 在 `settings_service.go` 中添加 Get 和 Update 方法
3. 在 `settings_handler.go` 中添加对应的 HTTP 处理器
4. 在 `cmd/server/app.go` 的路由中注册新的端点

### 添加配置验证

在 Update 方法中添加验证逻辑，例如：

```go
if req.Port != nil {
    if *req.Port < 1 || *req.Port > 65535 {
        return errors.New("端口号必须在 1-65535 之间")
    }
}
```

## 依赖

- `github.com/spf13/viper` - 配置文件读取
- `gopkg.in/yaml.v3` - YAML 序列化
- `github.com/gin-gonic/gin` - HTTP 框架
- `go.uber.org/zap` - 日志记录
- `gorm.io/gorm` - 数据库操作

## 注意事项

1. **配置变更时机**: 部分配置（如数据库、网络配置）需要重启服务才能生效
2. **并发安全**: 配置保存使用原子操作，确保并发安全
3. **备份建议**: 重要配置变更前建议备份配置文件
4. **权限控制**: 设置 API 需要管理员权限才能访问
