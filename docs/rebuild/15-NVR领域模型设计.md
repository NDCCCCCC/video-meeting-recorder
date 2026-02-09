# NVR领域模型设计

## 一、领域驱动设计概述

### 1.1 DDD在NVR模块中的应用

NVR模块采用**领域驱动设计**（Domain-Driven Design），遵循以下原则：

1. **聚合根（Aggregate Root）** - NVR作为聚合根管理摄像头和录像
2. **值对象（Value Object）** - 不可变的值类型，如NVRID、IPAddress
3. **实体（Entity）** - 有唯一标识和生命周期的对象，如Camera
4. **领域事件（Domain Event）** - 领域内发生的事件，如NVRCreated、MotionDetected
5. **仓储（Repository）** - 聚合根的持久化抽象

### 1.2 领域模型层次

```
┌─────────────────────────────────────────────────────────────────┐
│                         NVR 聚合根                              │
│                        (聚合根)                                   │
├─────────────────────────────────────────────────────────────────┤
│ 属性:                                                             │
│  - NVRID (值对象)                                                  │
│  - Name (值对象)                                                   │
│  - IPAddress (值对象)                                             │
│  - Status (值对象)                                                 │
│  - Version (乐观锁)                                                │
├─────────────────────────────────────────────────────────────────┤
│ 实体集合:                                                         │
│  - Camera[] (摄像头实体)                                          │
├─────────────────────────────────────────────────────────────────┤
│ 关联:                                                             │
│  - LinkedConferenceID (可选)                                      │
├─────────────────────────────────────────────────────────────────┤
│ 方法:                                                             │
│  - Connect()       连接NVR设备                                     │
│  - Disconnect()    断开NVR连接                                     │
│  - AddCamera()     添加摄像头                                     │
│  - RemoveCamera()  移除摄像头                                     │
│  - StartMotionRecording()  启动运动检测录制                      │
│  - StartContinuousRecording() 启动持续录制                       │
│  - LinkToConference()  关联会议                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Camera 实体                               │
│                        (属于NVR聚合根)                             │
├─────────────────────────────────────────────────────────────────┤
│ 属性:                                                             │
│  - CameraID (值对象)                                               │
│  - Name (字符串)                                                   │
│  - StreamURL (值对象)                                             │
│  - Status (值对象)                                                 │
│  - RecordingConfig (值对象)                                       │
│  - CurrentRecording (值对象)                                      │
├─────────────────────────────────────────────────────────────────┤
│ 方法:                                                             │
│  - StartRecording()      开始录制                                │
│  - StopRecording()       停止录制                                │
│  - UpdateConfig()         更新配置                                │
└─────────────────────────────────────────────────────────────────┘
```

## 二、值对象设计

### 2.1 NVRID

```go
// NVRID NVR设备ID值对象
type NVRID struct {
    value string
}

// NewNVRID 创建NVRID
func NewNVRID(value string) (NVRID, error) {
    value = strings.TrimSpace(value)
    if value == "" {
        return NVRID{}, ErrInvalidNVRID
    }

    // 验证格式（可选）
    if len(value) < 3 || len(value) > 64 {
        return NVRID{}, ErrInvalidNVRID
    }

    return NVRID{value: value}, nil
}

// Value 返回ID值
func (id NVRID) Value() string {
    return id.value
}

// Equals 比较两个NVRID是否相等
func (id NVRID) Equals(other NVRID) bool {
    return id.value == other.value
}

// String 返回字符串表示
func (id NVRID) String() string {
    return id.value
}
```

### 2.2 IPAddress

```go
// IPAddress IP地址值对象
type IPAddress struct {
    value string
}

// NewIPAddress 创建IPAddress
func NewIPAddress(value string) (IPAddress, error) {
    value = strings.TrimSpace(value)

    // 验证IP地址格式
    if net.ParseIP(value) == nil {
        return IPAddress{}, ErrInvalidIPAddress
    }

    return IPAddress{value: value}, nil
}

// Value 返回IP地址
func (ip IPAddress) Value() string {
    return ip.value
}

// IsIPv4 检查是否为IPv4地址
func (ip IPAddress) IsIPv4() bool {
    return strings.Contains(ip.value, ".")
}

// IsIPv6 检查是否为IPv6地址
func (ip IPAddress) IsIPv6() bool {
    return strings.Contains(ip.value, ":")
}

// String 返回字符串表示
func (ip IPAddress) String() string {
    return ip.value
}
```

### 2.3 StreamURL

```go
// StreamURL RTSP流URL值对象
type StreamURL struct {
    value string
}

// NewStreamURL 创建StreamURL
func NewStreamURL(value string) (StreamURL, error) {
    value = strings.TrimSpace(value)

    // 验证RTSP URL格式
    if !strings.HasPrefix(value, "rtsp://") {
        return StreamURL{}, ErrInvalidStreamURL
    }

    // 验证URL有效性
    _, err := url.Parse(value)
    if err != nil {
        return StreamURL{}, ErrInvalidStreamURL
    }

    return StreamURL{value: value}, nil
}

// Value 返回URL
func (u StreamURL) Value() string {
    return u.value
}

// ExtractHost 提取主机地址
func (u StreamURL) ExtractHost() string {
    parsed, _ := url.Parse(u.value)
    return parsed.Host
}

// ExtractPort 提取端口
func (u StreamURL) ExtractPort() int {
    parsed, _ := url.Parse(u.value)
    if port := parsed.Port(); port > 0 {
        return port
    }
    return 554 // RTSP默认端口
}

// String 返回字符串表示
func (u StreamURL) String() string {
    return u.value
}
```

### 2.4 RecordingConfig

```go
// RecordingConfig 录制配置值对象
type RecordingConfig struct {
    resolution         Resolution
    fps                int
    videoBitrate       string
    audioBitrate       string
    segmentDuration    time.Duration
    motionDetection    MotionDetectionConfig
}

// Resolution 分辨率值对象
type Resolution struct {
    width  int
    height int
}

func NewResolution(width, height int) (Resolution, error) {
    if width <= 0 || height <= 0 {
        return Resolution{}, errors.New("分辨率必须大于0")
    }
    if width > 7680 || height > 4320 {
        return Resolution{}, errors.New("分辨率超出支持范围")
    }

    return Resolution{width: width, height: height}, nil
}

func (r Resolution) Width() int   { return r.width }
func (r Resolution) Height() int  { return r.height }
func (r Resolution) String() string {
    return fmt.Sprintf("%dx%d", r.width, r.height)
}

// MotionDetectionConfig 运动检测配置
type MotionDetectionConfig struct {
    enabled           bool
    sensitivity       string  // "low", "medium", "high"
    preRecordSeconds  int     // 预录像秒数
    postRecordSeconds int     // 延迟停止秒数
    detectionZones    []DetectionZone
}

// DetectionZone 检测区域
type DetectionZone struct {
    Name       string
    Coordinates string // "x1,y1,x2,y2"
}

// DefaultRecordingConfig 默认录制配置
func DefaultRecordingConfig() RecordingConfig {
    return RecordingConfig{
        resolution:      Resolution{1920, 1080},
        fps:             25,
        videoBitrate:    "3000k",
        audioBitrate:    "128k",
        segmentDuration: 5 * time.Minute,
        motionDetection: MotionDetectionConfig{
            enabled:           false,
            sensitivity:       "medium",
            preRecordSeconds:  3,
            postRecordSeconds: 5,
        },
    }
}
```

## 三、聚合根设计

### 3.1 NVR聚合根

```go
// NVR NVR聚合根
type NVR struct {
    // 唯一标识和元数据
    id        NVRID
    createdAt time.Time
    updatedAt time.Time
    version   int

    // 值对象（不可变）
    name           string
    ipAddress      IPAddress
    status         NVRStatus

    // 实体集合
    cameras        CameraList

    // 可选关联
    linkedConference *string

    // 领域事件
    eventList      []events.Event
    mu             sync.RWMutex
}

// NVRStatus NVR状态值对象
type NVRStatus string

const (
    NVRStatusOnline      NVRStatus = "online"
    NVRStatusOffline     NVRStatus = "offline"
    NVRStatusRecording   NVRStatus = "recording"
    NVRStatusError       NVRStatus = "error"
    NVRStatusMaintaining NVRStatus = "maintaining"
)

// 状态转换规则
func (s NVRStatus) CanTransitionTo(newStatus NVRStatus) bool {
    transitions := map[NVRStatus][]NVRStatus{
        NVRStatusOffline:     {NVRStatusOnline, NVRStatusError},
        NVRStatusOnline:      {NVRStatusOffline, NVRStatusRecording, NVRStatusError, NVRStatusMaintaining},
        NVRStatusRecording:  {NVRStatusOnline, NVRStatusError},
        NVRStatusError:       {NVRStatusOffline, NVRStatusOnline},
        NVRStatusMaintaining: {NVRStatusOnline, NVRStatusOffline},
    }

    allowed, ok := transitions[s]
    if !ok {
        return false
    }

    for _, status := range allowed {
        if status == newStatus {
            return true
        }
    }
    return false
}

// NewNVR 创建新NVR（工厂方法）
func NewNVR(
    id NVRID,
    name string,
    ipAddress IPAddress,
) (*NVR, error) {
    if id.Value() == "" {
        return nil, ErrInvalidNVRID
    }

    nvr := &NVR{
        id:          id,
        name:        name,
        ipAddress:   ipAddress,
        status:      NVRStatusOffline,
        cameras:     NewCameraList(),
        createdAt:   time.Now(),
        updatedAt:   time.Now(),
        version:     1,
        eventList:   make([]events.Event, 0),
    }

    nvr.recordEvent(NewNVRCreatedEvent(id, name, ipAddress, time.Now()))

    return nvr, nil
}

// Connect 连接NVR设备
func (n *NVR) Connect(ctx context.Context) error {
    n.mu.Lock()
    defer n.mu.Unlock()

    if !n.status.CanTransitionTo(NVRStatusOnline) {
        return ErrInvalidStatusTransition
    }

    n.status = NVRStatusOnline
    n.incrementVersion()
    n.recordTime()
    n.recordEvent(NewNVROnlineEvent(n.id, time.Now()))

    return nil
}

// Disconnect 断开NVR连接
func (n *NVR) Disconnect(ctx context.Context, reason string) error {
    n.mu.Lock()
    defer n.mu.Unlock()

    if !n.status.CanTransitionTo(NVRStatusOffline) {
        return ErrInvalidStatusTransition
    }

    n.status = NVRStatusOffline
    n.incrementVersion()
    n.recordTime()
    n.recordEvent(NewNVROfflineEvent(n.id, time.Now(), reason))

    return nil
}

// AddCamera 添加摄像头
func (n *NVR) AddCamera(
    cameraID CameraID,
    name string,
    streamURL StreamURL,
    config RecordingConfig,
) error {
    n.mu.Lock()
    defer n.mu.Unlock()

    camera := NewCamera(cameraID, name, streamURL, config)

    if err := n.cameras.Add(camera); err != nil {
        return err
    }

    n.incrementVersion()
    n.recordTime()
    n.recordEvent(NewCameraAddedEvent(n.id, cameraID, time.Now()))

    return nil
}

// StartMotionRecording 启动运动检测录制
func (n *NVR) StartMotionRecording(
    ctx context.Context,
    cameraID CameraID,
) error {
    n.mu.Lock()
    defer n.mu.Unlock()

    if n.status != NVRStatusOnline {
        return ErrNVRNotOnline
    }

    camera, err := n.cameras.Get(cameraID)
    if err != nil {
        return err
    }

    if err := camera.StartRecording(ctx); err != nil {
        return err
    }

    n.status = NVRStatusRecording
    n.incrementVersion()
    n.recordTime()
    n.recordEvent(NewMotionRecordingStartedEvent(n.id, cameraID, time.Now()))

    return nil
}

// LinkToConference 关联会议
func (n *NVR) LinkToConference(conferenceID string) error {
    n.mu.Lock()
    defer n.mu.Unlock()

    n.linkedConference = &conferenceID
    n.incrementVersion()
    n.recordTime()

    return nil
}

// UnlinkFromConference 取消关联会议
func (n *NVR) UnlinkFromConference() error {
    n.mu.Lock()
    defer n.mu.Unlock()

    n.linkedConference = nil
    n.incrementVersion()
    n.recordTime()

    return nil
}

// GetEvents 获取并清空领域事件
func (n *NVR) GetEvents() []events.Event {
    n.mu.Lock()
    defer n.mu.Unlock()

    events := n.eventList
    n.eventList = make([]events.Event, 0)
    return events
}

// 内部方法
func (n *NVR) recordEvent(event events.Event) {
    n.eventList = append(n.eventList, event)
}

func (n *NVR) recordTime() {
    n.updatedAt = time.Now()
}

func (n *NVR) incrementVersion() {
    n.version++
}

// Getters
func (n *NVR) ID() NVRID { return n.id }
func (n *NVR) Name() string { return n.name }
func (n *NVR) IPAddress() IPAddress { return n.ipAddress }
func (n *NVR) Status() NVRStatus { return n.status }
func (n *NVR) Cameras() CameraList { return n.cameras }
func (n *NVR) Version() int { return n.version }
func (n *NVR) CreatedAt() time.Time { return n.createdAt }
func (n *NVR) UpdatedAt() time.Time { return n.updatedAt }
```

### 3.2 Camera实体

```go
// Camera 摄像头实体
type Camera struct {
    id              CameraID
    name            string
    streamURL       StreamURL
    status          CameraStatus
    recordingConfig RecordingConfig
    currentRecording *RecordingID
    createdAt       time.Time
    updatedAt       time.Time
}

// CameraID 摄像头ID值对象
type CameraID struct {
    value string
}

func NewCameraID(value string) (CameraID, error) {
    value = strings.TrimSpace(value)
    if value == "" {
        return CameraID{}, ErrInvalidCameraID
    }
    return CameraID{value: value}, nil
}

func (id CameraID) Value() string { return id.value }
func (id CameraID) String() string { return id.value }

// CameraStatus 摄像头状态
type CameraStatus string

const (
    CameraStatusOnline    CameraStatus = "online"
    CameraStatusOffline   CameraStatus = "offline"
    CameraStatusRecording CameraStatus = "recording"
)

// NewCamera 创建新摄像头
func NewCamera(
    id CameraID,
    name string,
    streamURL StreamURL,
    config RecordingConfig,
) *Camera {
    return &Camera{
        id:              id,
        name:            name,
        streamURL:       streamURL,
        status:          CameraStatusOffline,
        recordingConfig: config,
        createdAt:       time.Now(),
        updatedAt:       time.Now(),
    }
}

// StartRecording 开始录制
func (c *Camera) StartRecording(ctx context.Context) error {
    if c.status == CameraStatusRecording {
        return ErrCameraAlreadyRecording
    }
    c.status = CameraStatusRecording
    c.updatedAt = time.Now()
    return nil
}

// StopRecording 停止录制
func (c *Camera) StopRecording(ctx context.Context) error {
    if c.status != CameraStatusRecording {
        return ErrCameraNotRecording
    }
    c.status = CameraStatusOnline
    c.updatedAt = time.Now()
    c.currentRecording = nil
    return nil
}

// Getters
func (c *Camera) ID() CameraID               { return c.id }
func (c *Camera) Name() string              { return c.name }
func (c *Camera) StreamURL() StreamURL       { return c.streamURL }
func (c *Camera) Status() CameraStatus       { return c.status }
func (c *Camera) Config() RecordingConfig    { return c.recordingConfig }
```

### 3.3 CameraList值对象

```go
// CameraList 摄像头列表值对象
type CameraList struct {
    cameras map[CameraID]Camera
}

// NewCameraList 创建新的摄像头列表
func NewCameraList() CameraList {
    return CameraList{
        cameras: make(map[CameraID]Camera),
    }
}

// Add 添加摄像头
func (l *CameraList) Add(camera Camera) error {
    if _, exists := l.cameras[camera.id]; exists {
        return ErrCameraAlreadyExists
    }
    l.cameras[camera.id] = camera
    return nil
}

// Get 获取摄像头
func (l *CameraList) Get(id CameraID) (*Camera, error) {
    camera, exists := l.cameras[id]
    if !exists {
        return nil, ErrCameraNotFound
    }
    return &camera, nil
}

// Remove 移除摄像头
func (l *CameraList) Remove(id CameraID) error {
    if _, exists := l.cameras[id]; !exists {
        return ErrCameraNotFound
    }
    delete(l.cameras, id)
    return nil
}

// List 列出所有摄像头
func (l *CameraList) List() []Camera {
    result := make([]Camera, 0, len(l.cameras))
    for _, camera := range l.cameras {
        result = append(result, camera)
    }
    return result
}

// Count 返回摄像头数量
func (l *CameraList) Count() int {
    return len(l.cameras)
}
```

## 四、领域事件

### 4.1 事件类型

```go
// NVRCreatedEvent NVR创建事件
type NVRCreatedEvent struct {
    NVRID     string
    Name      string
    IPAddress string
    CreatedAt time.Time
}

// NVROnlineEvent NVR上线事件
type NVROnlineEvent struct {
    NVRID    string
    OnlineAt time.Time
}

// NVROfflineEvent NVR离线事件
type NVROfflineEvent struct {
    NVRID     string
    OfflineAt time.Time
    Reason     string
}

// CameraAddedEvent 摄像头添加事件
type CameraAddedEvent struct {
    NVRID   string
    CameraID string
    AddedAt  time.Time
}

// MotionRecordingStartedEvent 运动录制开始事件
type MotionRecordingStartedEvent struct {
    NVRID     string
    CameraID  string
    StartedAt  time.Time
}

// NVRMotionDetectedEvent 运动检测事件
type NVRMotionDetectedEvent struct {
    NVRID        string
    CameraID     string
    DetectedAt   time.Time
    Confidence   float64
    SnapshotPath string
}

// ConferenceLinkedEvent 会议关联事件
type ConferenceLinkedEvent struct {
    NVRID         string
    ConferenceID  string
    LinkedAt      time.Time
}

// ConferenceUnlinkedEvent 会议取消关联事件
type ConferenceUnlinkedEvent struct {
    NVRID         string
    UnlinkedAt    time.Time
}
```

### 4.2 事件工厂函数

```go
func NewNVRCreatedEvent(id NVRID, name string, ipAddress IPAddress, at time.Time) events.Event {
    return NVRCreatedEvent{
        NVRID:     id.Value(),
        Name:      name,
        IPAddress: ipAddress.Value(),
        CreatedAt: at,
    }
}

func NewNVROnlineEvent(id NVRID, at time.Time) events.Event {
    return NVROnlineEvent{
        NVRID:    id.Value(),
        OnlineAt: at,
    }
}

func NewNVROfflineEvent(id NVRID, at time.Time, reason string) events.Event {
    return NVROfflineEvent{
        NVRID:     id.Value(),
        OfflineAt: at,
        Reason:    reason,
    }
}

func NewCameraAddedEvent(nvrID NVRID, cameraID CameraID, at time.Time) events.Event {
    return CameraAddedEvent{
        NVRID:    nvrID.Value(),
        CameraID: cameraID.Value(),
        AddedAt:  at,
    }
}

func NewMotionRecordingStartedEvent(nvrID NVRID, cameraID CameraID, at time.Time) events.Event {
    return MotionRecordingStartedEvent{
        NVRID:    nvrID.Value(),
        CameraID: cameraID.Value(),
        StartedAt: at,
    }
}

func NewNVRMotionDetectedEvent(nvrID NVRID, cameraID CameraID, confidence float64, snapshotPath string, at time.Time) events.Event {
    return NVRMotionDetectedEvent{
        NVRID:        nvrID.Value(),
        CameraID:     cameraID.Value(),
        DetectedAt:   at,
        Confidence:   confidence,
        SnapshotPath: snapshotPath,
    }
}
```

## 五、领域错误

### 5.1 错误定义

```go
var (
    // NVR相关错误
    ErrNVRNotFound         = NewDomainError("NVR_NOT_FOUND", "NVR不存在")
    ErrNVRAlreadyExists    = NewDomainError("NVR_ALREADY_EXISTS", "NVR已存在")
    ErrInvalidNVRID        = NewDomainError("INVALID_NVR_ID", "NVR ID无效")
    ErrInvalidIPAddress    = NewDomainError("INVALID_IP_ADDRESS", "IP地址无效")
    ErrNVRNotOnline        = NewDomainError("NVR_NOT_ONLINE", "NVR未在线")
    ErrInvalidStatusTransition = NewDomainError("INVALID_STATUS_TRANSITION", "无效的状态转换")

    // 摄像头相关错误
    ErrCameraNotFound       = NewDomainError("CAMERA_NOT_FOUND", "摄像头不存在")
    ErrCameraAlreadyExists  = NewDomainError("CAMERA_ALREADY_EXISTS", "摄像头已存在")
    ErrInvalidCameraID      = NewDomainError("INVALID_CAMERA_ID", "摄像头ID无效")
    ErrInvalidStreamURL     = NewDomainError("INVALID_STREAM_URL", "流URL无效")
    ErrCameraAlreadyRecording = NewDomainError("CAMERA_ALREADY_RECORDING", "摄像头正在录制")
    ErrCameraNotRecording   = NewDomainError("CAMERA_NOT_RECORDING", "摄像头未在录制")

    // 录制相关错误
    ErrMaxCamerasExceeded   = NewDomainError("MAX_CAMERAS_EXCEEDED", "已超过最大摄像头数量")
    ErrRecordingFailed      = NewDomainError("RECORDING_FAILED", "录制失败")
)

// NewDomainError 创建领域错误
func NewDomainError(code, message string) *DomainError {
    return &DomainError{
        Code:    code,
        Message: message,
    }
}

// DomainError 领域错误
type DomainError struct {
    Code    string
    Message string
}

func (e *DomainError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}
```

## 六、仓储接口

### 6.1 Repository接口

```go
// Repository NVR仓储接口
type Repository interface {
    // Save 保存NVR聚合根
    Save(ctx context.Context, nvr *NVR) error

    // FindByID 根据ID查找NVR
    FindByID(ctx context.Context, id NVRID) (*NVR, error)

    // FindByIPAddress 根据IP地址查找NVR
    FindByIPAddress(ctx context.Context, ip IPAddress) (*NVR, error)

    // FindByStatus 根据状态查找NVR列表
    FindByStatus(ctx context.Context, status NVRStatus) ([]*NVR, error)

    // FindAll 查找所有NVR
    FindAll(ctx context.Context) ([]*NVR, error)

    // Delete 删除NVR
    Delete(ctx context.Context, id NVRID) error

    // Exists 检查NVR是否存在
    Exists(ctx context.Context, id NVRID) (bool, error)

    // NextID 生成下一个NVR ID
    NextID(ctx context.Context) (NVRID, error)
}
```

## 七、单元测试

### 7.1 聚合根测试示例

```go
func TestNVR_AddCamera(t *testing.T) {
    tests := []struct {
        name    string
        nvr     *NVR
        camera  Camera
        wantErr error
    }{
        {
            name: "成功添加摄像头",
            nvr:  createTestNVR(),
            camera: Camera{
                id:        mustNewCameraID("cam-001"),
                name:      "测试摄像头",
                streamURL: mustNewStreamURL("rtsp://192.168.1.100/stream"),
            },
            wantErr: nil,
        },
        {
            name: "添加重复摄像头",
            nvr: func() *NVR {
                nvr := createTestNVR()
                camera := Camera{
                    id:        mustNewCameraID("cam-001"),
                    name:      "测试摄像头",
                    streamURL: mustNewStreamURL("rtsp://192.168.1.100/stream"),
                }
                nvr.cameras.Add(camera)
                return nvr
            }(),
            camera: Camera{
                id:        mustNewCameraID("cam-001"),
                name:      "重复摄像头",
                streamURL: mustNewStreamURL("rtsp://192.168.1.101/stream"),
            },
            wantErr: ErrCameraAlreadyExists,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.nvr.AddCamera(tt.camera.id, tt.camera.name, tt.camera.streamURL, tt.camera.recordingConfig)
            if tt.wantErr != nil {
                if err == nil || err.Error() != tt.wantErr.Error() {
                    t.Errorf("AddCamera() error = %v, want %v", err, tt.wantErr)
                }
            } else {
                if err != nil {
                    t.Errorf("AddCamera() unexpected error = %v", err)
                }
            }
        })
    }
}
```

## 八、相关文档

- [14-NVR模块概述.md](./14-NVR模块概述.md)
- [16-NVR架构与实现.md](./16-NVR架构与实现.md)
- [17-NVR配置与运维.md](./17-NVR配置与运维.md)
