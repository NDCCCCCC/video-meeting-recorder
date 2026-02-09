# NVR架构与实现

## 一、架构概述

### 1.1 分层架构

NVR模块遵循严格的分层架构，每层有明确的职责：

```
┌─────────────────────────────────────────────────────────────────┐
│                       Interfaces Layer                          │
│                    HTTP API / WebSocket                        │
│                    (handlers/nvr.go - 560行)                     │
└─────────────────────────────────────────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Application Layer                           │
│                    CQRS - Command/Query Separation               │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Command Handlers (nvr_commands.go - 520行)               │   │
│  │  - CreateNVR                                              │   │
│  │  - ConnectNVR                                             │   │
│  │  - AddCamera                                              │   │
│  │  - StartMotionRecording                                   │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Query Handlers (nvr_queries.go - 370行)                 │   │
│  │  - GetNVR                                                 │   │
│  │  - ListNVRs                                               │   │
│  │  - GetCameras                                              │   │
│  │  - GetRecordings                                           │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Motion Detection Service                                 │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Domain Layer                               │
│                    DDD - Aggregates & Value Objects                │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ NVR Aggregate (aggregate.go - 671行)                     │   │
│  │  - NVR聚合根                                              │   │
│  │  - Camera实体                                              │   │
│  │  - CameraList值对象                                        │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Value Objects (value_objects.go - 488行)                  │   │
│  │  - NVRID, IPAddress, StreamURL                            │   │
│  │  - RecordingConfig, Resolution                           │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Domain Events (events.go - 524行)                         │   │
│  │  - 19种领域事件                                           │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Infrastructure Layer                          │
│              External Systems & Persistence                       │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ NVR Repository (nvr_repository_impl.go - 388行)           │   │
│  │  - GORM持久化                                              │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Storage Manager (storage/manager.go - 320行)               │   │
│  │  - 磁盘空间管理                                            │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ ONVIF Adapter (adapters/onvif/)                            │   │
│  │  - ONVIF协议实现                                           │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 代码统计

| 层 | 文件 | 行数 | 说明 |
|----|------|------|------|
| **Domain Layer** | | | |
| | aggregate.go | 671 | NVR聚合根和Camera实体 |
| | value_objects.go | 488 | 值对象定义 |
| | events.go | 524 | 领域事件定义 |
| | errors.go | 92 | 领域错误定义 |
| | repository.go | 81 | 仓储接口 |
| | aggregate_test.go | 467 | 单元测试 |
| **小计** | 6个文件 | 2323行 | |
| **Application Layer** | | | |
| | nvr_commands.go | 520 | 命令处理器 |
| | nvr_queries.go | 370 | 查询处理器 |
| | providers.go | 90 | Wire依赖注入 |
| **小计** | 3个文件 | 980行 | |
| **Infrastructure Layer** | | | |
| | nvr_repository_impl.go | 388 | NVR仓储实现 |
| | models/nvr.go | 122 | 数据库模型 |
| | storage/manager.go | 320 | 存储管理器 |
| **小计** | 3个文件 | 830行 | |
| **Interfaces Layer** | | | |
| | handlers/nvr.go | 560 | HTTP处理器 |
| | router.go | 80 | 路由配置 |
| **小计** | 2个文件 | 640行 | |
| **总计** | 14个文件 | 4773行 | 核心业务代码 |

## 二、应用层实现

### 2.1 Command Handler基类

```go
// internal/application/nvr/command/handler.go
type CommandHandler struct {
    repository nvr.Repository
    eventBus   events.EventBus
    logger     *zap.Logger
}

func NewCommandHandler(
    repo nvr.Repository,
    eventBus events.EventBus,
    logger *zap.Logger,
) *CommandHandler {
    return &CommandHandler{
        repository: repo,
        eventBus:   eventBus,
        logger:     logger,
    }
}

// execute 通用执行模式
func (h *CommandHandler) execute(
    ctx context.Context,
    aggregateName string,
    operation string,
    fn func() error,
) error {
    h.logger.Info("开始执行命令",
        zap.String("aggregate", aggregateName),
        zap.String("operation", operation),
    )

    start := time.Now()

    err := fn()

    duration := time.Since(start)

    if err != nil {
        h.logger.Error("命令执行失败",
            zap.String("aggregate", aggregateName),
            zap.String("operation", operation),
            zap.Duration("duration", duration),
            zap.Error(err),
        )
        return err
    }

    h.logger.Info("命令执行成功",
        zap.String("aggregate", aggregateName),
        zap.String("operation", operation),
        zap.Duration("duration", duration),
    )

    return nil
}
```

### 2.2 NVR Commands

```go
// internal/application/nvr/command/nvr_commands.go
type NVRCommands struct {
    *CommandHandler
}

// CreateNVRCommand 创建NVR命令
type CreateNVRCommand struct {
    NVRID     string
    Name      string
    IPAddress string
}

func (c *NVRCommands) Handle(ctx context.Context, cmd *CreateNVRCommand) (*nvr.NVR, error) {
    var result *nvr.NVR
    var err error

    err = c.execute(ctx, "NVR", "CreateNVR", func() error {
        // 1. 验证并创建值对象
        nvrID, err := nvr.NewNVRID(cmd.NVRID)
        if err != nil {
            return err
        }

        ipAddress, err := nvr.NewIPAddress(cmd.IPAddress)
        if err != nil {
            return err
        }

        // 2. 创建聚合根
        aggregate, err := nvr.NewNVR(nvrID, cmd.Name, ipAddress)
        if err != nil {
            return err
        }

        // 3. 保存
        if err := c.repository.Save(ctx, aggregate); err != nil {
            return fmt.Errorf("保存NVR失败: %w", err)
        }

        // 4. 发布事件
        c.publishEvents(ctx, aggregate)

        result = aggregate
        return nil
    })

    return result, err
}

// ConnectNVRCommand 连接NVR命令
type ConnectNVRCommand struct {
    NVRID string
}

func (c *NVRCommands) Handle(ctx context.Context, cmd *ConnectNVRCommand) error {
    return c.execute(ctx, "NVR", "ConnectNVR", func() error {
        // 1. 加载聚合根
        nvrID, _ := nvr.NewNVRID(cmd.NVRID)
        aggregate, err := c.repository.FindByID(ctx, nvrID)
        if err != nil {
            return err
        }

        // 2. 执行业务操作
        if err := aggregate.Connect(ctx); err != nil {
            return err
        }

        // 3. 保存
        if err := c.repository.Save(ctx, aggregate); err != nil {
            return fmt.Errorf("保存NVR失败: %w", err)
        }

        // 4. 发布事件
        c.publishEvents(ctx, aggregate)

        return nil
    })
}

// AddCameraCommand 添加摄像头命令
type AddCameraCommand struct {
    NVRID            string
    CameraID          string
    Name              string
    StreamURL         string
    RecordingConfig   RecordingConfigDTO
}

func (c *NVRCommands) Handle(ctx context.Context, cmd *AddCameraCommand) error {
    return c.execute(ctx, "NVR", "AddCamera", func() error {
        // 1. 加载聚合根
        nvrID, _ := nvr.NewNVRID(cmd.NVRID)
        aggregate, err := c.repository.FindByID(ctx, nvrID)
        if err != nil {
            return err
        }

        // 2. 验证并创建值对象
        cameraID, err := nvr.NewCameraID(cmd.CameraID)
        if err != nil {
            return err
        }

        streamURL, err := nvr.NewStreamURL(cmd.StreamURL)
        if err != nil {
            return err
        }

        config := cmd.RecordingConfig.ToDomainModel()

        // 3. 执行业务操作
        if err := aggregate.AddCamera(cameraID, cmd.Name, streamURL, config); err != nil {
            return err
        }

        // 4. 保存
        if err := c.repository.Save(ctx, aggregate); err != nil {
            return fmt.Errorf("保存NVR失败: %w", err)
        }

        // 5. 发布事件
        c.publishEvents(ctx, aggregate)

        return nil
    })
}

// StartMotionRecordingCommand 启动运动检测录制命令
type StartMotionRecordingCommand struct {
    NVRID    string
    CameraID string
}

func (c *NVRCommands) Handle(ctx context.Context, cmd *StartMotionRecordingCommand) error {
    return c.execute(ctx, "NVR", "StartMotionRecording", func() error {
        // 1. 加载聚合根
        nvrID, _ := nvr.NewNVRID(cmd.NVRID)
        aggregate, err := c.repository.FindByID(ctx, nvrID)
        if err != nil {
            return err
        }

        // 2. 验证摄像头ID
        cameraID, _ := nvr.NewCameraID(cmd.CameraID)

        // 3. 执行业务操作
        if err := aggregate.StartMotionRecording(ctx, cameraID); err != nil {
            return err
        }

        // 4. 保存
        if err := c.repository.Save(ctx, aggregate); err != nil {
            return fmt.Errorf("保存NVR失败: %w", err)
        }

        // 5. 发布事件
        c.publishEvents(ctx, aggregate)

        return nil
    })
}

// publishEvents 发布领域事件
func (c *CommandHandler) publishEvents(ctx context.Context, aggregate *nvr.NVR) {
    events := aggregate.GetEvents()
    for _, event := range events {
        c.eventBus.Publish(ctx, event)
    }
}
```

### 2.3 Query Handlers

```go
// internal/application/nvr/query/nvr_queries.go
type NVRQueries struct {
    repository nvr.Repository
    cache     *cache.Cache
    logger    *zap.Logger
}

// GetNVR 获取NVR详情
func (q *NVRQueries) GetNVR(ctx context.Context, nvrID string) (*nvr.NVR, error) {
    id, err := nvr.NewNVRID(nvrID)
    if err != nil {
        return nil, err
    }

    return q.repository.FindByID(ctx, id)
}

// ListNVRs 获取NVR列表
func (q *NVRQueries) ListNVRs(ctx context.Context) ([]*nvr.NVR, error) {
    return q.repository.FindAll(ctx)
}

// GetCameras 获取摄像头列表
func (q *NVRQueries) GetCameras(ctx context.Context, nvrID string) ([]*nvr.Camera, error) {
    id, err := nvr.NewNVRID(nvrID)
    if err != nil {
        return nil, err
    }

    nvr, err := q.repository.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }

    cameras := nvr.Cameras().List()
    result := make([]*nvr.Camera, len(cameras))
    for i, cam := range cameras {
        result[i] = &cam
    }

    return result, nil
}

// GetRecordings 获取录像列表
func (q *NVRQueries) GetRecordings(ctx context.Context, nvrID string) ([]*models.NVRRecording, error) {
    // 直接从数据库查询
    var recordings []*models.NVRRecording
    err := q.repository.(*persistence.NVRRepositoryImpl).db.
        Where("nvr_id = ?", nvrID).
        Order("started_at DESC").
        Find(&recordings).Error

    return recordings, err
}
```

## 三、基础设施层实现

### 3.1 NVR Repository

```go
// internal/infrastructure/persistence/nvr_repository_impl.go
type NVRRepositoryImpl struct {
    db *gorm.DB
}

func NewNVRRepository(db *gorm.DB) nvr.Repository {
    return &NVRRepositoryImpl{db: db}
}

// Save 保存NVR聚合根
func (r *NVRRepositoryImpl) Save(ctx context.Context, nvr *nvr.NVR) error {
    // 1. 转换为数据模型
    model := &models.NVRDevice{
        NVRID:              nvr.ID().Value(),
        Name:                nvr.Name(),
        IPAddress:           nvr.IPAddress().Value(),
        Status:              string(nvr.Status()),
        LinkedConferenceID:  nvr.LinkedConference(),
        Version:             nvr.Version(),
    }

    // 2. 使用事务保存
    err := r.db.Transaction(func(tx *gorm.DB) error {
        // 保存或更新NVR
        tx.Where("nvr_id = ?", model.NVRID).
            Assign(model).
            FirstOrCreate(model)

        // 处理摄像头关联
        if err := r.saveCameras(tx, nvr, model); err != nil {
            return err
        }

        return nil
    })

    return err
}

// saveCameras 保存摄像头
func (r *NVRRepositoryImpl) saveCameras(tx *gorm.DB, nvr *nvr.NVR, model *models.NVRDevice) error {
    // 1. 获取现有摄像头
    var existingCameras []models.NVRCamera
    tx.Where("nvr_id = ?", model.NVRID).Find(&existingCameras)

    existingMap := make(map[string]models.NVRCamera)
    for _, cam := range existingCameras {
        existingMap[cam.CameraID] = cam
    }

    // 2. 遍历聚合根中的摄像头
    for _, camera := range nvr.Cameras().List() {
        cameraModel := &models.NVRCamera{
            CameraID:              camera.ID().Value(),
            NVRID:                model.NVRID,
            Name:                  camera.Name(),
            StreamURL:             camera.StreamURL().Value(),
            Status:                string(camera.Status()),
            Resolution:            camera.Config().Resolution.String(),
            FPS:                   camera.Config().FPS,
            MotionDetectionEnabled: camera.Config().MotionDetection.Enabled,
        }

        if existing, exists := existingMap[cameraModel.CameraID]; exists {
            // 更新
            tx.Model(&existing).Updates(cameraModel)
        } else {
            // 新增
            tx.Create(cameraModel)
        }
    }

    // 3. 删除不再存在的摄像头
    for _, existing := range existingCameras {
        found := false
        for _, camera := range nvr.Cameras().List() {
            if camera.ID().Value() == existing.CameraID {
                found = true
                break
            }
        }
        if !found {
            tx.Delete(&existing)
        }
    }

    return nil
}

// FindByID 根据ID查找NVR
func (r *NVRRepositoryImpl) FindByID(ctx context.Context, id nvr.NVRID) (*nvr.NVR, error) {
    var model models.NVRDevice
    err := r.db.Where("nvr_id = ?", id.Value()).First(&model).Error
    if err != nil {
        return nil, err
    }

    return r.modelToAggregate(&model)
}

// modelToAggregate 数据模型转聚合根
func (r *NVRRepositoryImpl) modelToAggregate(model *models.NVRDevice) (*nvr.NVR, error) {
    nvrID, _ := nvr.NewNVRID(model.NVRID)
    ipAddress, _ := nvr.NewIPAddress(model.IPAddress)

    aggregate := &nvr.NVR{
        id:               nvrID,
        name:             model.Name,
        ipAddress:       ipAddress,
        status:           nvr.NVRStatus(model.Status),
        linkedConference: model.LinkedConferenceID,
        createdAt:       model.CreatedAt,
        updatedAt:       model.UpdatedAt,
        version:          model.Version,
    }

    // 加载摄像头
    var cameras []models.NVRCamera
    r.db.Where("nvr_id = ?", model.NVRID).Find(&cameras)

    cameraList := nvr.NewCameraList()
    for _, cam := range cameras {
        cameraID, _ := nvr.NewCameraID(cam.CameraID)
        streamURL, _ := nvr.NewStreamURL(cam.StreamURL)

        camera := nvr.NewCamera(
            cameraID,
            cam.Name,
            streamURL,
            nvr.RecordingConfig{
                Resolution: nvr.Resolution{Width: 1920, Height: 1080}, // 简化处理
                FPS:            cam.FPS,
                VideoBitrate:    "3000k",
                AudioBitrate:    "128k",
                SegmentDuration: 5 * time.Minute,
                MotionDetection: nvr.MotionDetectionConfig{
                    Enabled:           cam.MotionDetectionEnabled,
                    Sensitivity:       "medium",
                    PreRecordSeconds:  3,
                    PostRecordSeconds: 5,
                },
            },
        )
        cameraList.Add(*camera)
    }

    aggregate.cameras = cameraList

    return aggregate, nil
}
```

### 3.2 Storage Manager

```go
// internal/infrastructure/storage/manager.go
type StorageManager struct {
    db        *gorm.DB
    config    *config.NVRConfig
    logger    *zap.Logger
    ticker    *time.Ticker
}

// Start 启动存储管理器
func (m *StorageManager) Start() error {
    m.ticker = time.NewTicker(m.config.Storage.DiskManagement.CheckInterval)
    go m.monitorLoop()
    return nil
}

// monitorLoop 监控循环
func (m *StorageManager) monitorLoop() {
    for range m.ticker.C {
        if err := m.checkDiskSpace(); err != nil {
            m.logger.Error("磁盘空间检查失败", zap.Error(err))
        }
    }
}

// checkDiskSpace 检查磁盘空间
func (m *StorageManager) checkDiskSpace() error {
    var stat syscall.Statfs_t
    if err := syscall.Statfs(m.config.Storage.BasePath, &stat); err != nil {
        return err
    }

    // 计算可用空间
    available := stat.Bavail * uint64(stat.Bsize)
    minRequired := m.config.Storage.DiskManagement.MinFreeSpace

    if available < minRequired {
        m.logger.Warn("磁盘空间不足，开始清理",
            zap.Uint64("available", available),
            zap.Uint64("min_required", minRequired),
        )

        return m.cleanupOldRecordings()
    }

    return nil
}

// cleanupOldRecordings 清理旧录像
func (m *StorageManager) cleanupOldRecordings() error {
    // 根据策略删除旧录像
    switch m.config.Storage.DeletionStrategy {
    case "smart":
        return m.smartCleanup()
    case "oldest_first":
        return m.oldestFirstCleanup()
    default:
        return m.oldestFirstCleanup()
}

// smartCleanup 智能清理
func (m *StorageManager) smartCleanup() error {
    // 1. 删除与会议无关的旧录像
    // 2. 删除超过保留期限的录像
    // 3. 尊保护标志的录像

    retentionDate := time.Now().AddDate(0, 0, -m.config.Storage.Retention.DefaultDays)

    result := m.db.Where(`
        (linked_conference_id IS NULL OR stopped_at < ?)
        AND stopped_at < ?
    `, retentionDate).Delete(&models.NVRRecording{}).Error

    return result
}
```

## 四、接口层实现

### 4.1 NVR Handler

```go
// internal/interfaces/http/handlers/nvr.go
type NVRHandler struct {
    commands *application.NVRCommands
    queries  *application.NVRQueries
    logger   *zap.Logger
}

// CreateNVR 创建NVR
func (h *NVRHandler) CreateNVR(c *gin.Context) {
    var req command.CreateNVRCommand
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    nvr, err := h.commands.Handle(c.Request.Context(), &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, nvr.ToDTO())
}

// ListNVRs 获取NVR列表
func (h *NVRHandler) ListNVRs(c *gin.Context) {
    nvrs, err := h.queries.ListNVRs(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    dtos := make([]*dto.NVRDTO, len(nvrs))
    for i, nvr := range nvrs {
        dtos[i] = nvr.ToDTO()
    }

    c.JSON(http.StatusOK, gin.H{"nvrs": dtos})
}

// AddCamera 添加摄像头
func (h *NVRHandler) AddCamera(c *gin.Context) {
    nvrID := c.Param("id")

    var req command.AddCameraCommand
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    req.NVRID = nvrID

    if err := h.commands.Handle(c.Request.Context(), &req); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"message": "摄像头添加成功"})
}

// StartMotionRecording 启动运动检测录制
func (h *NVRHandler) StartMotionRecording(c *gin.Context) {
    nvrID := c.Param("id")
    cameraID := c.Param("cameraId")

    cmd := &command.StartMotionRecordingCommand{
        NVRID:    nvrID,
        CameraID: cameraID,
    }

    if err := h.commands.Handle(c.Request.Context(), cmd); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "录制已启动"})
}
```

## 五、依赖注入

### 5.1 Wire Provider Set

```go
// internal/application/nvr/providers.go
var NVRProviderSet = wire.NewSet(
    // 基础设施层
    ProvideNVRRepository,
    ProvideStorageManager,

    // 应用层
    ProvideNVRCommands,
    ProvideNVRQueries,

    // 接口层
    ProvideNVRHandler,
)

func ProvideNVRRepository(db *gorm.DB) nvr.Repository {
    return persistence.NewNVRRepository(db)
}

func ProvideStorageManager(db *gorm.DB, config *config.Config, logger *zap.Logger) *StorageManager {
    return storage.NewManager(db, config, logger)
}

func ProvideNVRCommands(
    repo nvr.Repository,
    eventBus events.EventBus,
    logger *zap.Logger,
) *application.NVRCommands {
    return application.NewNVRCommands(repo, eventBus, logger)
}

func ProvideNVRQueries(
    repo nvr.Repository,
    cache *cache.Cache,
    logger *zap.Logger,
) *application.NVRQueries {
    return application.NewNVRQueries(repo, cache, logger)
}

func ProvideNVRHandler(
    commands *application.NVRCommands,
    queries *application.NVRQueries,
    logger *zap.Logger,
) *handlers.NVRHandler {
    return handlers.NewNVRHandler(commands, queries, logger)
}
```

## 六、前端实现

### 6.1 NVR API Service

```typescript
// frontend/src/services/api/nvr.service.ts
export class NVRService {
  private baseUrl = '/api/v1/nvr';

  async createNVR(data: CreateNVRRequest): Promise<NVRDTO> {
    const response = await axios.post<NVRDTO>(this.baseUrl, data);
    return response.data;
  }

  async listNVRs(): Promise<NVRDTO[]> {
    const response = await axios.get<{ nvrs: NVRDTO[] }>(this.baseUrl);
    return response.data.nvrs;
  }

  async getNVR(id: string): Promise<NVRDTO> {
    const response = await axios.get<NVRDTO>(`${this.baseUrl}/${id}`);
    return response.data;
  }

  async connectNVR(id: string): Promise<void> {
    await axios.post(`${this.baseUrl}/${id}/connect`);
  }

  async disconnectNVR(id: string): Promise<void> {
    await axios.post(`${this.baseUrl}/${id}/disconnect`);
  }

  async addCameras(nvrId: string, data: AddCameraRequest): Promise<void> {
    await axios.post(`${this.baseUrl}/${nvrId}/cameras`, data);
  }

  async getCameras(nvrId: string): Promise<CameraDTO[]> {
    const response = await axios.get<{ cameras: CameraDTO[] }>(`${this.baseUrl}/${nvrId}/cameras`);
    return response.data.cameras;
  }

  async startMotionRecording(nvrId: string, cameraId: string): Promise<void> {
    await axios.post(`${this.baseUrl}/${nvrId}/cameras/${cameraId}/motion/start`);
  }

  async stopMotionRecording(nvrId: string, cameraId: string): Promise<void> {
    await axios.post(`${this.baseUrl}/${nvrId}/cameras/${cameraId}/motion/stop`);
  }
}
```

### 6.2 React Hooks

```typescript
// frontend/src/hooks/useNVR.ts
export function useNVRList() {
  const [nvrs, setNVRs] = useState<NVRDTO[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchNVRs = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const service = new NVRService();
      const data = await service.listNVRs();
      setNVRs(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchNVRs();
  }, []);

  return { nvrs, loading, error, refetch: fetchNVRs };
}

export function useNVR(id: string) {
  const [nvr, setNVR] = useState<NVRDTO | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchNVR = async () => {
      setLoading(true);
      setError(null);
      try {
        const service = new NVRService();
        const data = await service.getNVR(id);
        setNVR(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    fetchNVR();
  }, [id]);

  return { nvr, loading, error };
}
```

## 七、相关文档

- [14-NVR模块概述.md](./14-NVR模块概述.md)
- [15-NVR领域模型设计.md](./15-NVR领域模型设计.md)
- [17-NVR配置与运维.md](./17-NVR配置与运维.md)
- [07-RTSP流媒体处理.md](./07-RTSP流媒体处理.md)
