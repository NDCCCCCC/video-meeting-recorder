# API接口设计

## 一、API概述

### 1.1 设计规范

API遵循RESTful设计规范：

- **资源导向** - URL表示资源，HTTP方法表示操作
- **统一响应** - 使用统一的响应格式
- **版本控制** - URL包含版本号 `/api/v1/`
- **认证授权** - 使用JWT进行认证
- **错误处理** - 统一的错误码和错误信息

### 1.2 统一响应格式

```go
// 成功响应
{
    "code": 0,
    "message": "操作成功",
    "data": { ... }
}

// 错误响应
{
    "code": 10001,
    "message": "会议不存在",
    "data": null
}

// 分页响应
{
    "code": 0,
    "message": "操作成功",
    "data": {
        "items": [ ... ],
        "total": 100,
        "page": 1,
        "page_size": 20,
        "total_pages": 5
    }
}
```

### 1.3 错误码定义

```go
// internal/common/errors.go
const (
    // 成功
    CodeSuccess = 0

    // 通用错误 (1xxx)
    CodeInvalidRequest    = 1001 // 请求参数无效
    CodeUnauthorized      = 1002 // 未授权
    CodeForbidden         = 1003 // 禁止访问
    CodeNotFound          = 1004 // 资源不存在
    CodeInternalError     = 1005 // 内部错误
    CodeServiceUnavailable = 1006 // 服务不可用

    // 用户错误 (2xxx)
    CodeUserNotFound      = 2001 // 用户不存在
    CodeUserExists        = 2002 // 用户已存在
    CodeInvalidPassword   = 2003 // 密码错误
    CodeUserDisabled      = 2004 // 用户已禁用

    // 会议错误 (3xxx)
    CodeConferenceNotFound     = 3001 // 会议不存在
    CodeConferenceExists       = 3002 // 会议已存在
    CodeConferenceInProgress   = 3003 // 会议进行中
    CodeConferenceNotStarted   = 3004 // 会议未开始

    // 任务错误 (4xxx)
    CodeTaskNotFound       = 4001 // 任务不存在
    CodeTaskInvalid        = 4002 // 任务数据无效
    CodeTaskExpired        = 4003 // 任务已过期
    CodeTaskInProgress     = 4004 // 任务执行中
    CodeTaskCannotCancel   = 4005 // 任务无法取消

    // 文件错误 (5xxx)
    CodeFileNotFound       = 5001 // 文件不存在
    CodeFileReadError      = 5002 // 文件读取错误
    CodeFileWriteError     = 5003 // 文件写入错误
    CodeDiskFull           = 5004 // 磁盘空间不足
)
```

## 二、认证接口

### 2.1 登录

```http
POST /api/v1/auth/login HTTP/1.1
Host: localhost:8080
Content-Type: application/json

{
    "username": "admin",
    "password": "admin123"
}
```

**响应**：

```json
{
    "code": 0,
    "message": "登录成功",
    "data": {
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
        "expires_at": "2024-01-16T12:00:00Z",
        "user": {
            "id": 1,
            "username": "admin",
            "email": "admin@example.com",
            "full_name": "系统管理员",
            "role": {
                "id": 1,
                "name": "admin"
            }
        }
    }
}
```

### 2.2 登出

```http
POST /api/v1/auth/logout HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

**响应**：

```json
{
    "code": 0,
    "message": "登出成功",
    "data": null
}
```

### 2.3 刷新令牌

```http
POST /api/v1/auth/refresh HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

## 三、用户管理接口

### 3.1 创建用户

```http
POST /api/v1/users HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
Content-Type: application/json

{
    "username": "newuser",
    "password": "password123",
    "email": "newuser@example.com",
    "full_name": "新用户",
    "role_id": 2
}
```

**响应**：

```json
{
    "code": 0,
    "message": "用户创建成功",
    "data": {
        "id": 2,
        "username": "newuser",
        "email": "newuser@example.com",
        "full_name": "新用户",
        "role_id": 2,
        "is_active": true,
        "created_at": "2024-01-15T12:00:00Z"
    }
}
```

### 3.2 获取用户列表

```http
GET /api/v1/users?page=1&page_size=20 HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

**响应**：

```json
{
    "code": 0,
    "message": "操作成功",
    "data": {
        "items": [
            {
                "id": 1,
                "username": "admin",
                "email": "admin@example.com",
                "full_name": "系统管理员",
                "role_id": 1,
                "is_active": true,
                "last_login_at": "2024-01-15T12:00:00Z",
                "created_at": "2024-01-01T00:00:00Z"
            }
        ],
        "total": 10,
        "page": 1,
        "page_size": 20,
        "total_pages": 1
    }
}
```

### 3.3 获取用户详情

```http
GET /api/v1/users/{user_id} HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

### 3.4 更新用户

```http
PUT /api/v1/users/{user_id} HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
Content-Type: application/json

{
    "email": "updated@example.com",
    "full_name": "更新用户"
}
```

### 3.5 删除用户

```http
DELETE /api/v1/users/{user_id} HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

## 四、会议管理接口

### 4.1 创建会议

```http
POST /api/v1/conferences HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
Content-Type: application/json

{
    "conference_number": "9912345678",
    "title": "周例会",
    "start_time": "2024-01-15T14:00:00Z",
    "end_time": "2024-01-15T15:00:00Z",
    "description": "每周例会",
    "huawei_config_id": 1
}
```

**响应**：

```json
{
    "code": 0,
    "message": "会议创建成功",
    "data": {
        "id": 1,
        "conference_number": "9912345678",
        "title": "周例会",
        "start_time": "2024-01-15T14:00:00Z",
        "end_time": null,
        "status": "not_started",
        "attendees": 0,
        "description": "每周例会",
        "created_at": "2024-01-15T12:00:00Z"
    }
}
```

### 4.2 获取会议列表

```http
GET /api/v1/conferences?status=not_started&page=1&page_size=20 HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

### 4.3 获取会议详情

```http
GET /api/v1/conferences/{conference_id} HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

### 4.4 更新会议

```http
PUT /api/v1/conferences/{conference_id} HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
Content-Type: application/json

{
    "title": "更新后的周例会",
    "description": "更新描述"
}
```

### 4.5 删除会议

```http
DELETE /api/v1/conferences/{conference_id} HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

## 五、录制任务接口

### 5.1 创建录制任务

```http
POST /api/v1/recordings HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
Content-Type: application/json

{
    "name": "周例会录制",
    "description": "每周例会录制",
    "start_time": "2024-01-15T14:00:00Z",
    "end_time": "2024-01-15T15:00:00Z",
    "pre_join_minutes": 5,
    "record_delay_minutes": 0,
    "conference_number": "9912345678",
    "huawei_config_id": 1
}
```

**响应**：

```json
{
    "code": 0,
    "message": "录制任务创建成功",
    "data": {
        "id": 1,
        "name": "周例会录制",
        "description": "每周例会录制",
        "start_time": "2024-01-15T14:00:00Z",
        "end_time": "2024-01-15T15:00:00Z",
        "pre_join_minutes": 5,
        "record_delay_minutes": 0,
        "conference_number": "9912345678",
        "huawei_config_id": 1,
        "status": "pending",
        "created_by": 1,
        "created_at": "2024-01-15T12:00:00Z",
        "trigger_time": "2024-01-15T13:55:00Z"
    }
}
```

### 5.2 获取录制任务列表

```http
GET /api/v1/recordings?status=pending&page=1&page_size=20 HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

### 5.3 获取录制任务详情

```http
GET /api/v1/recordings/{task_id} HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

**响应**：

```json
{
    "code": 0,
    "message": "操作成功",
    "data": {
        "id": 1,
        "name": "周例会录制",
        "status": "recording",
        "start_time": "2024-01-15T14:00:00Z",
        "end_time": "2024-01-15T15:00:00Z",
        "recording_file": "/recordings/task_1/recording_20240115140000.mkv",
        "recording_duration": 600,
        "huawei_config": {
            "id": 1,
            "name": "会议室1配置",
            "server": "10.62.10.3",
            "terminal_number": "9912345678"
        },
        "creator": {
            "id": 1,
            "username": "admin",
            "full_name": "系统管理员"
        },
        "conference_record": {
            "id": 1,
            "conference_number": "9912345678",
            "title": "周例会"
        }
    }
}
```

### 5.4 更新录制任务

```http
PUT /api/v1/recordings/{task_id} HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
Content-Type: application/json

{
    "name": "更新后的录制任务",
    "start_time": "2024-01-15T15:00:00Z",
    "end_time": "2024-01-15T16:00:00Z"
}
```

### 5.5 删除录制任务

```http
DELETE /api/v1/recordings/{task_id} HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

### 5.6 启动录制任务

```http
POST /api/v1/recordings/{task_id}/start HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

### 5.7 停止录制任务

```http
POST /api/v1/recordings/{task_id}/stop HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

### 5.8 取消录制任务

```http
POST /api/v1/recordings/{task_id}/cancel HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

### 5.9 重试录制任务

```http
POST /api/v1/recordings/{task_id}/retry HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

### 5.10 获取任务日志

```http
GET /api/v1/recordings/{task_id}/logs?page=1&page_size=50 HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

**响应**：

```json
{
    "code": 0,
    "message": "操作成功",
    "data": {
        "items": [
            {
                "id": 1,
                "task_id": 1,
                "log_type": "task_started",
                "details": {
                    "message": "任务开始执行",
                    "timestamp": "2024-01-15T13:55:00Z"
                },
                "created_at": "2024-01-15T13:55:00Z"
            },
            {
                "id": 2,
                "task_id": 1,
                "log_type": "conference_join",
                "details": {
                    "message": "连接华为会议成功",
                    "timestamp": "2024-01-15T13:55:05Z"
                },
                "created_at": "2024-01-15T13:55:05Z"
            }
        ],
        "total": 100,
        "page": 1,
        "page_size": 50,
        "total_pages": 2
    }
}
```

## 六、文件管理接口

### 6.1 获取文件列表

```http
GET /api/v1/files?conference_id=1&page=1&page_size=20 HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

**响应**：

```json
{
    "code": 0,
    "message": "操作成功",
    "data": {
        "items": [
            {
                "id": 1,
                "file_name": "recording_20240115140000.mp4",
                "file_path": "/recordings/task_1/recording_20240115140000.mp4",
                "file_size": 524288000,
                "duration": 3600,
                "format": "mp4",
                "resolution": "1920x1080",
                "bitrate": 5000,
                "status": "ready",
                "created_at": "2024-01-15T14:00:00Z"
            }
        ],
        "total": 5,
        "page": 1,
        "page_size": 20,
        "total_pages": 1
    }
}
```

### 6.2 下载文件

```http
GET /api/v1/files/{file_id}/download HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

### 6.3 删除文件

```http
DELETE /api/v1/files/{file_id} HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

## 七、系统管理接口

### 7.1 获取系统统计

```http
GET /api/v1/system/stats HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

**响应**：

```json
{
    "code": 0,
    "message": "操作成功",
    "data": {
        "tasks": {
            "total": 100,
            "pending": 10,
            "connecting": 2,
            "recording": 5,
            "completed": 80,
            "failed": 2,
            "cancelled": 1
        },
        "conferences": {
            "total": 150,
            "not_started": 20,
            "in_progress": 5,
            "completed": 125
        },
        "files": {
            "total": 200,
            "total_size": 107374182400,
            "total_size_gb": 100
        },
        "system": {
            "uptime": 86400,
            "version": "2.0.0",
            "go_version": "go1.19"
        }
    }
}
```

### 7.2 获取系统配置

```http
GET /api/v1/system/config HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
```

### 7.3 更新系统配置

```http
PUT /api/v1/system/config HTTP/1.1
Host: localhost:8080
Authorization: Bearer {token}
Content-Type: application/json

{
    "huawei": {
        "conference_server": "10.62.10.3"
    },
    "storage": {
        "recordings_dir": "/recordings"
    }
}
```

## 八、WebSocket接口

### 8.1 连接WebSocket

```javascript
// 前端连接
const ws = new WebSocket('ws://localhost:8080/ws/events?token={jwt_token}');

// 监听消息
ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('收到消息:', data);
};

// 发送消息
ws.send(JSON.stringify({
    action: 'subscribe',
    channels: ['tasks', 'conferences']
}));
```

### 8.2 事件推送

**任务状态更新**：

```json
{
    "channel": "tasks",
    "event": "status_updated",
    "data": {
        "task_id": 1,
        "status": "recording",
        "message": "任务开始录制",
        "timestamp": "2024-01-15T14:00:00Z"
    }
}
```

**会议状态更新**：

```json
{
    "channel": "conferences",
    "event": "status_updated",
    "data": {
        "conference_id": 1,
        "status": "in_progress",
        "timestamp": "2024-01-15T14:00:00Z"
    }
}
```

## 九、Handler实现

### 9.1 BaseHandler

```go
// internal/handlers/base.go
type BaseHandler struct {
    Logger *zap.Logger
    Config *config.Config
}

// SuccessResponse 成功响应
func (h *BaseHandler) SuccessResponse(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, gin.H{
        "code":    0,
        "message": "操作成功",
        "data":    data,
    })
}

// ErrorResponse 错误响应
func (h *BaseHandler) ErrorResponse(c *gin.Context, code int, message string) {
    c.JSON(http.StatusOK, gin.H{
        "code":    code,
        "message": message,
        "data":    nil,
    })
}

// GetUserID 获取用户ID
func (h *BaseHandler) GetUserID(c *gin.Context) uint {
    if userID, exists := c.Get("user_id"); exists {
        return userID.(uint)
    }
    return 0
}

// GetUserID 获取用户信息
func (h *BaseHandler) GetUser(c *gin.Context) *models.User {
    if user, exists := c.Get("user"); exists {
        return user.(*models.User)
    }
    return nil
}
```

### 9.2 RecordingHandler

```go
// internal/handlers/recording_handler.go
type RecordingHandler struct {
    BaseHandler
    taskService *services.VideoRecordingTaskService
}

// CreateTask 创建录制任务
func (h *RecordingHandler) CreateTask(c *gin.Context) {
    var req services.CreateTaskRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.ErrorResponse(c, common.CodeInvalidRequest, "请求参数无效: "+err.Error())
        return
    }

    userID := h.GetUserID(c)

    task, err := h.taskService.CreateTask(&req, userID)
    if err != nil {
        h.ErrorResponse(c, common.CodeInternalError, err.Error())
        return
    }

    h.SuccessResponse(c, task)
}

// GetTask 获取录制任务
func (h *RecordingHandler) GetTask(c *gin.Context) {
    taskID, err := strconv.ParseUint(c.Param("task_id"), 10, 32)
    if err != nil {
        h.ErrorResponse(c, common.CodeInvalidRequest, "无效的任务ID")
        return
    }

    task, err := h.taskService.GetTask(uint(taskID))
    if err != nil {
        h.ErrorResponse(c, common.CodeTaskNotFound, "任务不存在")
        return
    }

    h.SuccessResponse(c, task)
}

// ListTasks 获取任务列表
func (h *RecordingHandler) ListTasks(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
    status := c.Query("status")

    tasks, total, err := h.taskService.ListTasks(page, pageSize, status)
    if err != nil {
        h.ErrorResponse(c, common.CodeInternalError, err.Error())
        return
    }

    h.SuccessResponse(c, gin.H{
        "items":       tasks,
        "total":       total,
        "page":        page,
        "page_size":   pageSize,
        "total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
    })
}
```

## 十、相关文档

- [01-系统架构总览.md](./01-系统架构总览.md)
- [08-服务层设计.md](./08-服务层设计.md)
- [10-配置管理详解.md](./10-配置管理详解.md)
