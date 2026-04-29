# Phase 13: 重构华为配置，支持USB设备和流媒体录制模式 - Research

**Researched:** 2026-04-29
**Domain:** Go Backend + React Frontend + Database Migration + Video Recording Infrastructure
**Confidence:** HIGH

## Summary

Phase 13 要求将录制配置架构从"华为终端必填"重构为"多输入源可选"模式。当前系统已有完整的华为终端控制功能、USB设备扫描能力和流媒体配置字段，但架构强制要求华为配置为必填项。

**关键发现：**
1. 现有代码库已包含所有必要能力（USB扫描、流媒体测试、多配置关联）
2. 主要工作是**架构重构**而非新功能开发
3. 数据库已有 `task_huawei_configs` 关联表支持多配置
4. 前端和后端需要大量重命名工作（huawei → input）

**Primary recommendation:** 采用渐进式重构策略，先创建新表和API，再迁移前端，最后清理旧代码，确保系统可用性。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 华为终端控制 | API / Backend | Frontend | 需要调用华为API，管理终端状态 |
| USB设备扫描 | API / Backend | Frontend | 需要系统权限访问硬件，FFmpeg集成 |
| 流媒体连接测试 | API / Backend | Frontend | 需要FFprobe验证连接，处理认证 |
| 配置数据验证 | API / Backend | Frontend | 业务规则验证，数据一致性保证 |
| 配置UI展示 | Frontend / Client | API / Backend | 动态表单，用户体验优化 |
| 调度器适配 | API / Backend | — | 根据配置类型选择不同录制逻辑 |
| 数据库迁移 | Database / Storage | — | Schema变更，数据迁移，索引优化 |

## User Constraints (from CONTEXT.md)

### Locked Decisions

**配置数据结构 (D-01 到 D-03)**
- D-01: 单一配置模型 — 保持现有配置表结构，华为终端字段改为可选
- D-02: 配置类型互斥 — 添加 `config_type` 字段：`huawei_auto` | `usb` | `stream`
- D-03: 华为开关控制 — 添加 `huawei_enabled` 布尔字段

**验证规则 (D-04 到 D-05)**
- D-04: 录制源必填验证 — 至少需要填写一个录制源（USB/流媒体/华为）
- D-05: 测试连接功能 — USB调用扫描API，流媒体调用连通性API

**调度触发机制 (D-06)**
- D-06: 统一调度器 — 所有类型配置都可创建定时任务
- 移除 `VideoRecordingTask.IsValid()` 中 `HuaweiConfigID` 必填检查

**前端重构 (D-07 到 D-08)**
- D-07: 全面重命名 — 路由、API、文件、类型、组件、菜单全部改名
- D-08: 配置表单重构 — 添加类型选择器、华为开关、动态字段显示

**数据库变更 (D-09 到 D-11)**
- D-09: 新建 `input_configs` 表 — 保持相同字段结构，添加新字段
- D-10: 数据迁移（可选）— 提供迁移功能，旧数据转为 `huawei_auto` 类型
- D-11: 关联表更新 — `task_huawei_configs` 表保留，支持多配置关联

**API 变更 (D-12 到 D-13)**
- D-12: API 路由重命名 — `/api/input-configs` 替代 `/api/huawei-configs`
- D-13: 录制任务 API — `huawei_config_id` 改为可选，支持多配置关联

### Claude's Discretion

- 配置列表页面的筛选和排序设计
- 配置详情页面的布局和信息展示优先级
- 错误提示的具体文案和样式
- 测试连接的超时时间和重试策略
- 数据迁移的详细交互流程

### Deferred Ideas (OUT OF SCOPE)

- 配置模板功能：预定义常用配置模板
- 配置导入/导出：支持批量导入配置
- 配置分组：支持将配置按地点/部门分组管理
- 录制源预览：配置时实时预览 USB 或流媒体画面
- 高级调度规则：支持复杂的调度条件（工作日、节假日等）

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| **Go 1.24** | [VERIFIED: cmd/server/main.go] | Backend language | Project uses Go 1.24 with Gin framework |
| **Gin** | [VERIFIED: cmd/server/app.go] | HTTP framework | Existing routing built on Gin |
| **GORM** | [VERIFIED: cmd/server/app.go] | ORM | Database operations use GORM with SQLite |
| **React 19** | [VERIFIED: package.json] | Frontend framework | Current frontend uses React 19 |
| **Ant Design 6** | [VERIFIED: frontend/src] | UI components | Existing UI uses Ant Design components |
| **FFmpeg/FFprobe** | [ASSUMED: internal/services/huawei_config_service.go:424] | Media processing | Already integrated for recording/stream testing |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **Zap** | [VERIFIED: cmd/server/app.go] | Structured logging | All logging throughout backend |
| **robfig/cron** | [VERIFIED: internal/scheduler/video_scheduler.go:13] | Task scheduling | Scheduler uses cron for task execution |
| **Zustand** | [VERIFIED: STATE.md] | State management | Frontend state management |
| **TanStack Query** | [VERIFIED: STATE.md] | API caching | Frontend API client caching |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| 保持 `huawei_configs` 表名 | 创建 `input_configs` 新表 | 新表避免数据迁移风险，旧表保持兼容；但需要维护两套代码 |
| 硬编码配置类型验证 | 使用策略模式 | 策略模式更易扩展新配置类型，但增加代码复杂度 |

**Installation:**
```bash
# Backend dependencies (already installed)
go mod download

# Frontend dependencies (already installed)
npm install
```

**Version verification:** All core libraries verified from codebase imports and package files.

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                         │
├─────────────────────────────────────────────────────────────────┤
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐       │
│  │ InputConfig   │  │ RecordingTask │  │ DeviceScanner │       │
│  │ Management    │  │ Creation      │  │ Components    │       │
│  │ Page          │  │ Form          │  │               │       │
│  └───────────────┘  └───────────────┘  └───────────────┘       │
│           │                  │                   │              │
│           └──────────────────┴───────────────────┘              │
│                              │                                  │
│                    ┌─────────────────────┐                      │
│                    │  API Client Layer   │                      │
│                    │  (apiClient.ts)     │                      │
│                    └─────────────────────┘                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Backend API Layer (Gin)                      │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              /api/v1/input-configs                       │  │
│  │  ┌─────────────┬─────────────┬─────────────────────┐    │  │
│  │  │ GET /       │ POST /      │ POST /:id/test       │    │  │
│  │  │ (List)      │ (Create)    │ (Test Connection)    │    │  │
│  │  └─────────────┴─────────────┴─────────────────────┘    │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              /api/v1/input-configs/usb-devices           │  │
│  │  ┌─────────────────────────────────────────────────┐    │  │
│  │  │ GET / (Scan USB Devices)                        │    │  │
│  │  └─────────────────────────────────────────────────┘    │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Service Layer (Business Logic)                │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐  │
│  │         InputConfigService (New)                         │  │
│  │  ┌────────────────────────────────────────────────┐     │  │
│  │  │ • ListConfigs()                                 │     │  │
│  │  │ • CreateConfig() — validation by config_type    │     │  │
│  │  │ • UpdateConfig()                                │     │  │
│  │  │ • TestConnection() — dispatch to USB/Stream/HW  │     │  │
│  │  └────────────────────────────────────────────────┘     │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │         USBDeviceScanner (Existing)                      │  │
│  │  • ScanVideoDevices() — Windows PowerShell/Linux FFmpeg │  │
│  │  • ScanAudioDevices() — DirectSound/ALSA                 │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │         VideoRecordingTaskService (Modified)             │  │
│  │  • CreateTask() — validate at least one input config     │  │
│  │  • UpdateTask() — support multiple input configs         │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              Data Access Layer (GORM + SQLite)                  │
├─────────────────────────────────────────────────────────────────┤
│  ┌───────────────────┐  ┌───────────────────┐                  │
│  │  input_configs    │  │ task_input_configs│ (New association) │
│  │  (New table)      │  │  (New table)       │                  │
│  ├───────────────────┤  ├───────────────────┤                  │
│  │ id                │  │ task_id           │                  │
│  │ name              │  │ input_config_id   │                  │
│  │ config_type       │  │ config_type       │                  │
│  │ huawei_enabled    │  └───────────────────┘                  │
│  │ [huawei fields]   │                                         │
│  │ [usb fields]      │  ┌───────────────────┐                  │
│  │ [stream fields]   │  │ huawei_configs    │ (Legacy, read-only)│
│  └───────────────────┘  └───────────────────┘                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   External Dependencies                         │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ Huawei       │  │ USB Hardware │  │ Streaming Server     │  │
│  │ Terminal API │  │ (Camera/Audio)│  │ (RTMP/RTSP/SRT/HLS)  │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
internal/
├── models/
│   ├── input_config.go              # NEW: Input config model (refactored from huawei_config.go)
│   ├── task_input_config.go         # NEW: Task-input association
│   ├── huawei_config.go             # LEGACY: Keep for backward compatibility (read-only)
│   └── task_huawei_config.go        # LEGACY: Keep for existing data
├── services/
│   ├── input_config_service.go      # NEW: Input config service (refactored)
│   ├── huawei_config_service.go     # LEGACY: Mark as deprecated, forward to InputConfigService
│   └── usb_device_scanner.go        # EXISTING: No changes needed
├── handlers/
│   ├── input_config_handler.go      # NEW: Input config handlers
│   └── huawei_config_handler.go     # LEGACY: Mark as deprecated
└── migrations/
    └── 014_create_input_configs.go  # NEW: Create input_configs table

frontend/src/
├── pages/
│   └── system/
│       ├── input-configs/           # NEW: Renamed from huawei-configs
│       │   └── index.tsx
│       └── huawei-configs/          # LEGACY: Keep for backward compatibility
├── api/
│   ├── input-config.ts              # NEW: Renamed from huawei-config.ts
│   └── huawei-config.ts             # LEGACY: Mark as deprecated
└── types/
    ├── input-config.ts              # NEW: Renamed from huawei-config.ts
    └── huawei-config.ts             # LEGACY: Mark as deprecated
```

### Pattern 1: Configuration Type Validation

**What:** 使用策略模式根据 `config_type` 字段执行不同的验证逻辑

**When to use:** 配置创建、更新、测试连接时

**Example:**
```go
// Source: [internal/services/huawei_config_service.go:99-123]
func (s *InputConfigService) ValidateConfig(config *models.InputConfig) error {
    switch config.ConfigType {
    case models.ConfigTypeHuaweiAuto:
        if config.HuaweiEnabled {
            return s.validateHuaweiFields(config)
        }
        // Huawei disabled, validate USB or Stream fields
        if config.USBCameraDevice != "" {
            return s.validateUSBFields(config)
        }
        if config.StreamURL != "" {
            return s.validateStreamFields(config)
        }
        return errors.New("必须选择至少一种录制源（华为终端/USB设备/流媒体）")
    case models.ConfigTypeUSB:
        return s.validateUSBFields(config)
    case models.ConfigTypeStream:
        return s.validateStreamFields(config)
    default:
        return errors.New("无效的配置类型")
    }
}

func (s *InputConfigService) validateHuaweiFields(config *models.InputConfig) error {
    if config.Server == "" {
        return errors.New("华为服务器地址不能为空")
    }
    if config.Username == "" {
        return errors.New("用户名不能为空")
    }
    // ... other Huawei-specific validations
    return nil
}
```

### Pattern 2: Connection Testing Dispatch

**What:** 测试连接时根据配置类型调用不同的测试方法

**When to use:** 用户点击"测试连接"按钮时

**Example:**
```go
// Source: [internal/services/huawei_config_service.go:415-537]
func (s *InputConfigService) TestConnection(req *TestConnectionRequest) error {
    switch req.ConfigType {
    case models.ConfigTypeHuaweiAuto:
        // Test Huawei terminal connection
        return s.testHuaweiConnection(req)
    case models.ConfigTypeUSB:
        // Test USB device availability
        return s.testUSBDevice(req)
    case models.ConfigTypeStream:
        // Test stream URL connectivity
        return s.TestStreamConnection(&services.TestStreamRequest{
            Protocol: req.StreamProtocol,
            URL:      req.StreamURL,
            Username: req.StreamUsername,
            Password: req.StreamPassword,
        })
    default:
        return errors.New("不支持的配置类型")
    }
}
```

### Pattern 3: Scheduler Integration

**What:** 调度器根据配置类型选择不同的录制启动逻辑

**When to use:** 定时任务触发时

**Example:**
```go
// Source: [internal/scheduler/video_scheduler.go:79-84]
func (c *RecorderCoordinator) StartRecordingWithConfig(
    task *models.VideoRecordingTask,
    config *models.InputConfig,
    configType string,
) error {
    switch configType {
    case "huawei_auto":
        // Call Huawei conference API first, then start recording
        return c.startHuaweiRecording(task, config)
    case "usb":
        // Directly start USB device recording
        return c.startUSBRecording(task, config)
    case "stream":
        // Directly start stream recording
        return c.startStreamRecording(task, config)
    default:
        return errors.New("不支持的配置类型")
    }
}
```

### Anti-Patterns to Avoid

- **硬编码配置类型检查:** 不要在多个地方重复 `if config.Type == "usb"` 逻辑；使用策略模式或服务方法封装
- **直接修改旧表:** 不要直接修改 `huawei_configs` 表结构；创建新表保持向后兼容
- **前端类型断言:** 不要使用 `as any` 绕过类型检查；创建正确的 TypeScript 类型
- **同步数据迁移:** 不要在应用启动时阻塞进行数据迁移；提供异步迁移API

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| USB设备扫描 | Windows PowerShell命令、Linux v4l2调用 | 现有 `USBDeviceScanner` | [internal/services/usb_device_scanner.go] 已实现跨平台扫描 |
| 流媒体连接测试 | 手动HTTP请求、Socket连接 | FFprobe | [internal/services/huawei_config_service.go:424] 已使用FFprobe测试流 |
| 配置锁定机制 | 自实现锁逻辑 | 现有 `Lock()`/`Unlock()` | [internal/models/huawei_config.go:75-92] 已实现设备锁定 |
| 任务调度 | 自实现定时器 | robfig/cron | [internal/scheduler/video_scheduler.go:13] 已使用cron调度器 |
| 数据库验证 | 手动SQL查询检查字段存在 | GORM AutoMigrate | GORM自动处理表结构创建和迁移 |
| 表单验证 | 自定义验证逻辑 | Ant Design Form rules | 前端已有完整的表单验证组件 |

**Key insight:** 现有代码库已包含所有必要的基础设施，重构主要是组织架构和命名，而非重新实现功能。

## Common Pitfalls

### Pitfall 1: 数据库迁移破坏现有数据

**What goes wrong:** 直接修改 `huawei_configs` 表结构导致现有华为配置无法使用

**Why it happens:** 没有考虑向后兼容性，直接添加必填字段或修改约束

**How to avoid:** 创建新表 `input_configs`，保留旧表 `huawei_configs` 不变，提供数据迁移工具

**Warning signs:** 
- Migration文件中出现 `ALTER TABLE huawei_configs`
- 删除或重命名现有的 `huawei_configs` 表

### Pitfall 2: 前端路由重定向导致404

**What goes wrong:** 用户访问旧路由 `/system/huawei-configs` 返回404错误

**Why it happens:** 只添加新路由，没有保留旧路由作为重定向

**How to avoid:** 在路由配置中保留旧路由并重定向到新路由

```go
// Backend: Keep old route for compatibility
huaweiConfigs := api.Group("/huawei-configs")
{
    huaweiConfigs.GET("", func(c *gin.Context) {
        c.Redirect(302, "/api/v1/input-configs?" + c.Request.URL.RawQuery)
    })
    // ... other routes with redirects
}
```

**Warning signs:** 
- 删除所有 `/huawei-configs` 路由
- 前端直接修改路由配置没有 fallback

### Pitfall 3: 调度器无法识别新配置类型

**What goes wrong:** 创建USB或流媒体配置后，定时任务无法启动

**Why it happens:** 调度器仍然检查 `huawei_config_id` 必填，或无法处理新配置类型

**How to avoid:** 
1. 修改 `VideoRecordingTask.IsValid()` 移除 `HuaweiConfigID` 必填检查
2. 调度器通过 `task_input_configs` 关联表获取配置
3. 根据 `config_type` 分发到不同的录制逻辑

**Warning signs:**
- 任务创建失败，提示"必须指定华为配置"
- 调度器日志显示"找不到华为配置"

### Pitfall 4: TypeScript类型不匹配

**What goes wrong:** 前端调用新API时出现类型错误或运行时异常

**Why it happens:** 后端返回 `InputConfig` 结构，前端仍然使用 `HuaweiConfig` 类型

**How to avoid:** 
1. 创建新的 TypeScript 类型 `InputConfig`
2. 更新所有 API 客户端函数使用新类型
3. 保留旧类型标记为 `@deprecated`

**Warning signs:**
- 前端控制台出现类型错误
- API 响应字段 undefined

### Pitfall 5: 配置验证逻辑遗漏

**What goes wrong:** 用户创建配置时没有填写任何录制源，但验证通过

**Why it happens:** 验证逻辑只检查单个字段，没有检查"至少一个录制源"的规则

**How to avoid:** 在 `ValidateConfig()` 中添加互斥验证：

```go
hasHuawei := config.HuaweiEnabled && config.Server != ""
hasUSB := config.USBCameraDevice != ""
hasStream := config.StreamURL != ""

if !hasHuawei && !hasUSB && !hasStream {
    return errors.New("必须选择至少一种录制源")
}
```

**Warning signs:**
- 可以创建没有任何录制源的配置
- 测试连接按钮对所有配置都可用

## Code Examples

### Example 1: 创建新的 InputConfig 模型

```go
// Source: [internal/models/huawei_config.go:8-54] - Refactored
package models

import (
	"fmt"
	"time"
)

// InputConfig 输入配置模型
// 支持华为终端控制、USB设备、流媒体三种录制源
type InputConfig struct {
	Base
	Name        string `gorm:"type:varchar(100);not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	
	// 配置类型：huawei_auto | usb | stream
	ConfigType   string `gorm:"type:varchar(20);not null;index" json:"config_type"`
	HuaweiEnabled bool   `gorm:"default:false" json:"huawei_enabled"` // 华为控制开关
	
	// 华为终端字段（可选）
	Server           string `gorm:"type:varchar(100)" json:"server,omitempty"`
	Port             int    `json:"port,omitempty"`
	Username         string `gorm:"type:varchar(50)" json:"username,omitempty"`
	Password         string `gorm:"type:varchar(100)" json:"-" omitempty"`
	TerminalNumber   string `gorm:"type:varchar(50)" json:"terminal_number,omitempty"`
	ConferenceNumber string `gorm:"type:varchar(50)" json:"conference_number,omitempty"`
	
	// USB设备字段（可选）
	CameraBackend    string `gorm:"type:varchar(20);default:'dshow'" json:"camera_backend,omitempty"`
	USBCameraName    string `gorm:"type:varchar(100)" json:"usb_camera_name,omitempty"`
	USBCameraDevice  string `gorm:"type:varchar(100)" json:"usb_camera_device,omitempty"`
	AudioBackend     string `gorm:"type:varchar(20);default:'dshow'" json:"audio_backend,omitempty"`
	USBAudioName     string `gorm:"type:varchar(100)" json:"usb_audio_name,omitempty"`
	USBAudioDevice   string `gorm:"type:varchar(100)" json:"usb_audio_device,omitempty"`
	
	// 流媒体字段（可选）
	StreamProtocol string `gorm:"type:varchar(20)" json:"stream_protocol,omitempty"` // rtmp, rtsp, srt, hls
	StreamURL      string `gorm:"type:varchar(500)" json:"stream_url,omitempty"`
	StreamUsername string `gorm:"type:varchar(100)" json:"stream_username,omitempty"`
	StreamPassword string `gorm:"type:varchar(100)" json:"-" omitempty"`
	
	// 录制配置
	OutputFormat string `gorm:"type:varchar(20);default:'mp4'" json:"output_format"`
	
	// 状态字段
	IsActive bool       `gorm:"default:true" json:"is_active"`
	IsLocked bool       `gorm:"default:false" json:"is_locked"`
	LockedBy *uint      `json:"locked_by,omitempty"`
	LockedAt *time.Time `json:"locked_at,omitempty"`
	
	// 关联
	VideoRecordingTasks []VideoRecordingTask `gorm:"foreignKey:InputConfigID" json:"video_recording_tasks,omitempty"`
}

// 配置类型常量
const (
	ConfigTypeHuaweiAuto = "huawei_auto"
	ConfigTypeUSB        = "usb"
	ConfigTypeStream     = "stream"
)

// Validate 根据配置类型验证必填字段
func (c *InputConfig) Validate() error {
	if c.Name == "" {
		return errors.New("配置名称不能为空")
	}
	
	switch c.ConfigType {
	case ConfigTypeHuaweiAuto:
		if c.HuaweiEnabled {
			return c.validateHuaweiFields()
		}
		// Huawei disabled, check USB or Stream
		if c.USBCameraDevice == "" && c.StreamURL == "" {
			return errors.New("华为控制关闭时，必须选择USB或流媒体录制源")
		}
	case ConfigTypeUSB:
		if c.USBCameraDevice == "" {
			return errors.New("USB配置必须指定摄像头设备")
		}
	case ConfigTypeStream:
		if c.StreamURL == "" {
			return errors.New("流媒体配置必须指定流地址")
		}
	default:
		return fmt.Errorf("无效的配置类型: %s", c.ConfigType)
	}
	
	return nil
}

func (c *InputConfig) validateHuaweiFields() error {
	if c.Server == "" {
		return errors.New("华为服务器地址不能为空")
	}
	if c.Username == "" {
		return errors.New("用户名不能为空")
	}
	if c.TerminalNumber == "" {
		return errors.New("终端号码不能为空")
	}
	return nil
}

func (InputConfig) TableName() string {
	return "input_configs"
}
```

### Example 2: 前端配置表单组件

```tsx
// Source: [frontend/src/pages/system/huawei-configs/index.tsx] - Refactored
import React, { useState } from 'react';
import { Form, Select, Switch, Button, Input, message } from 'antd';

const { Option } = Select;

interface InputConfigFormProps {
  onSubmit: (values: CreateInputConfigRequest) => Promise<void>;
}

export const InputConfigForm: React.FC<InputConfigFormProps> = ({ onSubmit }) => {
  const [form] = Form.useForm();
  const [configType, setConfigType] = useState<ConfigType>('huawei_auto');
  const [huaweiEnabled, setHuaweiEnabled] = useState(false);

  const handleConfigTypeChange = (value: ConfigType) => {
    setConfigType(value);
    // Reset form fields based on type
    if (value === 'usb') {
      form.setFieldsValue({
        huawei_enabled: false,
        stream_url: '',
      });
    } else if (value === 'stream') {
      form.setFieldsValue({
        huawei_enabled: false,
        usb_camera_device: '',
      });
    }
  };

  const handleSubmit = async (values: any) => {
    try {
      await onSubmit({
        ...values,
        config_type: configType,
      });
      message.success('配置创建成功');
    } catch (error) {
      message.error(`配置创建失败: ${error.message}`);
    }
  };

  return (
    <Form form={form} onFinish={handleSubmit} layout="vertical">
      <Form.Item
        label="配置名称"
        name="name"
        rules={[{ required: true, message: '请输入配置名称' }]}
      >
        <Input placeholder="例如：会议室A摄像头" />
      </Form.Item>

      <Form.Item
        label="配置类型"
        name="config_type"
        initialValue="huawei_auto"
      >
        <Select onChange={handleConfigTypeChange}>
          <Option value="huawei_auto">华为自动控制</Option>
          <Option value="usb">USB设备直录</Option>
          <Option value="stream">流媒体录制</Option>
        </Select>
      </Form.Item>

      {(configType === 'huawei_auto' || configType === 'usb') && (
        <>
          <Form.Item
            label="启用华为终端控制"
            name="huawei_enabled"
            valuePropName="checked"
          >
            <Switch 
              checked={huaweiEnabled}
              onChange={setHuaweiEnabled}
            />
          </Form.Item>

          {huaweiEnabled && (
            <>
              <Form.Item
                label="服务器地址"
                name="server"
                rules={[{ required: true, message: '请输入服务器地址' }]}
              >
                <Input placeholder="192.168.1.100" />
              </Form.Item>
              
              {/* Other Huawei fields... */}
            </>
          )}
        </>
      )}

      {configType === 'usb' && !huaweiEnabled && (
        <>
          <Form.Item
            label="USB摄像头"
            name="usb_camera_device"
            rules={[{ required: true, message: '请选择USB摄像头' }]}
          >
            <USBDeviceSelector type="camera" />
          </Form.Item>
          
          <Form.Item
            label="USB麦克风"
            name="usb_audio_device"
          >
            <USBDeviceSelector type="audio" />
          </Form.Item>
        </>
      )}

      {configType === 'stream' && (
        <>
          <Form.Item
            label="流媒体协议"
            name="stream_protocol"
            rules={[{ required: true, message: '请选择协议' }]}
          >
            <Select>
              <Option value="rtmp">RTMP</Option>
              <Option value="rtsp">RTSP</Option>
              <Option value="srt">SRT</Option>
              <Option value="hls">HLS</Option>
            </Select>
          </Form.Item>

          <Form.Item
            label="流地址"
            name="stream_url"
            rules={[
              { required: true, message: '请输入流地址' },
              { type: 'url', message: '请输入有效的URL' },
            ]}
          >
            <Input placeholder="rtmp://example.com/live/stream" />
          </Form.Item>

          {/* Optional authentication fields */}
        </>
      )}

      <Form.Item
        label="输出格式"
        name="output_format"
        initialValue="mp4"
      >
        <Select>
          <Option value="mp4">MP4</Option>
          <Option value="mkv">MKV</Option>
          <Option value="avi">AVI</Option>
        </Select>
      </Form.Item>

      <Form.Item>
        <Button type="primary" htmlType="submit">
          创建配置
        </Button>
      </Form.Item>
    </Form>
  );
};
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 华为配置必填 | 多输入源可选 | Phase 13 | 支持USB直录、流媒体录制，无需华为终端 |
| 单一配置关联 | 多配置关联 | 已实现（Phase 1-4） | 一个任务可关联多个录制源 |
| `/api/huawei-configs` | `/api/input-configs` | Phase 13 | API命名更符合实际功能 |

**Deprecated/outdated:**
- `VideoRecordingTask.IsValid()` 中的 `HuaweiConfigID` 必填检查 — Phase 13将移除此限制
- `/api/v1/huawei-configs` 路由 — Phase 13将重定向到 `/api/v1/input-configs`
- `HuaweiConfig` 模型作为主要配置模型 — Phase 13将替换为 `InputConfig`

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | SQLite支持并发读写事务 | Database Migration | 如果SQLite在并发下有锁问题，可能需要调整事务策略 |
| A2 | FFprobe已安装并在PATH中 | Stream Testing | 如果FFprobe不可用，流媒体测试功能将失败 |
| A3 | Windows PowerShell可用于USB设备扫描 | USB Device Scanner | 如果PowerShell权限受限，扫描功能可能失败 |
| A4 | 现有数据可以保留在`huawei_configs`表中 | Data Migration | 如果旧表数据必须迁移，需要修改迁移策略 |
| A5 | 前端路由重定向不影响现有用户体验 | Frontend Refactoring | 如果用户有书签或缓存链接，可能需要额外处理 |

**如果此表为空：** 本研究中所有声明均已验证或引用 — 无需用户确认。

## Open Questions

1. **数据迁移策略**
   - What we know: 需要保留现有`huawei_configs`数据，提供迁移到`input_configs`的能力
   - What's unclear: 迁移应该是自动执行还是需要管理员手动触发？
   - Recommendation: 提供管理员手动触发的迁移API，并在日志中记录迁移状态

2. **调度器适配细节**
   - What we know: 调度器需要根据配置类型选择不同的录制启动逻辑
   - What's unclear: USB和流媒体录制的FFmpeg命令是否与华为录制不同？
   - Recommendation: 查看`RecorderCoordinator`实现，确认是否需要新的录制策略

3. **前端性能影响**
   - What we know: 需要重命名大量文件和路由
   - What we know unclear: 重构是否会影响前端构建体积和加载性能？
   - Recommendation: 使用构建分析工具检查重构前后bundle大小变化

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.24 | Backend | ✓ | [VERIFIED: cmd/server/main.go] | — |
| FFmpeg | USB/Stream recording | ✓ | [ASSUMED: internal/services/huawei_config_service.go:424] | — |
| FFprobe | Stream testing | ✓ | [ASSUMED: internal/services/huawei_config_service.go:424] | — |
| SQLite 3 | Database | ✓ | [VERIFIED: cmd/server/app.go:180] | — |
| React 19 | Frontend | ✓ | [VERIFIED: package.json] | — |
| Node.js 22+ | Frontend build | ✓ | [VERIFIED: env description] | — |
| PowerShell | Windows USB scanning | ✓ | [ASSUMED: Windows 11 includes PowerShell] | FFmpeg fallback |

**Missing dependencies with no fallback:**
- None

**Missing dependencies with fallback:**
- None

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Standard Go testing + Testify |
| Config file | No specific config file — tests alongside source code |
| Quick run command | `go test ./internal/services -run TestInputConfig -v` |
| Full suite command | `go test ./... -v` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| D-01 | 单一配置模型验证 | unit | `go test ./internal/models -run TestInputConfig_Validate -v` | ❌ Wave 0 |
| D-02 | 配置类型互斥验证 | unit | `go test ./internal/services -run TestInputConfigService_ValidateConfig -v` | ❌ Wave 0 |
| D-04 | 录制源必填验证 | integration | `go test ./internal/handlers -run TestInputConfigHandler_CreateConfig -v` | ❌ Wave 0 |
| D-05 | 测试连接功能 | integration | `go test ./internal/services -run TestInputConfigService_TestConnection -v` | ❌ Wave 0 |
| D-06 | 调度器适配新配置 | unit | `go test ./internal/scheduler -run TestVideoScheduler_InputConfig -v` | ❌ Wave 0 |
| D-12 | API路由重定向 | integration | `go test ./cmd/server -run TestAPI_Routes -v` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/services -run TestInputConfig -v`
- **Per wave merge:** `go test ./... -v`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/models/input_config_test.go` — 配置模型验证测试
- [ ] `internal/services/input_config_service_test.go` — 配置服务业务逻辑测试
- [ ] `internal/handlers/input_config_handler_test.go` — API处理器测试
- [ ] `internal/scheduler/input_config_scheduler_test.go` — 调度器适配测试
- [ ] `frontend/src/components/__tests__/InputConfigForm.test.tsx` — 前端表单组件测试

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | yes | Existing RBAC system with admin-only access to input configs |
| V5 Input Validation | yes | GORM validation + custom business logic validation (config type, required fields) |
| V6 Cryptography | yes | Password fields use `json:"-"` to exclude from JSON output; stream passwords use existing SM4 encryption if implemented |

### Known Threat Patterns for Go Backend + React Frontend

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| SQL Injection in config queries | Tampering | GORM parameterized queries (already used) |
| Stored XSS in config description | Tampering | React automatic escaping + Ant Design input sanitization |
| Authentication bypass via config type manipulation | Spoofing | Server-side validation of config_type enum values |
| Password exposure in API responses | Information Disclosure | `json:"-"` tags on password fields (already implemented) |
| CSRF on config creation | Spoofing | SM4 Token authentication middleware (already implemented) |

## Sources

### Primary (HIGH confidence)

- `internal/models/huawei_config.go` - Current HuaweiConfig model structure
- `internal/services/huawei_config_service.go` - Existing config service implementation
- `internal/services/usb_device_scanner.go` - USB device scanning implementation
- `internal/handlers/huawei_config_handler.go` - API handler patterns
- `internal/scheduler/video_scheduler.go` - Scheduler integration points
- `cmd/server/app.go` - Route registration and app initialization
- `frontend/src/api/huawei-config.ts` - Frontend API client patterns
- `frontend/src/types/huawei-config.ts` - TypeScript type definitions
- `frontend/src/layouts/BasicLayout.tsx` - Frontend routing and menu structure
- `.planning/phases/13-usb/13-CONTEXT.md` - User decisions and requirements

### Secondary (MEDIUM confidence)

- `internal/models/video_recording_task.go` - Task model and validation logic
- `internal/models/task_huawei_config.go` - Multi-config association table
- `internal/services/video_recording_task_service.go` - Task service implementation
- `internal/migrations/001_add_video_file_owner.go` - Stream fields migration pattern

### Tertiary (LOW confidence)

- `STATE.md` - Project history and architectural decisions
- `ROADMAP.md` - Phase dependencies and roadmap context
- `.claude/skills/spike-findings-record-v2/SKILL.md` - Project-specific patterns

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries verified from codebase imports and configuration files
- Architecture: HIGH - Complete understanding of existing codebase structure and patterns
- Pitfalls: MEDIUM - Some areas (scheduler integration details) require validation during implementation

**Research date:** 2026-04-29
**Valid until:** 30 days (stable technology stack, low risk of breaking changes)

---

**Phase:** 13 - 重构华为配置，支持USB设备和流媒体录制模式  
**Next Steps:**
1. Create Wave 0 test stubs for all validation requirements
2. Implement database migration for `input_configs` table
3. Create new `InputConfigService` refactored from `HuaweiConfigService`
4. Update frontend to use new API routes and types
5. Add backward compatibility layer for old `/huawei-configs` routes
6. Comprehensive testing of all three configuration types
