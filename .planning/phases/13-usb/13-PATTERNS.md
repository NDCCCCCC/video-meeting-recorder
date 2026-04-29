# Phase 13: 重构华为配置，支持USB设备和流媒体录制模式 - Pattern Map

**Mapped:** 2026-04-29
**Files analyzed:** 12
**Analogs found:** 12 / 12

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/models/input_config.go` | model | CRUD | `internal/models/huawei_config.go` | exact |
| `internal/models/task_input_config.go` | model | CRUD | `internal/models/task_huawei_config.go` | exact |
| `internal/services/input_config_service.go` | service | CRUD | `internal/services/huawei_config_service.go` | exact |
| `internal/handlers/input_config_handler.go` | handler | request-response | `internal/handlers/huawei_config_handler.go` | exact |
| `internal/migrations/014_create_input_configs.go` | migration | transform | `internal/migrations/001_add_video_file_owner.go` | role-match |
| `frontend/src/api/input-config.ts` | api-client | request-response | `frontend/src/api/huawei-config.ts` | exact |
| `frontend/src/types/input-config.ts` | types | request-response | `frontend/src/types/huawei-config.ts` | exact |
| `frontend/src/pages/system/input-configs/index.tsx` | component | request-response | `frontend/src/pages/system/huawei-configs/index.tsx` | exact |
| `internal/scheduler/video_scheduler.go` (modified) | scheduler | event-driven | `internal/scheduler/video_scheduler.go` | self-modify |
| `internal/models/video_recording_task.go` (modified) | model | CRUD | `internal/models/video_recording_task.go` | self-modify |
| `cmd/server/app.go` (modified) | config | request-response | `cmd/server/app.go` | self-modify |
| `frontend/src/layouts/BasicLayout.tsx` (modified) | component | routing | `frontend/src/layouts/BasicLayout.tsx` | self-modify |

## Pattern Assignments

### `internal/models/input_config.go` (model, CRUD)

**Analog:** `internal/models/huawei_config.go`

**Imports pattern** (lines 1-6):
```go
package models

import (
	"fmt"
	"time"
)
```

**Model structure pattern** (lines 8-54):
```go
// HuaweiConfig 华为配置模型
// 包含华为终端连接配置和USB设备配置
// RTSP流作为独立输入源，配置在VideoRecordingTask中
type HuaweiConfig struct {
	Base
	Name             string `gorm:"type:varchar(100);not null" json:"name"`
	Description      string `gorm:"type:text" json:"description"`
	Server           string `gorm:"type:varchar(100);not null" json:"server"`
	// ... other fields with gorm tags
	IsActive bool       `gorm:"default:true" json:"is_active"`
	IsLocked bool       `gorm:"default:false" json:"is_locked"`
	LockedBy *uint      `json:"locked_by,omitempty"`
	LockedAt *time.Time `json:"locked_at,omitempty"`

	VideoRecordingTasks []VideoRecordingTask `gorm:"foreignKey:HuaweiConfigID" json:"video_recording_tasks,omitempty"`
}
```

**Validation method pattern** (lines 99-123):
```go
// Validate 验证华为配置
func (h *HuaweiConfig) Validate() error {
	var errs []string

	if h.Name == "" {
		errs = append(errs, "配置名称不能为空")
	}
	if h.Server == "" {
		errs = append(errs, "服务器地址不能为空")
	}
	// ... other validations

	if len(errs) > 0 {
		return fmt.Errorf("验证失败: %s", errs)
	}
	return nil
}
```

**Locking mechanism pattern** (lines 74-92):
```go
// Lock 锁定华为配置
func (h *HuaweiConfig) Lock(taskID uint) error {
	if h.IsLocked && h.LockedBy != nil && *h.LockedBy != taskID {
		return fmt.Errorf("配置已被其他任务锁定")
	}
	h.IsLocked = true
	now := time.Now()
	h.LockedBy = &taskID
	h.LockedAt = &now
	return nil
}

// Unlock 解锁华为配置
func (h *HuaweiConfig) Unlock() error {
	h.IsLocked = false
	h.LockedBy = nil
	h.LockedAt = nil
	return nil
}
```

**Table name pattern** (lines 125-128):
```go
// TableName 指定表名
func (HuaweiConfig) TableName() string {
	return "huawei_configs"
}
```

---

### `internal/models/task_input_config.go` (model, CRUD)

**Analog:** `internal/models/task_huawei_config.go`

**Pattern:** Create association table for many-to-many relationship between tasks and input configs, similar to existing `task_huawei_configs` table structure.

---

### `internal/services/input_config_service.go` (service, CRUD)

**Analog:** `internal/services/huawei_config_service.go`

**Imports pattern** (lines 1-16):
```go
package services

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)
```

**Service structure pattern** (lines 17-31):
```go
// HuaweiConfigService 华为配置服务
type HuaweiConfigService struct {
	db     *gorm.DB
	logger *zap.Logger
	config *config.Config
}

// NewHuaweiConfigService 创建华为配置服务
func NewHuaweiConfigService(db *gorm.DB, logger *zap.Logger, cfg *config.Config) *HuaweiConfigService {
	return &HuaweiConfigService{
		db:     db,
		logger: logger,
		config: cfg,
	}
}
```

**Request/Response DTOs pattern** (lines 33-104):
```go
// ListConfigsRequest 配置列表请求
type ListConfigsRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size" binding:"max=100"`
	Keyword  string `form:"keyword"`
	IsActive *bool  `form:"is_active"`
}

// CreateConfigRequest 创建配置请求
type CreateConfigRequest struct {
	Name             string `json:"name" binding:"required,max=100"`
	Description      string `json:"description" binding:"max=500"`
	Server           string `json:"server" binding:"required,max=100"`
	// ... other fields with validation tags
}
```

**List method pattern** (lines 106-143):
```go
// ListConfigs 获取配置列表
func (s *HuaweiConfigService) ListConfigs(req *ListConfigsRequest) (*ListConfigsResponse, error) {
	var configs []models.HuaweiConfig
	var total int64

	query := s.db.Model(&models.HuaweiConfig{}).Preload("VideoRecordingTasks")

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ? OR server LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 状态筛选
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.
		Offset(offset).
		Limit(req.PageSize).
		Order("id ASC").
		Find(&configs).Error; err != nil {
		return nil, err
	}

	return &ListConfigsResponse{
		Total: total,
		Items: configs,
	}, nil
}
```

**Create method pattern** (lines 154-206):
```go
// CreateConfig 创建配置
func (s *HuaweiConfigService) CreateConfig(req *CreateConfigRequest) (*models.HuaweiConfig, error) {
	config := &models.HuaweiConfig{
		Name:             req.Name,
		Description:      req.Description,
		Server:           req.Server,
		// ... map request fields to model
		IsActive:         true,
	}

	// 验证后端配置
	if err := s.validateBackendConfig(config); err != nil {
		return nil, err
	}

	// 设置默认值
	if config.CameraBackend == "" {
		config.CameraBackend = "dshow"
	}

	if err := s.db.Create(config).Error; err != nil {
		return nil, err
	}

	s.logger.Info("华为配置已创建",
		zap.Uint("config_id", config.ID),
		zap.String("name", config.Name),
	)

	return config, nil
}
```

**Stream testing pattern** (lines 415-537):
```go
// TestStreamConnection 测试流媒体连接
// 使用 FFprobe 检测流媒体源是否可用，超时时间 10 秒
func (s *HuaweiConfigService) TestStreamConnection(req *TestStreamRequest) error {
	s.logger.Info("测试流媒体连接",
		zap.String("protocol", req.Protocol),
		zap.String("url", req.URL),
	)

	// 检查 FFprobe 路径
	ffprobePath := s.config.FFmpeg.FFProbePath
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}

	// 构建输入参数
	var inputArgs []string
	switch req.Protocol {
	case "rtmp":
		inputArgs = []string{"-i", req.URL}
	case "rtsp":
		inputArgs = []string{"-rtsp_transport", "tcp", "-i", req.URL}
	// ... other protocols
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffprobePath, args...)
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("连接超时（15秒），请检查网络和流媒体地址是否正确")
	}

	// ... error handling and validation

	return nil
}
```

---

### `internal/handlers/input_config_handler.go` (handler, request-response)

**Analog:** `internal/handlers/huawei_config_handler.go`

**Imports pattern** (lines 1-12):
```go
package handlers

import (
	"fmt"
	"strings"

	"github.com/cpic/record_v2/internal/services"
	"github.com/cpic/record_v2/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)
```

**Handler structure pattern** (lines 13-31):
```go
// HuaweiConfigHandler 华为配置处理器
type HuaweiConfigHandler struct {
	configService *services.HuaweiConfigService
	logger        *zap.Logger
	usbScanner    *services.USBDeviceScanner
}

// NewHuaweiConfigHandler 创建华为配置处理器
func NewHuaweiConfigHandler(
	configService *services.HuaweiConfigService,
	logger *zap.Logger,
	usbScanner *services.USBDeviceScanner,
) *HuaweiConfigHandler {
	return &HuaweiConfigHandler{
		configService: configService,
		logger:        logger,
		usbScanner:    usbScanner,
	}
}
```

**List handler pattern** (lines 33-67):
```go
// ListConfigs 获取配置列表
// @Summary 获取华为配置列表
// @Description 分页获取华为配置列表，支持筛选
// @Tags 华为配置
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Success 200 {object} response.Response{data=services.ListConfigsResponse}
// @Router /api/v1/huawei-configs [get]
func (h *HuaweiConfigHandler) ListConfigs(c *gin.Context) {
	var req services.ListConfigsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	result, err := h.configService.ListConfigs(&req)
	if err != nil {
		h.logger.Error("Failed to list huawei configs", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取配置列表失败")
		return
	}

	response.GinSuccess(c, result)
}
```

**Create handler pattern** (lines 93-118):
```go
// CreateConfig 创建华为配置
// @Summary 创建华为配置
// @Tags 华为配置
// @Security Bearer
// @Accept json
// @Param request body services.CreateConfigRequest true "创建配置请求"
// @Success 200 {object} response.Response{data=models.HuaweiConfig}
// @Router /api/v1/huawei-configs [post]
func (h *HuaweiConfigHandler) CreateConfig(c *gin.Context) {
	var req services.CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	config, err := h.configService.CreateConfig(&req)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("Huawei config created", zap.Uint("config_id", config.ID))
	response.GinSuccess(c, config)
}
```

**USB device scanning pattern** (lines 196-206):
```go
// ScanUSBDevices 扫描USB设备
// @Summary 扫描系统USB设备
// @Tags 华为配置
// @Security Bearer
// @Success 200 {object} response.Response{data=map[string][]services.USBDeviceInfo}
// @Router /api/v1/huawei-configs/scan-devices [get]
func (h *HuaweiConfigHandler) ScanUSBDevices(c *gin.Context) {
	devices := h.usbScanner.ScanAllUSBDevices()
	response.GinSuccess(c, devices)
}
```

---

### `internal/migrations/014_create_input_configs.go` (migration, transform)

**Analog:** `internal/migrations/001_add_video_file_owner.go`

**Migration structure pattern** (lines 9-48):
```go
// AddVideoFileOwnerMigration 为 video_files 表添加 created_by 字段
type AddVideoFileOwnerMigration struct{}

func (m *AddVideoFileOwnerMigration) Name() string {
	return "001_add_video_file_owner"
}

func (m *AddVideoFileOwnerMigration) Up(db *gorm.DB) error {
	// 使用 PRAGMA table_info 检查列是否已存在（更可靠）
	var columnName string
	checkErr := db.Raw("SELECT name FROM pragma_table_info('video_files') WHERE name = 'created_by'").Scan(&columnName).Error

	// 如果查询成功且找到了列，说明已存在，跳过迁移
	if checkErr == nil && columnName != "" {
		return nil
	}

	// 列不存在，执行添加
	addResult := db.Exec("ALTER TABLE video_files ADD COLUMN created_by INTEGER NOT NULL DEFAULT 1")
	if addResult.Error != nil {
		if addResult.Error != nil && len(addResult.Error.Error()) > 0 {
			errStr := addResult.Error.Error()
			if contains(errStr, "duplicate column name") {
				return nil // 列已存在，忽略错误
			}
		}
		return addResult.Error
	}

	return nil
}

func (m *AddVideoFileOwnerMigration) Down(db *gorm.DB) error {
	// SQLite 不支持 DROP COLUMN，需要重建表
	// 这里简单处理：不执行回滚
	return nil
}
```

**Table creation pattern** (lines 108-134):
```go
// 第二部分：创建或验证 task_huawei_configs 表
var count int64
db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='task_huawei_configs'").Scan(&count)

if count == 0 {
	// 表不存在，直接创建
	err := db.Exec(`
		CREATE TABLE task_huawei_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER NOT NULL,
			huawei_config_id INTEGER NOT NULL,
			config_type VARCHAR(20) NOT NULL,
			created_at DATETIME,
			UNIQUE(task_id, huawei_config_id),
			FOREIGN KEY(task_id) REFERENCES video_recording_tasks(id) ON DELETE CASCADE,
			FOREIGN KEY(huawei_config_id) REFERENCES huawei_configs(id) ON DELETE CASCADE
		)
	`).Error
	if err != nil {
		return fmt.Errorf("failed to create task_huawei_configs table: %w", err)
	}
}

// 确保索引存在（幂等操作）
db.Exec("CREATE INDEX IF NOT EXISTS idx_task_huawei_config ON task_huawei_configs(task_id, huawei_config_id)")
```

---

### `frontend/src/api/input-config.ts` (api-client, request-response)

**Analog:** `frontend/src/api/huawei-config.ts`

**Imports pattern** (lines 1-14):
```typescript
// 华为配置管理 API 客户端

import type {
  HuaweiConfig,
  HuaweiConfigListParams,
  HuaweiConfigListResponse,
  CreateHuaweiConfigRequest,
  UpdateHuaweiConfigRequest,
  USBDevicesScanResult,
  TestStreamRequest,
} from '../types/huawei-config'
import type { ApiResponse } from '../types/auth'
import { apiRequest } from './apiClient'
```

**API function pattern** (lines 16-28):
```typescript
// 获取配置列表
export async function getHuaweiConfigList(
  params: HuaweiConfigListParams
): Promise<ApiResponse<HuaweiConfigListResponse>> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.keyword) queryParams.append('keyword', params.keyword)
  if (params.is_active !== undefined) queryParams.append('is_active', params.is_active.toString())

  const query = queryParams.toString()
  return apiRequest(`/api/v1/huawei-configs${query ? `?${query}` : ''}`)
}
```

**Create/Update pattern** (lines 38-65):
```typescript
// 创建配置
export async function createHuaweiConfig(
  req: CreateHuaweiConfigRequest
): Promise<ApiResponse<HuaweiConfig>> {
  return apiRequest('/api/v1/huawei-configs', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 更新配置
export async function updateHuaweiConfig(
  id: number,
  req: UpdateHuaweiConfigRequest
): Promise<ApiResponse<HuaweiConfig>> {
  return apiRequest(`/api/v1/huawei-configs/${id}`, {
    method: 'PUT',
    body: JSON.stringify(req),
  })
}
```

**USB scanning pattern** (lines 72-90):
```typescript
// 扫描USB设备
export async function scanUSBDevices(): Promise<ApiResponse<USBDevicesScanResult>> {
  return apiRequest('/api/v1/huawei-configs/scan-devices')
}

// 测试流媒体连接
export async function testStream(
  req: TestStreamRequest
): Promise<ApiResponse<void>> {
  return apiRequest('/api/v1/huawei-configs/test-stream', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}
```

---

### `frontend/src/types/input-config.ts` (types, request-response)

**Analog:** `frontend/src/types/huawei-config.ts`

**Type definitions pattern** (lines 1-51):
```typescript
// USB设备信息
export interface USBDeviceInfo {
  type: string // "camera" | "audio"
  name: string // 设备名称
  device_id: string // 设备ID (/dev/video0, hw:1,0)
  status: string // "available" | "in_use" | "error"
  backend: string // "v4l2" | "alsa"
}

export interface HuaweiConfig {
  id: number
  name: string
  description: string
  server: string
  port: number
  username: string
  // ... other fields
  is_active: boolean
  is_locked: boolean
  locked_by_task_id: number | null
  created_at: string
  updated_at: string
  video_recording_tasks?: Array<{
    id: number
    name: string
    status: string
  }>
}
```

**Request/Response types pattern** (lines 53-121):
```typescript
export interface HuaweiConfigListParams {
  page?: number
  page_size?: number
  keyword?: string
  is_active?: boolean
}

export interface HuaweiConfigListResponse {
  total: number
  items: HuaweiConfig[]
}

export interface CreateHuaweiConfigRequest {
  name: string
  description?: string
  server: string
  port: number
  username: string
  password: string
  // ... other optional fields
  stream_protocol?: 'rtmp' | 'rtsp' | 'srt' | 'hls'
  stream_url?: string
  stream_username?: string
  stream_password?: string
  stream_enabled?: boolean
}

export interface UpdateHuaweiConfigRequest {
  name?: string
  description?: string
  server?: string
  // ... other optional fields with same structure as Create
}
```

---

### `frontend/src/pages/system/input-configs/index.tsx` (component, request-response)

**Analog:** `frontend/src/pages/system/huawei-configs/index.tsx`

**Component structure pattern** (lines 1-98):
```typescript
// 华为配置管理页面

import { useState, useEffect } from 'react'
import {
  Table,
  Button,
  Space,
  Input,
  Modal,
  Form,
  message,
  Popconfirm,
  Tag,
  // ... other imports
} from 'antd'
import {
  PlusOutlined,
  SearchOutlined,
  // ... other icon imports
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import * as huaweiConfigApi from '../../../api/huawei-config'
import type {
  HuaweiConfig,
  HuaweiConfigListParams,
  CreateHuaweiConfigRequest,
  UpdateHuaweiConfigRequest,
  USBDeviceInfo,
  TestStreamRequest,
} from '../../../types/huawei-config'

export default function HuaweiConfigManagement() {
  const [configs, setConfigs] = useState<HuaweiConfig[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingConfig, setEditingConfig] = useState<HuaweiConfig | null>(null)
  const [form] = Form.useForm()

  const [params, setParams] = useState<HuaweiConfigListParams>({
    page: 1,
    page_size: 20,
  })
```

**Data loading pattern** (lines 64-93):
```typescript
const loadConfigs = async () => {
  setLoading(true)
  try {
    const response = await huaweiConfigApi.getHuaweiConfigList(params)
    if (response.data) {
      setConfigs(response.data.items)
      setTotal(response.data.total)
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '加载配置列表失败')
  } finally {
    setLoading(false)
  }
}

useEffect(() => {
  loadConfigs()
}, [params])

const handleSearch = (value: string) => {
  setParams({ ...params, keyword: value, page: 1 })
}
```

**Form submission pattern** (lines 136-218):
```typescript
const handleSubmit = async () => {
  try {
    const values = await form.validateFields()

    if (editingConfig) {
      // 编辑模式：密码为空则不更新密码
      const req: UpdateHuaweiConfigRequest = {
        name: values.name,
        description: values.description,
        server: values.server,
        // ... map other fields
      }
      // 只有在密码字段有值时才更新密码
      if (values.password && values.password.trim() !== '') {
        req.password = values.password
      }
      await huaweiConfigApi.updateHuaweiConfig(editingConfig.id, req)
      message.success('更新成功')
    } else {
      // 新建模式：密码必填
      const req: CreateHuaweiConfigRequest = {
        name: values.name,
        description: values.description,
        server: values.server,
        // ... map other fields
      }
      await huaweiConfigApi.createHuaweiConfig(req)
      message.success('创建成功')
    }

    closeModal()
    loadConfigs()
  } catch (error) {
    // 改进错误提示，显示具体的验证错误信息
    const err = error as Error & { errorFields?: Array<{ name?: string[]; errors?: string[] }> }
    if (err.errorFields) {
      const firstError = err.errorFields[0]
      const fieldName = firstError?.name?.[0] || '字段'
      const errorMessage = firstError?.errors?.[0] || '验证失败'
      message.error(`${fieldName}: ${errorMessage}`)
    } else if (err.message) {
      message.error(err.message)
    } else {
      message.error('操作失败，请检查表单填写是否正确')
    }
  }
}
```

**USB device scanning pattern** (lines 236-291):
```typescript
// 扫描USB设备
const handleScanDevices = async () => {
  setScanningDevices(true)
  try {
    const response = await huaweiConfigApi.scanUSBDevices()
    if (response.data) {
      setDetectedCameras(response.data.cameras || [])
      setDetectedAudios(response.data.audios || [])
      message.success(`检测到 ${(response.data.cameras?.length || 0)} 个摄像头，${(response.data.audios?.length || 0)} 个音频设备`)
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '扫描USB设备失败')
  } finally {
    setScanningDevices(false)
  }
}

// 选择摄像头
const handleSelectCamera = (device: USBDeviceInfo) => {
  let deviceIndex = device.device_id
  if (device.backend === 'dshow' && device.device_id.startsWith('video=')) {
    deviceIndex = device.device_id.replace('video=', '')
  } else if (device.backend === 'v4l2' && device.device_id.startsWith('/dev/video')) {
    deviceIndex = device.device_id.replace('/dev/', '')
  }

  form.setFieldsValue({
    usb_camera_name: device.name,
    usb_camera_device: deviceIndex,
  })
  message.info(`已选择摄像头: ${device.name}`)
}
```

**Stream testing pattern** (lines 294-335):
```typescript
// 测试流媒体连接
const handleTestStream = async () => {
  try {
    const values = await form.validateFields(['stream_protocol', 'stream_url'])
    const req: TestStreamRequest = {
      protocol: values.stream_protocol,
      url: values.stream_url,
      username: values.stream_username,
      password: values.stream_password,
    }
    await huaweiConfigApi.testStream(req)
    message.success('流媒体连接测试成功')
  } catch (error: any) {
    if (error?.errorFields) {
      message.error('请先填写协议和URL')
    } else {
      message.error(error instanceof Error ? error.message : '连接测试失败')
    }
  }
}
```

**Tabbed form pattern** (lines 476-730):
```typescript
<Tabs
  items={[
    {
      key: 'basic',
      label: '基本配置',
      children: (
        <>
          <Form.Item
            name="name"
            label="配置名称"
            rules={[
              { required: true, message: '请输入配置名称' },
              { max: 100, message: '配置名称最多100个字符' },
            ]}
          >
            <Input placeholder="请输入配置名称" />
          </Form.Item>
          {/* ... other basic fields */}
        </>
      ),
    },
    {
      key: 'usb',
      label: 'USB设备',
      children: (
        <>
          <Button onClick={handleScanDevices}>自动检测USB设备</Button>
          {/* USB device selection UI */}
        </>
      ),
    },
    {
      key: 'stream',
      label: '流媒体配置',
      children: (
        <>
          <Form.Item name="stream_protocol" label="流媒体协议">
            <Select options={[...]} />
          </Form.Item>
          {/* ... other stream fields */}
        </>
      ),
    },
  ]}
/>
```

---

### `internal/scheduler/video_scheduler.go` (modified, event-driven)

**Analog:** Self-modification - add config type dispatch logic

**Pattern:** Add switch statement to handle different config types when starting recording tasks. Based on existing pattern in RESEARCH.md (lines 327-348).

---

### `internal/models/video_recording_task.go` (modified, CRUD)

**Analog:** Self-modification - remove HuaweiConfigID required validation

**Pattern:** Remove the required field validation in `IsValid()` method that enforces `HuaweiConfigID` must be set.

---

### `cmd/server/app.go` (modified, request-response)

**Analog:** Self-modification - add new routes and handler registration

**Route registration pattern** (lines 792-810):
```go
// 华为配置管理
huaweiConfigs := api.Group("/huawei-configs")
{
	huaweiConfigs.GET("/scan-devices", a.handlers.HuaweiConfig.ScanUSBDevices)
	huaweiConfigs.GET("/recommended-device", a.handlers.HuaweiConfig.GetRecommendedDevice)
	huaweiConfigs.GET("", a.handlers.HuaweiConfig.ListConfigs)
	huaweiConfigs.GET("/active", a.handlers.HuaweiConfig.GetActiveConfigs)
	huaweiConfigs.GET("/:id", a.handlers.HuaweiConfig.GetConfig)
	huaweiConfigs.POST("", a.handlers.HuaweiConfig.CreateConfig)
	huaweiConfigs.PUT("/:id", a.handlers.HuaweiConfig.UpdateConfig)
	huaweiConfigs.DELETE("/:id", a.handlers.HuaweiConfig.DeleteConfig)
}
```

**Pattern:** Add new `/api/v1/input-configs` route group with same structure, and add backward compatibility redirect for old `/api/v1/huawei-configs` routes.

---

### `frontend/src/layouts/BasicLayout.tsx` (modified, routing)

**Analog:** Self-modification - update menu items

**Pattern:** Update menu configuration to change "华为配置" to "输入配置" and update route from `/system/huawei-configs` to `/system/input-configs`.

---

## Shared Patterns

### Authentication Middleware
**Source:** `cmd/server/app.go` (lines 697-719)
**Apply to:** All new API route groups
```go
// 需要认证的路由
authenticated := a.router.Group("/api/v1/auth")
authenticated.Use(middleware.SM4Auth(a.tokenService))

// API路由组（支持SM4 Token和API Key认证）
api := a.router.Group("/api/v1")
api.Use(middleware.MultiAuth(a.db, a.tokenService, a.logger))
```

### Response Formatting
**Source:** `internal/handlers/huawei_config_handler.go` (lines 46-47, 61-66)
**Apply to:** All handler methods
```go
// Error response
response.GinError(c, response.CodeInvalidRequest, "请求参数错误")

// Success response
response.GinSuccess(c, result)
```

### Validation Pattern
**Source:** `internal/models/huawei_config.go` (lines 99-123)
**Apply to:** All model validation methods
```go
func (h *HuaweiConfig) Validate() error {
	var errs []string

	if h.Name == "" {
		errs = append(errs, "配置名称不能为空")
	}

	if len(errs) > 0 {
		return fmt.Errorf("验证失败: %s", errs)
	}
	return nil
}
```

### Error Handling Pattern
**Source:** `internal/services/huawei_config_service.go` (lines 154-206)
**Apply to:** All service methods
```go
if err := s.validateBackendConfig(config); err != nil {
	return nil, err
}

if err := s.db.Create(config).Error; err != nil {
	return nil, err
}

s.logger.Info("华为配置已创建",
	zap.Uint("config_id", config.ID),
	zap.String("name", config.Name),
)

return config, nil
```

### Frontend Form Validation
**Source:** `frontend/src/pages/system/huawei-configs/index.tsx` (lines 483-492)
**Apply to:** All form items
```typescript
<Form.Item
  name="name"
  label="配置名称"
  rules={[
    { required: true, message: '请输入配置名称' },
    { max: 100, message: '配置名称最多100个字符' },
  ]}
>
  <Input placeholder="请输入配置名称" />
</Form.Item>
```

### Database Migration Idempotency
**Source:** `internal/migrations/001_add_video_file_owner.go` (lines 16-27)
**Apply to:** All new migrations
```go
// 使用 PRAGMA table_info 检查列是否已存在（更可靠）
var columnName string
checkErr := db.Raw("SELECT name FROM pragma_table_info('video_files') WHERE name = 'created_by'").Scan(&columnName).Error

// 如果查询成功且找到了列，说明已存在，跳过迁移
if checkErr == nil && columnName != "" {
	return nil
}
```

### USB Device Selection Pattern
**Source:** `frontend/src/pages/system/huawei-configs/index.tsx` (lines 582-600)
**Apply to:** Input config USB device selection
```typescript
{detectedCameras.length > 0 && (
  <Form.Item label="检测到的摄像头">
    <Select
      placeholder="选择检测到的摄像头设备"
      onChange={(value) => {
        const device = detectedCameras.find(d => d.device_id === value)
        if (device) handleSelectCamera(device)
      }}
      options={detectedCameras.map(device => ({
        label: (
          <Space>
            <VideoCameraOutlined />
            {device.name}
            <Tag color={device.status === 'available' ? 'green' : 'orange'}>{device.status}</Tag>
          </Space>
        ),
        value: device.device_id,
      }))}
    />
  </Form.Item>
)}
```

---

## No Analog Found

**None** - All files have close analogs in the existing codebase. The refactoring is primarily renaming and restructuring existing patterns, not introducing completely new functionality.

---

## Metadata

**Analog search scope:**
- `internal/models/` - Database models
- `internal/services/` - Business logic services
- `internal/handlers/` - API handlers
- `internal/migrations/` - Database migrations
- `frontend/src/api/` - API client functions
- `frontend/src/types/` - TypeScript type definitions
- `frontend/src/pages/system/` - Page components
- `cmd/server/` - Application setup and routing

**Files scanned:** 12
**Pattern extraction date:** 2026-04-29

**Key insights:**
1. All new files follow exact patterns from existing HuaweiConfig implementation
2. The refactoring is primarily renaming (huawei → input) and adding config_type field
3. Frontend patterns are well-established with tabbed forms and device scanning UI
4. Migration pattern ensures idempotency for SQLite ALTER TABLE operations
5. Authentication and response formatting are consistent across all handlers
