# NVR模块概述

## 一、NVR模块简介

### 1.1 模块定义

NVR（Network Video Recorder）硬盘录像机模块是作为**独立微服务**部署的专业监控录像子系统，与主服务通过gRPC进行通信，提供完整的网络摄像头管理、录像、运动检测等功能。

### 1.2 架构模式

**微服务架构** - NVR作为独立服务运行，具有以下特性：

| 特性 | 说明 |
|------|------|
| **独立部署** | 可独立部署、升级、扩展 |
| **资源隔离** | 独立的CPU/内存限制 |
| **故障隔离** | NVR故障不影响主服务 |
| **数据隔离** | 独立的数据库和存储 |
| **通信协议** | 与主服务通过gRPC/TLS通信 |

### 1.3 核心功能

| 功能 | 描述 | 状态 |
|------|------|------|
| **RTSP流接收** | 接受来自网络摄像头的RTSP流输入 | ✅ 已完成 |
| **ONVIF集成** | 通过ONVIF协议管理支持PTZ控制的摄像头 | ✅ 已完成 |
| **运动检测录制** | 检测画面运动时自动触发录制 | ✅ 已完成 |
| **持续录制** | 支持全天候持续录制模式 | ✅ 已完成 |
| **智能存储管理** | 磁盘空间不足时自动删除旧录像 | ✅ 已完成 |
| **会议集成** | 通过gRPC与会议系统关联，会议开始时同步录制 | ✅ 已完成 |
| **gRPC API** | 提供标准gRPC接口供主服务调用 | ✅ 已完成 |

### 1.4 技术规格

| 项目 | 规格 |
|------|------|
| 支持协议 | RTSP, ONVIF, gRPC |
| 视频编码 | H.264/H.265 |
| 音频编码 | AAC |
| 封装格式 | MP4, TS |
| 最大并发路数 | 4路（当前配置2路） |
| 分辨率支持 | 最高4K（推荐1080p） |
| 录制分段 | 5分钟/文件 |
| 存储策略 | 循环删除 + 保护机制 |
| 架构模式 | 微服务 + DDD领域驱动设计 |
| 通信端口 | 9091(gRPC), 8081(HTTP), 8082(WebSocket) |

### 1.5 与RTSP流录制的区别

| 特性 | RTSP流录制 | NVR微服务 |
|------|------------|-----------|
| **部署方式** | 内嵌于主服务 | 独立微服务 |
| **通信协议** | 函数调用 | gRPC |
| **主要用途** | 华为会议录制 | 专业监控系统 |
| **运动检测** | ❌ 不支持 | ✅ 支持 |
| **ONVIF协议** | ❌ 不支持 | ✅ 支持 |
| **PTZ控制** | ❌ 不支持 | ✅ 支持 |
| **设备管理** | 简单 | 完整（NVR设备、摄像头） |
| **存储管理** | 基础 | 智能（循环删除、保护机制） |
| **故障隔离** | ❌ 故障影响主服务 | ✅ 故障隔离 |
| **独立扩展** | ❌ 需整体扩展 | ✅ 按需扩展NVR实例 |
| **数据隔离** | ❌ 共享数据库 | ✅ 独立数据库 |
| **架构设计** | 传统服务层 | 微服务 + DDD |
| **代码复杂度** | 简单 | 复杂（独立服务8000+行） |
| **适用场景** | 华为会议录制 | 专业监控系统 |

## 二、系统架构

### 2.1 微服务整体架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           主服务 (Main Service)                             │
│                       :8080 (HTTP) / :9090 (gRPC)                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │ Conference   │  │   User       │  │   Task       │  │ NVR Client   │    │
│  │   Service    │  │  Service     │  │  Service     │  │ (gRPC Client)│    │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────┬───────┘    │
└────────────────────────────────────────────────────────────┼───────────────┘
                                                               │
                                                          gRPC/TLS
                                                               │
┌────────────────────────────────────────────────────────────┼───────────────┐
│                    NVR微服务 (NVR Service)                  │              │
│                       :9091 (gRPC) / :8081 (HTTP)          │              │
│  ┌──────────────────────────────────────────────────────────┐          │
│  │                 gRPC Server Layer                       │          │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │          │
│  │  │ NVR Service  │  │Camera Service│  │Recording     │    │          │
│  │  │  (gRPC)      │  │  (gRPC)      │  │Service(gRPC) │    │          │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘    │          │
│  └─────────┼──────────────────┼──────────────────┼────────────┘          │
│            │                  │                  │                       │
│  ┌─────────┼──────────────────┼──────────────────┼─────────────────────┐ │
│  │         │      Application Layer (CQRS)        │                     │ │
│  │  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────────▼──┐               │ │
│  │  │  NVR Commands│  │  NVR Queries │  │Motion Detect │               │ │
│  │  │  (Write)    │  │   (Read)    │  │   Service    │               │ │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘               │ │
│  └─────────┼──────────────────┼──────────────────┼─────────────────────┘ │
│            │                  │                  │                       │
│  ┌─────────┼──────────────────┼──────────────────┼─────────────────────┐ │
│  │         │         Domain Layer (DDD)                             │ │
│  │  ┌──────▼──────┐  ┌──────┬───────┐  ┌────────▼──────┐           │ │
│  │  │   NVR       │  │Camera│Recording│ Domain Events  │           │ │
│  │  │  Aggregate  │  │Entity│ Entity  │                │           │ │
│  │  └──────┬──────┘  └──────┴───────┘  └───────────────┘           │ │
│  └─────────┼──────────────────┬──────────────────────────────────────┘ │
│            │                  │                                        │
│  ┌─────────┼──────────────────┼──────────────────────────────────────┐ │
│  │         │    Infrastructure Layer                               │ │
│  │  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────────▼──┐             │ │
│  │  │NVR Repository│  │ ONVIF      │  │ Storage     │             │ │
│  │  │(Persistence) │  │ Adapter    │  │ Manager     │             │ │
│  │  └─────────────┘  └─────┬──────┘  └─────────────┘             │ │
│  │                        │                                      │ │
│  │              ┌─────────┴──────────┐                           │ │
│  │              ▼                    ▼                           │ │
│  │         ┌─────────┐         ┌─────────┐                       │ │
│  │         │  RTSP   │         │ FFmpeg  │                       │ │
│  │         │ Adapter │         │ Adapter │                       │ │
│  │         └─────────┘         └─────────┘                       │ │
│  └──────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  独立数据库: data/nvr.db                                              │
│  独立存储: data/recordings/                                           │
│  独立配置: configs/nvr.yaml                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### 2.2 服务通信

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         主服务                                          │
│                                                                          │
│  internal/services/nvr_client/                                          │
│  ├── client.go           # gRPC客户端                                    │
│  ├── nvr.go              # NVR操作封装                                  │
│  ├── camera.go           # 摄像头操作封装                               │
│  └── recording.go        # 录制操作封装                                │
└─────────────────────────────────────────────────────────────────────────┘
                                │ gRPC/TLS
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                       NVR微服务                                          │
│                                                                          │
│  api/grpc/proto/                                                         │
│  ├── nvr/nvr.proto           # NVR服务定义                              │
│  ├── nvr/camera.proto        # 摄像头服务定义                           │
│  ├── nvr/recording.proto     # 录制服务定义                             │
│  └── nvr/events.proto        # 事件流定义                               │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.3 NVR微服务目录结构

```
nvr-service/                              # 独立的微服务项目
├── cmd/
│   └── server/
│       └── main.go                     # 服务入口
│
├── api/
│   ├── grpc/
│   │   └── proto/                      # Protobuf定义
│   │       ├── nvr/
│   │       │   ├── nvr.proto
│   │       │   ├── camera.proto
│   │       │   ├── recording.proto
│   │       │   └── events.proto
│   │       └── common/
│   │           ├── common.proto
│   │           └── error.proto
│   └── http/
│       └── handlers/                   # HTTP处理器（可选，用于管理界面）
│
├── internal/
│   ├── domain/                          # 领域层
│   │   ├── nvr/
│   │   │   ├── aggregate.go            # NVR聚合根
│   │   │   ├── value_objects.go        # 值对象
│   │   │   ├── events.go               # 领域事件
│   │   │   ├── repository.go           # 仓储接口
│   │   │   └── errors.go               # 领域错误
│   │   └── camera/
│   │       ├── entity.go               # Camera实体
│   │       └── value_objects.go
│   │
│   ├── application/                    # 应用层
│   │   ├── nvr/
│   │   │   ├── command/                # 命令处理器
│   │   │   ├── query/                  # 查询处理器
│   │   │   └── dto.go                  # DTO定义
│   │   └── grpc/
│   │       ├── server.go               # gRPC服务器
│   │       └── interceptors/           # gRPC拦截器
│   │
│   ├── infrastructure/                 # 基础设施层
│   │   ├── persistence/                # 持久化
│   │   │   ├── sqlite/
│   │   │   │   └── repository.go       # SQLite实现
│   │   │   └── models.go               # 数据模型
│   │   ├── rtsp/                       # RTSP适配器
│   │   │   ├── client.go
│   │   │   └── recorder.go
│   │   ├── ffmpeg/                     # FFmpeg适配器
│   │   │   └── orchestrator.go
│   │   ├── onvif/                      # ONVIF适配器
│   │   │   ├── client.go
│   │   │   └── ptz.go
│   │   └── storage/                    # 存储管理
│   │       └── manager.go
│   │
│   └── config/                         # 配置
│       └── config.go
│
├── pkg/
│   └── api/                            # 生成的gRPC代码
│
├── configs/
│   └── nvr.yaml                        # 配置文件
│
├── scripts/
│   ├── generate.sh                     # 生成gRPC代码
│   └── build.sh                        # 构建脚本
│
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile
│   │   └── docker-compose.yml
│   └── k8s/
│       ├── deployment.yaml
│       └── service.yaml
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### 2.4 主服务集成

```
record_go/ (主服务)
├── internal/
│   ├── services/
│   │   └── nvr_client/                  # NVR服务客户端
│   │       ├── client.go               # gRPC客户端
│   │       ├── nvr.go                  # NVR操作封装
│   │       ├── camera.go               # 摄像头操作封装
│   │       └── recording.go            # 录制操作封装
│   │
│   └── interfaces/
│       └── http/
│           └── handlers/
│               └── nvr_proxy.go        # NVR API代理（可选）
│
└── configs/
    └── config.yaml                     # 添加NVR服务地址配置
```

### 2.5 架构原则

1. **微服务独立** - NVR作为独立服务部署、运行、扩展
2. **依赖倒置** - 领域层定义接口，基础设施层实现
3. **聚合根边界** - NVR作为独立聚合根管理摄像头和录像
4. **CQRS分离** - 命令和查询严格分离
5. **事件驱动** - 通过领域事件解耦模块
6. **gRPC通信** - 与主服务通过gRPC/TLS安全通信
7. **故障隔离** - NVR故障不影响主服务运行

## 三、核心组件

### 3.1 领域层组件

| 组件 | 文件 | 行数 | 职责 |
|------|------|------|------|
| NVR Aggregate | aggregate.go | 671 | NVR聚合根、Camera实体 |
| Value Objects | value_objects.go | 488 | 值对象定义 |
| Domain Events | events.go | 524 | 领域事件 |
| Repository Interface | repository.go | 81 | 仓储接口 |
| Domain Errors | errors.go | 92 | 领域错误 |

### 3.2 应用层组件

| 组件 | 文件 | 行数 | 职责 |
|------|------|------|------|
| gRPC Server | grpc/server.go | 200 | gRPC服务器 |
| gRPC Interceptors | grpc/interceptors/ | 150 | 认证、日志、追踪拦截器 |
| Command Handlers | command/nvr_commands.go | 520 | 命令处理 |
| Query Handlers | query/nvr_queries.go | 370 | 查询处理 |
| Motion Detection | service/motion_detection_service.go | - | 运动检测服务 |

### 3.3 基础设施层组件

| 组件 | 文件 | 行数 | 职责 |
|------|------|------|------|
| NVR Repository | persistence/nvr_repository_impl.go | 388 | 数据持久化 |
| Storage Manager | storage/manager.go | 320 | 存储管理 |
| ONVIF Adapter | adapters/onvif/adapter.go | 170 | ONVIF协议 |
| RTSP Adapter | rtsp/client.go | 250 | RTSP客户端 |
| FFmpeg Orchestrator | ffmpeg/orchestrator.go | 400 | FFmpeg进程管理 |

### 3.4 主服务客户端组件

| 组件 | 文件 | 行数 | 职责 |
|------|------|------|------|
| NVR Client | services/nvr_client/client.go | 150 | gRPC客户端封装 |
| NVR Operations | services/nvr_client/nvr.go | 200 | NVR操作封装 |
| Camera Operations | services/nvr_client/camera.go | 180 | 摄像头操作封装 |
| Recording Operations | services/nvr_client/recording.go | 220 | 录制操作封装 |

## 四、数据模型

### 4.1 核心实体

```
┌─────────────────────────────────────────────────────────────────┐
│                        NVR (聚合根)                              │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ NVRID                                                   │  │
│  │ Name                                                    │  │
│  │ IPAddress                                               │  │
│  │ Status (online/offline/recording/error)               │  │
│  │ LinkedConferenceID (可选)                              │  │
│  └───────────────────────────────────────────────────────────┘  │
│                           │                                   │
│                           ▼                                   │
│              ┌─────────────────────────────┐                │
│              │      CameraList (值对象)      │                │
│              │  ┌─────────────────────┐    │                │
│              │  │ Camera (实体)         │    │                │
│              │  │ - CameraID          │    │                │
│              │  │ - Name              │    │                │
│              │  │ - StreamURL         │    │                │
│              │  │ - Status            │    │                │
│              │  │ - RecordingConfig  │    │                │
│              │  └─────────────────────┘    │                │
│              └─────────────────────────────┘                │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 状态定义

#### NVRStatus (NVR状态)

```go
const (
    NVRStatusOnline      = "online"       // 在线
    NVRStatusOffline     = "offline"      // 离线
    NVRStatusRecording   = "recording"    // 录制中
    NVRStatusError       = "error"        // 错误
    NVRStatusMaintaining = "maintaining"  // 维护中
)
```

#### CameraStatus (摄像头状态)

```go
const (
    CameraStatusOnline    = "online"     // 在线
    CameraStatusOffline   = "offline"    // 离线
    CameraStatusRecording = "recording"  // 录制中
)
```

### 4.3 数据库表结构

#### nvr_devices (NVR设备表) - NVR服务独立数据库

```sql
CREATE TABLE nvr_devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nvr_id VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    port INTEGER DEFAULT 8000,
    username VARCHAR(100),
    password VARCHAR(255),
    status VARCHAR(20) DEFAULT 'offline',
    linked_conference_id VARCHAR(64),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    version INTEGER DEFAULT 1
);
```

#### nvr_cameras (摄像头表)

```sql
CREATE TABLE nvr_cameras (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    camera_id VARCHAR(64) UNIQUE NOT NULL,
    nvr_id VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    stream_url TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'offline',
    resolution VARCHAR(20) DEFAULT '1920x1080',
    fps INTEGER DEFAULT 25,
    motion_detection_enabled BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (nvr_id) REFERENCES nvr_devices(nvr_id) ON DELETE CASCADE
);
```

#### nvr_recordings (录像表)

```sql
CREATE TABLE nvr_recordings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    recording_id VARCHAR(64) UNIQUE NOT NULL,
    nvr_id VARCHAR(64) NOT NULL,
    camera_id VARCHAR(64) NOT NULL,
    file_path TEXT NOT NULL,
    duration INTEGER DEFAULT 0,
    file_size BIGINT DEFAULT 0,
    motion_detected BOOLEAN DEFAULT 0,
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    stopped_at DATETIME,
    FOREIGN KEY (nvr_id) REFERENCES nvr_devices(nvr_id) ON DELETE CASCADE
);
```

## 五、API接口

### 5.1 gRPC服务

| 服务 | 方法 | 描述 |
|------|------|------|
| **NVRService** | CreateNVR | 创建NVR设备 |
| | ListNVRs | 获取NVR列表 |
| | GetNVR | 获取NVR详情 |
| | UpdateNVR | 更新NVR配置 |
| | DeleteNVR | 删除NVR设备 |
| | ConnectNVR | 连接NVR设备 |
| | DisconnectNVR | 断开NVR连接 |
| | GetNVRStatus | 获取NVR状态 |
| | HealthCheck | 健康检查 |
| **CameraService** | AddCamera | 添加摄像头 |
| | ListCameras | 获取摄像头列表 |
| | GetCamera | 获取摄像头详情 |
| | UpdateCamera | 更新摄像头配置 |
| | DeleteCamera | 删除摄像头 |
| | GetSnapshot | 获取摄像头快照 |
| | GetPreviewURL | 获取预览流URL |
| **RecordingService** | StartMotionRecording | 启动运动检测录制 |
| | StopMotionRecording | 停止运动检测录制 |
| | StartContinuousRecording | 启动持续录制 |
| | StopContinuousRecording | 停止持续录制 |
| | GetRecordingStatus | 获取录制状态 |
| | ListRecordings | 获取录像列表 |
| | GetRecording | 获取录像详情 |
| | DeleteRecording | 删除录像 |
| | DownloadRecording | 下载录像（流式） |
| **EventStreamService** | SubscribeEventStream | 订阅NVR事件流 |
| | SubscribeCameraEvents | 订阅摄像头事件 |

### 5.2 HTTP管理API (可选)

| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/metrics` | Prometheus指标 |
| GET | `/api/v1/status` | 服务状态 |

### 5.3 WebSocket事件

| 事件类型 | 描述 |
|---------|------|
| `nvr.created` | NVR创建 |
| `nvr.online` | NVR上线 |
| `nvr.offline` | NVR离线 |
| `camera.added` | 摄像头添加 |
| `recording.started` | 录制开始 |
| `recording.stopped` | 录制停止 |
| `motion.detected` | 运动检测 |

## 六、使用场景

### 6.1 华为会议录制

使用RTSP流录制模块更合适：
- 集成简单，无需额外部署
- 直接获取华为终端RTSP流
- 与现有录制任务系统紧密集成

### 6.2 专业监控场景

使用NVR微服务更合适：
- 运动检测触发录制
- ONVIF设备管理
- PTZ云台控制
- 智能存储管理
- 独立扩展和管理

### 6.3 混合场景

两个模块可以共存使用：
- 华为会议录制 → RTSP流录制模块（内嵌）
- 监控摄像头录制 → NVR微服务（独立）

## 七、部署模式

### 7.1 单机部署

```bash
# 主服务
./record_go serve

# NVR服务
cd nvr-service
./nvr-service --config configs/nvr.yaml
```

### 7.2 Docker部署

```bash
docker-compose up -d
```

### 7.3 Kubernetes部署

```bash
kubectl apply -f deployments/k8s/
```

## 八、相关文档

- [15-NVR领域模型设计.md](./15-NVR领域模型设计.md) - DDD领域模型详解
- [16-NVR架构与实现.md](./16-NVR架构与实现.md) - 技术架构详解
- [17-NVR配置与运维.md](./17-NVR配置与运维.md) - 配置和运维指南
- [18-NVR微服务架构设计.md](./18-NVR微服务架构设计.md) - **微服务架构设计（新增）**
- [07-RTSP流媒体处理.md](./07-RTSP流媒体处理.md) - RTSP流录制模块
- [04-华为系统集成详解.md](./04-华为系统集成详解.md) - 华为集成
