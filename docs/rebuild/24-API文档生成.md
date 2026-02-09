# API文档生成 (Swagger/OpenAPI)

## 概述

本文档描述了如何使用 Swagger/OpenAPI 为项目生成和维护API文档。系统使用 `swag` 工具自动生成 OpenAPI 3.0 规范的文档。

## 核心概念

### OpenAPI 规范

OpenAPI 规范（原 Swagger 规范）是一种用于描述 REST API 的标准格式。它定义了：

- API 端点路径和 HTTP 方法
- 请求参数（路径、查询、头部、请求体）
- 响应格式和状态码
- 认证方式
- 数据模型（Schema）

### Swag 工具

`swag` 是 Go 语言流行的 Swagger 文档生成工具，通过代码注释自动生成文档。

## 安装配置

### 1. 安装 Swag

```bash
# 安装 swag 命令行工具
go install github.com/swaggo/swag/cmd/swag@latest

# 验证安装
swag --version
# 输出: swag version v1.16.1
```

### 2. 添加依赖

```bash
# 进入前端目录（如果需要 Swagger UI）
cd frontend
npm install swagger-ui swagger-ui-react

# 或者在 Go 中使用 gin-swagger 中间件
go get -u github.com/swaggo/gin-swagger
go get -u github.com/swaggo/files
```

### 3. 项目结构

```
record_go/
├── cmd/
│   └── server/
│       └── main.go          # Swagger 总体注释
├── docs/                    # 自动生成的文档
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── internal/
│   └── handlers/            # Handler 注释
│       ├── auth_handler.go
│       ├── user_handler.go
│       └── conference_handler.go
└── pkg/
    └── models/              # 模型注释
        ├── user.go
        └── conference.go
```

## 基础配置

### 主应用注释

**文件位置**: `cmd/server/main.go`

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    _ "record_go/docs" // 导入自动生成的 docs

    // swagger 相关导入
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           视频会议录制系统 API
// @version         1.0
// @description     这是一个用于管理和录制视频会议的系统 API 文档。
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.example.com/support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
    // ... 应用初始化代码

    // 创建 Gin 路由
    r := gin.Default()

    // Swagger 文档路由
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    // API 路由
    setupRoutes(r, app)

    // 启动服务器
    // ...
}
```

## API 注释规范

### 1. 通用注释

```go
// @Summary      简短描述（必需）
// @Description  详细描述（可选）
// @Tags         分组标签（必需）
// @Accept       json
// @Produce      json
// @Param        参数名 参数位置 参数类型 是否必需 "描述"
// @Success      200 {object} ResponseModel "成功响应"
// @Failure      400 {object} ErrorResponse "错误响应"
// @Router       /path [method]
```

### 2. 参数类型说明

| 参数位置 | 说明 | 示例 |
|---------|------|------|
| `query` | URL 查询参数 | `?page=1&size=10` |
| `path` | URL 路径参数 | `/users/:id` |
| `header` | HTTP 头部 | `Authorization: Bearer xxx` |
| `body` | 请求体 | JSON/XML |

### 3. Handler 注释示例

#### 用户列表 API

**文件位置**: `internal/handlers/user_handler.go`

```go
package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

// ListUsers 获取用户列表
// @Summary      获取用户列表
// @Description  分页获取系统中的所有用户，支持关键词搜索和状态过滤
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        page   query    int    false "页码"    minimum(1)    default(1)
// @Param        size   query    int    false "每页数量" minimum(1) maximum(100) default(20)
// @Param        keyword query   string false "搜索关键词（用户名或邮箱）"
// @Param        status query   string false "用户状态" Enums(active, inactive, locked)
// @Param        order  query    string false "排序字段" Enums(id, username, created_at)
// @Param        sort   query    string false "排序方向" Enums(asc, desc) default(asc)
// @Success      200 {object} UserListResponse "成功响应"
// @Failure      400 {object} ErrorResponse "参数错误"
// @Failure      401 {object} ErrorResponse "未授权"
// @Failure      500 {object} ErrorResponse "服务器错误"
// @Router       /users [get]
// @Security     BearerAuth
func (h *UserHandler) ListUsers(c *gin.Context) {
    // 实现代码...
}

// GetUser 获取单个用户
// @Summary      获取用户详情
// @Description  根据 ID 获取用户详细信息
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "用户ID"
// @Success      200  {object}  UserResponse "成功响应"
// @Failure      400  {object}  ErrorResponse "参数错误"
// @Failure      401  {object}  ErrorResponse "未授权"
// @Failure      404  {object}  ErrorResponse "用户不存在"
// @Failure      500  {object}  ErrorResponse "服务器错误"
// @Router       /users/{id} [get]
// @Security     BearerAuth
func (h *UserHandler) GetUser(c *gin.Context) {
    // 实现代码...
}

// CreateUser 创建用户
// @Summary      创建新用户
// @Description  创建一个新的系统用户
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        request body CreateUserRequest true "创建用户请求"
// @Success      201 {object} UserResponse "创建成功"
// @Failure      400 {object} ErrorResponse "参数错误"
// @Failure      401 {object} ErrorResponse "未授权"
// @Failure      403 {object} ErrorResponse "权限不足"
// @Failure      409 {object} ErrorResponse "用户已存在"
// @Failure      500 {object} ErrorResponse "服务器错误"
// @Router       /users [post]
// @Security     BearerAuth
func (h *UserHandler) CreateUser(c *gin.Context) {
    // 实现代码...
}

// UpdateUser 更新用户
// @Summary      更新用户信息
// @Description  更新指定用户的信息
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        id      path     int               true "用户ID"
// @Param        request body     UpdateUserRequest true "更新用户请求"
// @Success      200     {object} UserResponse      "更新成功"
// @Failure      400     {object} ErrorResponse     "参数错误"
// @Failure      401     {object} ErrorResponse     "未授权"
// @Failure      403     {object} ErrorResponse     "权限不足"
// @Failure      404     {object} ErrorResponse     "用户不存在"
// @Failure      500     {object} ErrorResponse     "服务器错误"
// @Router       /users/{id} [put]
// @Security     BearerAuth
func (h *UserHandler) UpdateUser(c *gin.Context) {
    // 实现代码...
}

// DeleteUser 删除用户
// @Summary      删除用户
// @Description  删除指定的用户（软删除）
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "用户ID"
// @Success      204  "删除成功"
// @Failure      400  {object}  ErrorResponse "参数错误"
// @Failure      401  {object}  ErrorResponse "未授权"
// @Failure      403  {object}  ErrorResponse "权限不足"
// @Failure      404  {object}  ErrorResponse "用户不存在"
// @Failure      500  {object}  ErrorResponse "服务器错误"
// @Router       /users/{id} [delete]
// @Security     BearerAuth
func (h *UserHandler) DeleteUser(c *gin.Context) {
    // 实现代码...
}

// UpdateUserPassword 修改用户密码
// @Summary      修改用户密码
// @Description  修改指定用户的登录密码
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        id      path     int                    true "用户ID"
// @Param        request body     UpdatePasswordRequest  true "修改密码请求"
// @Success      200     {object} Response              "修改成功"
// @Failure      400     {object} ErrorResponse         "参数错误"
// @Failure      401     {object} ErrorResponse         "未授权"
// @Failure      403     {object} ErrorResponse         "权限不足"
// @Failure      404     {object} ErrorResponse         "用户不存在"
// @Failure      500     {object} ErrorResponse         "服务器错误"
// @Router       /users/{id}/password [put]
// @Security     BearerAuth
func (h *UserHandler) UpdateUserPassword(c *gin.Context) {
    // 实现代码...
}
```

#### 认证 API

**文件位置**: `internal/handlers/auth_handler.go`

```go
// Login 用户登录
// @Summary      用户登录
// @Description  使用用户名和密码登录系统，返回 JWT Token
// @Tags         认证授权
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "登录请求"
// @Success      200 {object} LoginResponse "登录成功"
// @Failure      400 {object} ErrorResponse "参数错误"
// @Failure      401 {object} ErrorResponse "认证失败"
// @Failure      429 {object} ErrorResponse "请求过于频繁"
// @Failure      500 {object} ErrorResponse "服务器错误"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
    // 实现代码...
}

// Register 用户注册
// @Summary      用户注册
// @Description  注册新用户账号
// @Tags         认证授权
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "注册请求"
// @Success      201 {object} UserResponse "注册成功"
// @Failure      400 {object} ErrorResponse "参数错误"
// @Failure      409 {object} ErrorResponse "用户已存在"
// @Failure      500 {object} ErrorResponse "服务器错误"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
    // 实现代码...
}

// RefreshToken 刷新 Token
// @Summary      刷新访问令牌
// @Description  使用刷新令牌获取新的访问令牌
// @Tags         认证授权
// @Accept       json
// @Produce      json
// @Param        request body RefreshTokenRequest true "刷新请求"
// @Success      200 {object} TokenResponse "刷新成功"
// @Failure      400 {object} ErrorResponse "参数错误"
// @Failure      401 {object} ErrorResponse "刷新令牌无效"
// @Failure      500 {object} ErrorResponse "服务器错误"
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
    // 实现代码...
}

// Logout 用户登出
// @Summary      用户登出
// @Description  登出当前用户并使 Token 失效
// @Tags         认证授权
// @Accept       json
// @Produce      json
// @Success      200 {object} Response "登出成功"
// @Failure      401 {object} ErrorResponse "未授权"
// @Failure      500 {object} ErrorResponse "服务器错误"
// @Router       /auth/logout [post]
// @Security     BearerAuth
func (h *AuthHandler) Logout(c *gin.Context) {
    // 实现代码...
}
```

#### 会议管理 API

**文件位置**: `internal/handlers/conference_handler.go`

```go
// ListConferences 获取会议列表
// @Summary      获取会议列表
// @Description  分页获取会议列表，支持日期范围和状态过滤
// @Tags         会议管理
// @Accept       json
// @Produce      json
// @Param        page       query    int      false "页码" minimum(1) default(1)
// @Param        size       query    int      false "每页数量" minimum(1) maximum(100) default(20)
// @Param        start_date query    string  false "开始日期" format(date) "2024-01-01"
// @Param        end_date   query    string  false "结束日期" format(date) "2024-12-31"
// @Param        status     query    string  false "会议状态" Enums(pending, running, stopped)
// @Success      200 {object} ConferenceListResponse "成功响应"
// @Failure      400 {object} ErrorResponse "参数错误"
// @Failure      401 {object} ErrorResponse "未授权"
// @Failure      500 {object} ErrorResponse "服务器错误"
// @Router       /conferences [get]
// @Security     BearerAuth
func (h *ConferenceHandler) ListConferences(c *gin.Context) {
    // 实现代码...
}

// CreateConference 创建会议
// @Summary      创建会议
// @Description  创建一个新的视频会议
// @Tags         会议管理
// @Accept       json
// @Produce      json
// @Param        request body CreateConferenceRequest true "创建会议请求"
// @Success      201 {object} ConferenceResponse "创建成功"
// @Failure      400 {object} ErrorResponse "参数错误"
// @Failure      401 {object} ErrorResponse "未授权"
// @Failure      403 {object} ErrorResponse "权限不足"
// @Failure      500 {object} ErrorResponse "服务器错误"
// @Router       /conferences [post]
// @Security     BearerAuth
func (h *ConferenceHandler) CreateConference(c *gin.Context) {
    // 实现代码...
}

// StartConference 启动会议
// @Summary      启动会议
// @Description  启动指定会议，连接到华为会议系统
// @Tags         会议管理
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "会议ID"
// @Success      200  {object}  ConferenceResponse "启动成功"
// @Failure      400  {object}  ErrorResponse "参数错误"
// @Failure      401  {object}  ErrorResponse "未授权"
// @Failure      403  {object}  ErrorResponse "权限不足"
// @Failure      404  {object}  ErrorResponse "会议不存在"
// @Failure      409  {object}  ErrorResponse "会议状态不允许启动"
// @Failure      500  {object}  ErrorResponse "服务器错误"
// @Router       /conferences/{id}/start [post]
// @Security     BearerAuth
func (h *ConferenceHandler) StartConference(c *gin.Context) {
    // 实现代码...
}

// StopConference 停止会议
// @Summary      停止会议
// @Description  停止正在进行的会议
// @Tags         会议管理
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "会议ID"
// @Success      200  {object}  ConferenceResponse "停止成功"
// @Failure      400  {object}  ErrorResponse "参数错误"
// @Failure      401  {object}  ErrorResponse "未授权"
// @Failure      403  {object}  ErrorResponse "权限不足"
// @Failure      404  {object}  ErrorResponse "会议不存在"
// @Failure      409  {object}  ErrorResponse "会议状态不允许停止"
// @Failure      500  {object}  ErrorResponse "服务器错误"
// @Router       /conferences/{id}/stop [post]
// @Security     BearerAuth
func (h *ConferenceHandler) StopConference(c *gin.Context) {
    // 实现代码...
}

// StartRecording 开始录制
// @Summary      开始会议录制
// @Description  开始录制指定会议的视频
// @Tags         会议管理
// @Accept       json
// @Produce      json
// @Param        id      path      int                    true "会议ID"
// @Param        request body     StartRecordingRequest  true "录制请求"
// @Success      200     {object}  RecordingResponse      "录制开始成功"
// @Failure      400     {object}  ErrorResponse          "参数错误"
// @Failure      401     {object}  ErrorResponse          "未授权"
// @Failure      403     {object}  ErrorResponse          "权限不足"
// @Failure      404     {object}  ErrorResponse          "会议不存在"
// @Failure      409     {object}  ErrorResponse          "会议状态不允许录制"
// @Failure      500     {object}  ErrorResponse          "服务器错误"
// @Router       /conferences/{id}/recording/start [post]
// @Security     BearerAuth
func (h *ConferenceHandler) StartRecording(c *gin.Context) {
    // 实现代码...
}

// StopRecording 停止录制
// @Summary      停止会议录制
// @Description  停止正在进行的会议录制
// @Tags         会议管理
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "会议ID"
// @Success      200  {object}  RecordingResponse "停止成功"
// @Failure      400  {object}  ErrorResponse "参数错误"
// @Failure      401  {object}  ErrorResponse "未授权"
// @Failure      403  {object}  ErrorResponse "权限不足"
// @Failure      404  {object}  ErrorResponse "会议不存在"
// @Failure      409  {object}  ErrorResponse "没有进行中的录制"
// @Failure      500  {object}  ErrorResponse "服务器错误"
// @Router       /conferences/{id}/recording/stop [post]
// @Security     BearerAuth
func (h *ConferenceHandler) StopRecording(c *gin.Context) {
    // 实现代码...
}
```

### 4. 模型注释

**文件位置**: `pkg/models/response.go`

```go
package models

// Response 通用响应结构
// @Description  API 通用响应格式
type Response struct {
    Code    string      `json:"code" example:"SUCCESS"`       // 响应码
    Message string      `json:"message" example:"操作成功"`    // 响应消息
    Data    interface{} `json:"data"`                          // 响应数据
}

// ErrorResponse 错误响应结构
// @Description  错误响应格式
type ErrorResponse struct {
    Code    string                 `json:"code" example:"VALIDATION_ERROR"` // 错误码
    Message string                 `json:"message" example:"参数验证失败"`   // 错误消息
    Details map[string]interface{} `json:"details"`                          // 详细信息
}

// PaginationMeta 分页元数据
// @Description  分页信息
type PaginationMeta struct {
    Page      int64 `json:"page" example:"1"`            // 当前页码
    PageSize  int64 `json:"page_size" example:"20"`      // 每页数量
    Total     int64 `json:"total" example:"100"`         // 总记录数
    TotalPage int64 `json:"total_page" example:"5"`      // 总页数
}

// UserResponse 用户响应
// @Description  用户信息响应
type UserResponse struct {
    ID        uint      `json:"id" example:"1"`                         // 用户ID
    Username  string    `json:"username" example:"admin"`                // 用户名
    Email     string    `json:"email" example:"admin@example.com"`      // 邮箱
    Phone     string    `json:"phone" example:"13800138000"`            // 电话
    Avatar    string    `json:"avatar" example:"https://..."`           // 头像URL
    Role      Role      `json:"role"`                                   // 角色
    Status    string    `json:"status" example:"active"`                // 状态
    CreatedAt time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"` // 创建时间
    UpdatedAt time.Time `json:"updated_at" example:"2024-01-01T00:00:00Z"` // 更新时间
}

// UserListResponse 用户列表响应
// @Description  用户列表响应
type UserListResponse struct {
    Code    string          `json:"code" example:"SUCCESS"`
    Message string          `json:"message" example:"获取成功"`
    Data    UserListData    `json:"data"`
}

// UserListData 用户列表数据
type UserListData struct {
    Items      []UserResponse  `json:"items"`       // 用户列表
    Pagination PaginationMeta `json:"pagination"`  // 分页信息
}

// ConferenceResponse 会议响应
// @Description  会议信息响应
type ConferenceResponse struct {
    ID          string              `json:"id" example:"conf-123"`                 // 会议ID
    Name        string              `json:"name" example："项目周会"`                // 会议名称
    Description string              `json:"description" example："讨论项目进展"`      // 会议描述
    StartTime   time.Time           `json:"start_time" example:"2024-01-01T10:00:00Z"` // 开始时间
    EndTime     time.Time           `json:"end_time" example:"2024-01-01T11:00:00Z"`   // 结束时间
    Status      ConferenceStatus    `json:"status" example:"running"`              // 状态
    Creator     UserResponse        `json:"creator"`                               // 创建者
    Attendees   int                 `json:"attendees" example:"10"`                // 参会人数
    Recording   *RecordingResponse  `json:"recording"`                             // 录制信息
    CreatedAt   time.Time           `json:"created_at"`                            // 创建时间
    UpdatedAt   time.Time           `json:"updated_at"`                            // 更新时间
}

// RecordingResponse 录制响应
// @Description  录制信息响应
type RecordingResponse struct {
    ID            string         `json:"id" example:"rec-123"`                   // 录制ID
    ConferenceID  string         `json:"conference_id" example:"conf-123"`       // 会议ID
    Status        RecordingStatus `json:"status" example:"recording"`           // 录制状态
    RTSPUrl       string         `json:"rtsp_url" example:"rtsp://..."`         // RTSP地址
    StartTime     time.Time      `json:"start_time"`                            // 开始时间
    Duration      int64          `json:"duration" example:"3600"`              // 持续时间(秒)
    FileSize      int64          `json:"file_size" example:"104857600"`        // 文件大小(字节)
    FilePath      string         `json:"file_path" example:"/videos/..."`      // 文件路径
}

// LoginRequest 登录请求
type LoginRequest struct {
    Username string `json:"username" binding:"required" example:"admin"`        // 用户名
    Password string `json:"password" binding:"required" example:"Admin@123"`    // 密码
}

// LoginResponse 登录响应
type LoginResponse struct {
    AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`  // 访问令牌
    RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIs..."` // 刷新令牌
    ExpiresIn    int64  `json:"expires_in" example:"3600"`                      // 过期时间(秒)
    TokenType    string `json:"token_type" example:"Bearer"`                   // 令牌类型
    User         UserResponse `json:"user"`                                     // 用户信息
}
```

## 生成文档

### 1. 生成命令

```bash
# 在项目根目录执行
swag init -g cmd/server/main.go -o docs

# 常用选项
swag init \
  --generalInfo cmd/server/main.go \  # 主文件路径
  --output docs \                      # 输出目录
  --parseDependency \                  # 解析依赖
  --parseInternal \                    # 解析内部包
  --parseDepth 5 \                     # 解析深度
  --markdownFiles docs/md \            # Markdown 文档目录
  --outputTypes go,json,yaml           # 输出格式
```

### 2. Makefile 集成

**文件位置**: `Makefile`

```makefile
.PHONY: generate swagger

# 生成 Swagger 文档
generate:
	@echo "生成 Swagger 文档..."
	swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# 仅生成 JSON
swagger-json:
	@echo "生成 Swagger JSON..."
	swag init -g cmd/server/main.go -o docs --outputTypes json

# 格式化 Swagger 注释
swagger-fmt:
	@echo "格式化 Swagger 注释..."
	swag fmt

# 验证 Swagger 文档
swagger-validate:
	@echo "验证 Swagger 文档..."
	swag fmt --dir . && swag init -g cmd/server/main.go -o docs
```

### 3. CI/CD 集成

**文件位置**: `.github/workflows/swagger.yml`

```yaml
name: Swagger Docs

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  swagger:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install swag
        run: go install github.com/swaggo/swag/cmd/swag@latest

      - name: Generate swagger docs
        run: make generate

      - name: Check for changes
        run: |
          if [ -n "$(git diff --name-only docs/)" ]; then
            echo "Swagger 文档需要更新"
            git diff docs/
            exit 1
          fi
```

## 访问文档

### 1. Swagger UI

启动应用后，访问：

```
http://localhost:8080/swagger/index.html
```

### 2. JSON 格式

```
http://localhost:8080/swagger/doc.json
```

### 3. YAML 格式

```
http://localhost:8080/swagger/doc.yaml
```

## 高级功能

### 1. 多文件文档

对于大型项目，可以将文档拆分为多个文件：

```
docs/
├── md/
│   ├── api/
│   │   ├── user.md
│   │   ├── conference.md
│   │   └── recording.md
│   └── models/
│       ├── user.md
│       └── conference.md
```

在 `user.md` 中：

```markdown
# 用户管理 API

## 用户列表

获取系统中的所有用户。

### 请求

`GET /api/v1/users`

### 查询参数

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| size | int | 否 | 每页数量，默认 20 |
| keyword | string | 否 | 搜索关键词 |
| status | string | 否 | 用户状态: active, inactive, locked |

### 响应

<details>
<summary>200 OK</summary>

```json
{
  "code": "SUCCESS",
  "message": "获取成功",
  "data": {
    "items": [
      {
        "id": 1,
        "username": "admin",
        "email": "admin@example.com",
        ...
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 100,
      "total_page": 5
    }
  }
}
```
</details>
```

### 2. 自定义配置

```go
// @host      api.example.com
// @BasePath  /v1

// 开发环境
// @host      localhost:8080
// @BasePath  /api/v1
```

### 3. 多认证方式

```go
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

// @securityDefinitions.oauth2.implicit OAuth2
// @authorizationUrl https://example.com/oauth/authorize
// @scope.read Grants read access
// @scope.write Grants write access

// 在 API 中使用
// @Security BearerAuth
// @Security ApiKeyAuth
// @Security OAuth2
```

### 4. 枚举定义

```go
// Role 用户角色
// @Description 用户角色枚举
type Role string

// @Enum Admin, User, Operator
const (
    RoleAdmin     Role = "admin"
    RoleUser      Role = "user"
    RoleOperator  Role = "operator"
)

// ConferenceStatus 会议状态
type ConferenceStatus string

// @Enum pending,running,stopped,failed
const (
    StatusPending  ConferenceStatus = "pending"
    StatusRunning  ConferenceStatus = "running"
    StatusStopped  ConferenceStatus = "stopped"
    StatusFailed   ConferenceStatus = "failed"
)
```

## 注释最佳实践

### 1. 组织 Tags

```go
// 建议的 Tag 组织方式
// @Tags 认证授权        // auth
// @Tags 用户管理        // users
// @Tags 会议管理        // conferences
// @Tags 录制管理        // recordings
// @Tags 文件管理        // files
// @Tags 系统管理        // system
// @Tags 监控统计        // monitoring
```

### 2. 错误码标准化

```go
// 统一使用预定义的错误码
// @Failure 400 {object} ErrorResponse "参数错误 (VALIDATION_ERROR)"
// @Failure 401 {object} ErrorResponse "未授权 (UNAUTHORIZED)"
// @Failure 403 {object} ErrorResponse "权限不足 (FORBIDDEN)"
// @Failure 404 {object} ErrorResponse "资源不存在 (NOT_FOUND)"
// @Failure 409 {object} ErrorResponse "资源冲突 (CONFLICT)"
// @Failure 500 {object} ErrorResponse "服务器错误 (INTERNAL_ERROR)"
```

### 3. 响应模型复用

```go
// 定义可复用的响应模型

// StandardResponse 标准成功响应
// @Description 标准成功响应格式
type StandardResponse struct {
    Code    string      `json:"code" example:"SUCCESS"`
    Message string      `json:"message" example:"操作成功"`
    Data    interface{} `json:"data"`
}

// ValidationError 验证错误响应
// @Description 参数验证错误响应
type ValidationError struct {
    Code    string            `json:"code" example:"VALIDATION_ERROR"`
    Message string            `json:"message" example:"参数验证失败"`
    Details map[string]string `json:"details"`
}

// 在 API 中使用
// @Success 200 {object} StandardResponse "成功"
// @Failure 400 {object} ValidationError "验证失败"
```

### 4. 示例值

```go
// 为字段添加有意义的示例值
type CreateUserRequest struct {
    Username string `json:"username" binding:"required" example:"john_doe" minLength:"4" maxLength:"32"`
    Email    string `json:"email" binding:"required,email" example:"john@example.com"`
    Password string `json:"password" binding:"required" example:"SecurePass123"`
    Role     string `json:"role" binding:"required" example:"user" Enums("admin", "user", "operator")`
}

type UserResponse struct {
    ID        uint      `json:"id" example:"42"`
    Username  string    `json:"username" example:"john_doe"`
    Email     string    `json:"email" example:"john@example.com"`
    CreatedAt time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
    UpdatedAt time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}
```

## 文档维护

### 1. 版本控制

```go
// 在主文件中定义版本
// @version 1.0.0
// @description Version 1.0.0 - Initial release

// 或使用环境变量
// @version {{env.API_VERSION}}
```

### 2. 变更日志

维护 `CHANGELOG.md` 记录 API 变更：

```markdown
# API 变更日志

## [1.1.0] - 2024-01-15

### Added
- 用户头像上传接口
- 会议模板功能

### Changed
- 用户列表新增分页参数
- 登录响应新增 refresh_token

### Deprecated
- 旧版会议接口将于 v2.0 移除

### Removed
- 移除 PPT 生成接口

### Fixed
- 修复会议时间格式错误
```

### 3. 自动检查

在 pre-commit hook 中检查文档：

```bash
#!/bin/bash
# .git/hooks/pre-commit

# 检查 Swagger 文档是否需要更新
make generate

if [ -n "$(git diff --name-only docs/)" ]; then
    echo "❌ Swagger 文档需要更新，请先执行 make generate"
    exit 1
fi
```

## 常见问题

### 1. 生成的文档为空

**原因**：注释格式错误或缺少必需字段

**解决**：检查是否包含 `@Summary`、`@Tags`、`@Router`

### 2. 模型未正确显示

**原因**：模型未导出或路径错误

**解决**：
- 确保模型是导出的（首字母大写）
- 使用完整的包路径

### 3. 多文件冲突

**原因**：多个 handler 重复定义相同的路由

**解决**：确保每个路由只在一个文件中定义

## 总结

通过使用 Swagger/OpenAPI：

1. **自动生成文档** - 从代码注释自动生成
2. **交互式测试** - Swagger UI 提供 Try it out 功能
3. **多语言支持** - 可生成客户端 SDK
4. **版本控制** - 文档与代码同步
5. **标准化** - 遵循 OpenAPI 规范
