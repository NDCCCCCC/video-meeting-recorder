# NVR微服务架构设计

## 一、架构概述

### 1.1 微服务定位

NVR（Network Video Recorder）作为独立的微服务部署，专注于网络视频录制管理，与主服务通过gRPC进行通信。

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
│                       :8081 (HTTP) / :9091 (gRPC)          │              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │              │
│  │ NVR Domain   │  │ RTSP         │  │ ONVIF        │    │              │
│  │   (DDD)      │  │ Adapter      │  │ Adapter      │    │              │
│  └──────────────┘  └──────────────┘  └──────────────┘    │              │
│                                                              │              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │              │
│  │ FFmpeg       │  │ Storage      │  │ gRPC Server  │    │              │
│  │ Orchestrator │  │ Manager      │  │              │◄───┘              │
│  └──────────────┘  └──────────────┘  └──────────────┘                   │
│                                                                      │      │
│  独立数据库: data/nvr.db                                            │      │
│  独立存储: data/recordings/                                         │      │
│  独立配置: config/nvr.yaml                                         │      │
└──────────────────────────────────────────────────────────────────────────┘
```

### 1.2 架构优势

| 特性 | 单体架构 | 微服务架构 |
|------|---------|-----------|
| **独立部署** | ❌ 需重启主服务 | ✅ 独立部署和升级 |
| **资源隔离** | ❌ 共享资源 | ✅ 独立的CPU/内存限制 |
| **技术栈** | ❌ 必须一致 | ✅ 可采用不同技术 |
| **故障隔离** | ❌ NVR故障影响主服务 | ✅ 故障隔离 |
| **水平扩展** | ❌ 整体扩展 | ✅ 按需扩展NVR实例 |
| **数据隔离** | ❌ 共享数据库 | ✅ 独立数据库 |

### 1.3 通信协议

| 协议 | 用途 | 端口 | 说明 |
|------|------|------|------|
| gRPC | 主服务 ↔ NVR服务 | 9091 | 高性能RPC调用 |
| HTTP | NVR管理API | 8081 | 前端直接调用 |
| WebSocket | NVR实时事件 | 8082 | 运动检测事件推送 |

### 1.4 服务发现

```yaml
# 支持的服务发现方式
service_discovery:
  type: "consul"  # 或 "etcd", "k8s"
  address: "localhost:8500"
  health_check:
    interval: 10s
    timeout: 3s
```

## 二、gRPC接口设计

### 2.1 Proto文件结构

```
api/grpc/proto/
├── nvr/
│   ├── nvr.proto              # NVR服务定义
│   ├── camera.proto           # 摄像头服务定义
│   ├── recording.proto        # 录制服务定义
│   └── events.proto           # 事件流定义
├── common/
│   ├── common.proto           # 通用消息定义
│   └── error.proto            # 错误码定义
└── buf.yaml                   # buf工具配置
```

### 2.2 核心服务定义

```protobuf
// api/grpc/proto/nvr/nvr.proto
syntax = "proto3";

package nvr.v1;
option go_package = "github.com/record_go/api/grpc/proto/nvr/v1;nvr_v1";

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
import "common/common.proto";

// NVR服务 - 管理NVR设备
service NVRService {
  // 创建NVR设备
  rpc CreateNVR(CreateNVRRequest) returns (CreateNVRResponse);

  // 获取NVR列表
  rpc ListNVRs(ListNVRsRequest) returns (ListNVRsResponse);

  // 获取NVR详情
  rpc GetNVR(GetNVRRequest) returns (GetNVRResponse);

  // 更新NVR配置
  rpc UpdateNVR(UpdateNVRRequest) returns (UpdateNVRResponse);

  // 删除NVR设备
  rpc DeleteNVR(DeleteNVRRequest) returns (google.protobuf.Empty);

  // 连接NVR设备
  rpc ConnectNVR(ConnectNVRRequest) returns (ConnectNVRResponse);

  // 断开NVR连接
  rpc DisconnectNVR(DisconnectNVRRequest) returns (google.protobuf.Empty);

  // 获取NVR状态
  rpc GetNVRStatus(GetNVRStatusRequest) returns (GetNVRStatusResponse);

  // 健康检查
  rpc HealthCheck(google.protobuf.Empty) returns (HealthCheckResponse);
}

// 摄像头服务 - 管理摄像头
service CameraService {
  // 添加摄像头
  rpc AddCamera(AddCameraRequest) returns (AddCameraResponse);

  // 获取摄像头列表
  rpc ListCameras(ListCamerasRequest) returns (ListCamerasResponse);

  // 获取摄像头详情
  rpc GetCamera(GetCameraRequest) returns (GetCameraResponse);

  // 更新摄像头配置
  rpc UpdateCamera(UpdateCameraRequest) returns (UpdateCameraResponse);

  // 删除摄像头
  rpc DeleteCamera(DeleteCameraRequest) returns (google.protobuf.Empty);

  // 获取摄像头快照
  rpc GetSnapshot(GetSnapshotRequest) returns (GetSnapshotResponse);

  // 获取摄像头预览流URL
  rpc GetPreviewURL(GetPreviewURLRequest) returns (GetPreviewURLResponse);
}

// 录制服务 - 控制录制
service RecordingService {
  // 启动运动检测录制
  rpc StartMotionRecording(StartMotionRecordingRequest) returns (StartMotionRecordingResponse);

  // 停止运动检测录制
  rpc StopMotionRecording(StopMotionRecordingRequest) returns (google.protobuf.Empty);

  // 启动持续录制
  rpc StartContinuousRecording(StartContinuousRecordingRequest) returns (StartContinuousRecordingResponse);

  // 停止持续录制
  rpc StopContinuousRecording(StopContinuousRecordingRequest) returns (google.protobuf.Empty);

  // 获取录制状态
  rpc GetRecordingStatus(GetRecordingStatusRequest) returns (GetRecordingStatusResponse);

  // 获取录像列表
  rpc ListRecordings(ListRecordingsRequest) returns (ListRecordingsResponse);

  // 获取录像详情
  rpc GetRecording(GetRecordingRequest) returns (GetRecordingResponse);

  // 删除录像
  rpc DeleteRecording(DeleteRecordingRequest) returns (google.protobuf.Empty);

  // 下载录像
  rpc DownloadRecording(DownloadRecordingRequest) returns (stream DownloadRecordingChunk);
}

// 事件流服务 - 实时事件推送
service EventStreamService {
  // 订阅NVR事件流
  rpc SubscribeEventStream(SubscribeEventStreamRequest) returns (stream EventStreamResponse);

  // 订阅摄像头事件流
  rpc SubscribeCameraEvents(SubscribeCameraEventsRequest) returns (stream CameraEvent);
}

// ============== 消息定义 ==============

// NVR设备信息
message NVRDevice {
  string id = 1;                      // NVR唯一ID
  string name = 2;                    // NVR名称
  string ip_address = 3;              // IP地址
  int32 port = 4;                     // 端口号
  NVRStatus status = 5;               // 状态
  string linked_conference_id = 6;    // 关联的会议ID（可选）
  google.protobuf.Timestamp created_at = 7;
  google.protobuf.Timestamp updated_at = 8;
  int32 version = 9;
}

// NVR状态
enum NVRStatus {
  NVR_STATUS_UNKNOWN = 0;
  NVR_STATUS_ONLINE = 1;              // 在线
  NVR_STATUS_OFFLINE = 2;             // 离线
  NVR_STATUS_RECORDING = 3;           // 录制中
  NVR_STATUS_ERROR = 4;               // 错误
  NVR_STATUS_MAINTAINING = 5;         // 维护中
}

// 摄像头信息
message Camera {
  string id = 1;                      // 摄像头ID
  string nvr_id = 2;                  // 所属NVR ID
  string name = 3;                    // 名称
  string stream_url = 4;              // RTSP流地址
  CameraStatus status = 5;            // 状态
  RecordingConfig config = 6;         // 录制配置
  google.protobuf.Timestamp created_at = 7;
  google.protobuf.Timestamp updated_at = 8;
}

// 摄像头状态
enum CameraStatus {
  CAMERA_STATUS_UNKNOWN = 0;
  CAMERA_STATUS_ONLINE = 1;           // 在线
  CAMERA_STATUS_OFFLINE = 2;          // 离线
  CAMERA_STATUS_RECORDING = 3;        // 录制中
}

// 录制配置
message RecordingConfig {
  Resolution resolution = 1;          // 分辨率
  int32 fps = 2;                      // 帧率
  string video_bitrate = 3;           // 视频比特率
  string audio_bitrate = 4;           // 音频比特率
  int32 segment_duration = 5;         // 分段时长（秒）
  MotionDetectionConfig motion_detection = 6;
}

// 分辨率
message Resolution {
  int32 width = 1;
  int32 height = 2;
}

// 运动检测配置
message MotionDetectionConfig {
  bool enabled = 1;
  string sensitivity = 2;             // low, medium, high
  int32 pre_record_seconds = 3;       // 预录制秒数
  int32 post_record_seconds = 4;      // 后录制秒数
  repeated DetectionZone zones = 5;   // 检测区域
}

// 检测区域
message DetectionZone {
  string name = 1;
  string coordinates = 2;             // "x1,y1,x2,y2"
}

// 录像信息
message Recording {
  string id = 1;                      // 录像ID
  string nvr_id = 2;
  string camera_id = 3;
  string file_path = 4;
  int64 duration = 5;                 // 时长（秒）
  int64 file_size = 6;                // 文件大小（字节）
  bool motion_detected = 7;           // 是否检测到运动
  double confidence = 8;              // 运动检测置信度
  google.protobuf.Timestamp started_at = 9;
  google.protobuf.Timestamp stopped_at = 10;
  RecordingStatus status = 11;
}

// 录像状态
enum RecordingStatus {
  RECORDING_STATUS_UNKNOWN = 0;
  RECORDING_STATUS_PENDING = 1;
  RECORDING_STATUS_RECORDING = 2;
  RECORDING_STATUS_COMPLETED = 3;
  RECORDING_STATUS_FAILED = 4;
}

// ============== 请求/响应消息 ==============

// CreateNVR
message CreateNVRRequest {
  string id = 1;
  string name = 2;
  string ip_address = 3;
  int32 port = 4;
  string username = 5;
  string password = 6;
}

message CreateNVRResponse {
  NVRDevice nvr = 1;
}

// ListNVRs
message ListNVRsRequest {
  int32 page_size = 1;
  string page_token = 2;
  NVRStatus filter_status = 3;        // 状态过滤
}

message ListNVRsResponse {
  repeated NVRDevice nvr_devices = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

// GetNVR
message GetNVRRequest {
  string id = 1;
}

message GetNVRResponse {
  NVRDevice nvr = 1;
  repeated Camera cameras = 2;        // 关联的摄像头
  RecordingStats stats = 3;           // 录制统计
}

// UpdateNVR
message UpdateNVRRequest {
  string id = 1;
  optional string name = 2;
  optional string ip_address = 3;
  optional int32 port = 4;
  optional string username = 5;
  optional string password = 6;
}

message UpdateNVRResponse {
  NVRDevice nvr = 1;
}

// DeleteNVR
message DeleteNVRRequest {
  string id = 1;
}

// ConnectNVR
message ConnectNVRRequest {
  string id = 1;
  int32 timeout = 2;                  // 连接超时（秒）
}

message ConnectNVRResponse {
  NVRDevice nvr = 1;
  repeated Camera discovered_cameras = 2;  // 自动发现的摄像头
}

// DisconnectNVR
message DisconnectNVRRequest {
  string id = 1;
  string reason = 2;
}

// GetNVRStatus
message GetNVRStatusRequest {
  string id = 1;
}

message GetNVRStatusResponse {
  NVRStatus status = 1;
  repeated CameraStatus camera_statuses = 2;
  SystemHealth health = 3;
}

// HealthCheck
message HealthCheckResponse {
  bool healthy = 1;
  string version = 2;
  google.protobuf.Timestamp started_at = 3;
  int32 uptime_seconds = 4;
}

// ============== 摄像头相关消息 ==============

// AddCamera
message AddCameraRequest {
  string nvr_id = 1;
  string id = 2;
  string name = 3;
  string stream_url = 4;
  RecordingConfig config = 5;
}

message AddCameraResponse {
  Camera camera = 1;
}

// ListCameras
message ListCamerasRequest {
  string nvr_id = 1;
  int32 page_size = 2;
  string page_token = 3;
}

message ListCamerasResponse {
  repeated Camera cameras = 1;
  string next_page_token = 2;
}

// GetCamera
message GetCameraRequest {
  string nvr_id = 1;
  string camera_id = 2;
}

message GetCameraResponse {
  Camera camera = 1;
  RecordingInfo current_recording = 2;
}

// UpdateCamera
message UpdateCameraRequest {
  string nvr_id = 1;
  string camera_id = 2;
  optional string name = 3;
  optional string stream_url = 4;
  optional RecordingConfig config = 5;
}

message UpdateCameraResponse {
  Camera camera = 1;
}

// DeleteCamera
message DeleteCameraRequest {
  string nvr_id = 1;
  string camera_id = 2;
}

// GetSnapshot
message GetSnapshotRequest {
  string nvr_id = 1;
  string camera_id = 2;
}

message GetSnapshotResponse {
  bytes image_data = 1;              // JPEG格式
  string content_type = 2;
  google.protobuf.Timestamp captured_at = 3;
}

// GetPreviewURL
message GetPreviewURLRequest {
  string nvr_id = 1;
  string camera_id = 2;
  string protocol = 3;               // hls, flv, webrtc
}

message GetPreviewURLResponse {
  string url = 1;
  google.protobuf.Timestamp expires_at = 2;
}

// ============== 录制相关消息 ==============

// StartMotionRecording
message StartMotionRecordingRequest {
  string nvr_id = 1;
  string camera_id = 2;
  uint32 started_by = 3;             // 启动用户ID
  RecordingConfig override_config = 4;  // 临时覆盖配置
}

message StartMotionRecordingResponse {
  Recording recording = 1;
}

// StopMotionRecording
message StopMotionRecordingRequest {
  string nvr_id = 1;
  string camera_id = 2;
}

// StartContinuousRecording
message StartContinuousRecordingRequest {
  string nvr_id = 1;
  string camera_id = 2;
  uint32 started_by = 3;
  int32 duration_seconds = 4;        // 持续时长（0表示无限）
}

message StartContinuousRecordingResponse {
  Recording recording = 1;
}

// StopContinuousRecording
message StopContinuousRecordingRequest {
  string nvr_id = 1;
  string camera_id = 2;
}

// GetRecordingStatus
message GetRecordingStatusRequest {
  string nvr_id = 1;
  string camera_id = 2;
}

message GetRecordingStatusResponse {
  bool is_recording = 1;
  RecordingType recording_type = 2;
  Recording current_recording = 3;
}

enum RecordingType {
  RECORDING_TYPE_UNKNOWN = 0;
  RECORDING_TYPE_MOTION = 1;         // 运动检测录制
  RECORDING_TYPE_CONTINUOUS = 2;     // 持续录制
}

// ListRecordings
message ListRecordingsRequest {
  string nvr_id = 1;
  optional string camera_id = 2;
  google.protobuf.Timestamp start_time = 3;
  google.protobuf.Timestamp end_time = 4;
  optional bool motion_only = 5;     // 仅运动检测录像
  int32 page_size = 6;
  string page_token = 7;
}

message ListRecordingsResponse {
  repeated Recording recordings = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

// GetRecording
message GetRecordingRequest {
  string id = 1;
}

message GetRecordingResponse {
  Recording recording = 1;
  repeated MotionDetectionEvent motion_events = 2;  // 关联的运动事件
}

// DeleteRecording
message DeleteRecordingRequest {
  string id = 1;
}

// DownloadRecording
message DownloadRecordingRequest {
  string id = 1;
  int32 offset = 2;                  // 分片下载偏移量
}

message DownloadRecordingChunk {
  bytes data = 1;
  int64 offset = 2;
  bool is_last = 3;
}

// ============== 事件流相关消息 ==============

// SubscribeEventStream
message SubscribeEventStreamRequest {
  string nvr_id = 1;                 // 订阅特定NVR（可选）
  repeated EventType event_types = 2; // 订阅的事件类型
}

message EventStreamResponse {
  NVREvent event = 1;
}

enum EventType {
  EVENT_TYPE_UNKNOWN = 0;
  EVENT_TYPE_NVR_CREATED = 1;
  EVENT_TYPE_NVR_ONLINE = 2;
  EVENT_TYPE_NVR_OFFLINE = 3;
  EVENT_TYPE_CAMERA_ADDED = 4;
  EVENT_TYPE_RECORDING_STARTED = 5;
  EVENT_TYPE_RECORDING_STOPPED = 6;
  EVENT_TYPE_MOTION_DETECTED = 7;
  EVENT_TYPE_ERROR = 8;
}

// SubscribeCameraEvents
message SubscribeCameraEventsRequest {
  string nvr_id = 1;
  string camera_id = 2;
  repeated CameraEventType event_types = 3;
}

message CameraEvent {
  string nvr_id = 1;
  string camera_id = 2;
  CameraEventType event_type = 3;
  google.protobuf.Timestamp timestamp = 4;
  map<string, string> metadata = 5;
}

enum CameraEventType {
  CAMERA_EVENT_TYPE_UNKNOWN = 0;
  CAMERA_EVENT_TYPE_ONLINE = 1;
  CAMERA_EVENT_TYPE_OFFLINE = 2;
  CAMERA_EVENT_TYPE_RECORDING_STARTED = 3;
  CAMERA_EVENT_TYPE_RECORDING_STOPPED = 4;
  CAMERA_EVENT_TYPE_MOTION_DETECTED = 5;
}

// ============== 辅助消息 ==============

// 录制信息
message RecordingInfo {
  string recording_id = 1;
  RecordingType type = 2;
  google.protobuf.Timestamp started_at = 3;
  int64 duration_seconds = 4;
  int64 bytes_written = 5;
}

// 录制统计
message RecordingStats {
  int32 total_recordings = 1;
  int64 total_duration_seconds = 2;
  int64 total_size_bytes = 3;
  int32 active_recordings = 4;
}

// 系统健康状态
message SystemHealth {
  double cpu_usage_percent = 1;
  double memory_usage_percent = 2;
  int64 disk_free_bytes = 3;
  int64 disk_total_bytes = 4;
  int32 active_streams = 5;
}

// 运动检测事件
message MotionDetectionEvent {
  string event_id = 1;
  string camera_id = 2;
  google.protobuf.Timestamp detected_at = 3;
  double confidence = 4;
  string snapshot_path = 5;
  string region_coordinates = 6;
}

// NVR事件
message NVREvent {
  string event_id = 1;
  EventType type = 2;
  string aggregate_id = 3;          // NVR ID或Camera ID
  google.protobuf.Timestamp occurred_at = 4;
  map<string, string> data = 5;     // 事件数据
}
```

### 2.3 通用定义

```protobuf
// api/grpc/proto/common/error.proto
syntax = "proto3";

package common.v1;
option go_package = "github.com/record_go/api/grpc/proto/common/v1;common_v1";

// 错误响应
message Error {
  string code = 1;                  // 错误码
  string message = 2;               // 错误消息
  map<string, string> details = 3;  // 详细信息
}

// 分页请求
message PageRequest {
  int32 page_size = 1;
  string page_token = 2;
}

// 分页响应
message PageResponse {
  string next_page_token = 1;
  int32 total_count = 2;
}
```

## 三、项目结构

### 3.1 NVR微服务目录结构

```
nvr-service/
├── cmd/
│   └── server/
│       └── main.go                 # 服务入口
│
├── api/
│   ├── grpc/
│   │   └── proto/                  # Protobuf定义
│   │       ├── nvr/
│   │       │   ├── nvr.proto
│   │       │   ├── camera.proto
│   │       │   ├── recording.proto
│   │       │   └── events.proto
│   │       └── common/
│   │           ├── common.proto
│   │           └── error.proto
│   └── http/
│       └── handlers/               # HTTP处理器（可选，用于管理界面）
│
├── internal/
│   ├── domain/                     # 领域层
│   │   ├── nvr/
│   │   │   ├── aggregate.go        # NVR聚合根
│   │   │   ├── value_objects.go    # 值对象
│   │   │   ├── events.go           # 领域事件
│   │   │   ├── repository.go       # 仓储接口
│   │   │   └── errors.go           # 领域错误
│   │   └── camera/
│   │       ├── entity.go           # Camera实体
│   │       └── value_objects.go
│   │
│   ├── application/                # 应用层
│   │   ├── nvr/
│   │   │   ├── command/            # 命令处理器
│   │   │   ├── query/              # 查询处理器
│   │   │   └── dto.go              # DTO定义
│   │   └── grpc/
│   │       ├── server.go           # gRPC服务器
│   │       └── interceptors/       # gRPC拦截器
│   │
│   ├── infrastructure/             # 基础设施层
│   │   ├── persistence/            # 持久化
│   │   │   ├── sqlite/
│   │   │   │   └── repository.go   # SQLite实现
│   │   │   └── models.go           # 数据模型
│   │   ├── rtsp/                   # RTSP适配器
│   │   │   ├── client.go
│   │   │   └── recorder.go
│   │   ├── ffmpeg/                 # FFmpeg适配器
│   │   │   └── orchestrator.go
│   │   ├── onvif/                  # ONVIF适配器
│   │   │   ├── client.go
│   │   │   └── ptz.go
│   │   └── storage/                # 存储管理
│   │       └── manager.go
│   │
│   └── config/                     # 配置
│       └── config.go
│
├── pkg/
│   └── api/                        # 生成的gRPC代码
│
├── configs/
│   └── nvr.yaml                    # 配置文件
│
├── scripts/
│   ├── generate.sh                 # 生成gRPC代码
│   └── build.sh                    # 构建脚本
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

### 3.2 主服务集成

```
record_go/ (主服务)
├── internal/
│   ├── services/
│   │   └── nvr_client/              # NVR服务客户端
│   │       ├── client.go           # gRPC客户端
│   │       ├── nvr.go              # NVR操作封装
│   │       ├── camera.go           # 摄像头操作封装
│   │       └── recording.go        # 录制操作封装
│   │
│   └── interfaces/
│       └── http/
│           └── handlers/
│               └── nvr_proxy.go    # NVR API代理（可选）
│
└── configs/
    └── config.yaml                 # 添加NVR服务地址配置
```

## 四、配置管理

### 4.1 NVR服务配置

```yaml
# configs/nvr.yaml
server:
  name: "nvr-service"
  version: "1.0.0"

  # gRPC服务器
  grpc:
    host: "0.0.0.0"
    port: 9091
    timeout: 30s
    max_connection_age: 1h
    max_connection_idle: 5m
    # TLS配置
    tls:
      enabled: true
      cert_file: "./certs/server.crt"
      key_file: "./certs/server.key"
      ca_file: "./certs/ca.crt"
      client_ca_required: true

  # HTTP服务器（可选，用于健康检查和管理）
  http:
    host: "0.0.0.0"
    port: 8081
    read_timeout: 30s
    write_timeout: 30s
    shutdown_timeout: 10s

  # WebSocket服务器（事件推送）
  websocket:
    host: "0.0.0.0"
    port: 8082
    read_buffer_size: 4096
    write_buffer_size: 4096
    ping_period: 30s
    pong_timeout: 60s

# 数据库配置
database:
  driver: "sqlite"
  source: "./data/nvr.db"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

  # WAL模式（推荐用于SQLite）
  wal_mode: true
  journal_mode: "WAL"
  synchronous: "NORMAL"
  cache_size: 1000

# NVR业务配置
nvr:
  # 并发配置
  max_concurrent_streams: 4
  max_concurrent_recordings: 2

  # 连接配置
  connection:
    timeout: 30s
    retry_interval: 10s
    max_retries: 3
    keepalive_interval: 60s

  # 存储配置
  storage:
    base_path: "./data/recordings"
    deletion_strategy: "smart"
    disk_management:
      enabled: true
      check_interval: 5m
      min_free_space: 10737418240  # 10GB
      warning_threshold: 0.8
    retention:
      default_days: 30
      conference_days: 90
      motion_days: 7

  # 运动检测配置
  motion_detection:
    enabled: true
    default_sensitivity: "medium"
    pre_record_seconds: 3
    post_record_seconds: 5
    analysis_interval: 500ms

  # ONVIF配置
  onvif:
    enabled: true
    discovery_timeout: 5s
    ptz_timeout: 10s
    profile_token: "000"

  # RTSP配置
  rtsp:
    transport: "tcp"
    buffer_size: 1048576  # 1MB
    reconnect_delay: 5s
    max_reconnect_attempts: 5

  # 录制配置
  recording:
    format: "mp4"
    segment_duration: 5m
    video_codec: "h264"
    audio_codec: "aac"
    video_bitrate: "3000k"
    audio_bitrate: "128k"
    preset: "medium"

# FFmpeg配置
ffmpeg:
  path: "ffmpeg"
  process:
    start_timeout: 30s
    stop_timeout: 10s
    kill_timeout: 5s
  buffer:
    input_buffer_size: "1024k"
    output_buffer_size: "1024k"
  log_level: "error"

# 日志配置
logging:
  level: "info"
  format: "json"
  output: "stdout"
  file:
    path: "./logs/nvr.log"
    max_size: 100
    max_backups: 10
    max_age: 30
    compress: true

# 监控配置
monitoring:
  enabled: true
  metrics_port: 9092
  health_check_interval: 30s
  prometheus:
    enabled: true

# 服务发现配置
service_discovery:
  enabled: true
  type: "consul"  # consul, etcd, k8s
  address: "localhost:8500"
  health_check:
    interval: 10s
    timeout: 3s
    deregister_critical_service_after: "30s"
  registration:
    name: "nvr-service"
    tags: ["nvr", "recording"]
    address: ""
    port: 9091
```

### 4.2 主服务配置

```yaml
# configs/config.yaml (主服务)
# ... 其他配置 ...

# NVR服务配置
nvr_service:
  # gRPC客户端配置
  grpc:
    address: "localhost:9091"
    timeout: 30s
    # TLS配置
    tls:
      enabled: true
      cert_file: "./certs/client.crt"
      key_file: "./certs/client.key"
      ca_file: "./certs/ca.crt"
      server_name: "nvr-service"
    # 重试配置
    retry:
      max_attempts: 3
      initial_backoff: 1s
      max_backoff: 10s
      backoff_multiplier: 2.0
    # 连接池
    pool:
      max_connections: 10
      idle_timeout: 5m
      max_idle_connections: 2

  # 服务发现（如果启用）
  service_discovery:
    enabled: true
    type: "consul"
    service_name: "nvr-service"
    refresh_interval: 30s
```

## 五、安全设计

### 5.1 gRPC安全

```go
// internal/application/grpc/server.go
type Server struct {
    server *grpc.Server
}

func NewServer(config *config.Config) (*Server, error) {
    // TLS凭证
    creds, err := credentials.NewServerTLSFromFile(
        config.Server.GRPC.TLS.CertFile,
        config.Server.GRPC.TLS.KeyFile,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create TLS credentials: %w", err)
    }

    // mTLS（双向认证）
    if config.Server.GRPC.TLS.ClientCARequired {
        creds = loadMTLSCredentials(config)
    }

    server := grpc.NewServer(
        grpc.Creds(creds),
        grpc.ChainUnaryInterceptor(
            interceptors.LoggingInterceptor(),
            interceptors.AuthInterceptor(),
            interceptors.RateLimitInterceptor(),
            interceptors.ValidationInterceptor(),
            interceptors.MetricsInterceptor(),
        ),
        grpc.MaxRecvMsgSize(100*1024*1024),  // 100MB
        grpc.MaxSendMsgSize(100*1024*1024),
    )

    return &Server{server: server}, nil
}
```

### 5.2 认证拦截器

```go
// internal/application/grpc/interceptors/auth.go
func AuthInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        // 从metadata获取token
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing metadata")
        }

        tokens := md["authorization"]
        if len(tokens) == 0 {
            return nil, status.Error(codes.Unauthenticated, "missing authorization token")
        }

        // 验证token
        token := strings.TrimPrefix(tokens[0], "Bearer ")
        if err := validateToken(token); err != nil {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }

        // 将用户ID添加到context
        userID := extractUserIDFromToken(token)
        ctx = context.WithValue(ctx, "user_id", userID)

        return handler(ctx, req)
    }
}
```

### 5.3 主服务客户端安全

```go
// internal/services/nvr_client/client.go
type Client struct {
    conn    *grpc.ClientConn
    nvrClient nvr_v1.NVRServiceClient
    cameraClient nvr_v1.CameraServiceClient
    recordingClient nvr_v1.RecordingServiceClient
}

func NewClient(cfg *config.NVRServiceConfig) (*Client, error) {
    // TLS凭证
    creds, err := credentials.NewClientTLSFromFile(
        cfg.GRPC.TLS.CAFile,
        cfg.GRPC.TLS.ServerName,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create TLS credentials: %w", err)
    }

    // 客户端证书（mTLS）
    if cfg.GRPC.TLS.CertFile != "" && cfg.GRPC.TLS.KeyFile != "" {
        cert, err := tls.LoadX509KeyPair(
            cfg.GRPC.TLS.CertFile,
            cfg.GRPC.TLS.KeyFile,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to load client certificate: %w", err)
        }

        creds = credentials.NewTLS(&tls.Config{
            Certificates: []tls.Certificate{cert},
            RootCAs:      loadCACert(cfg.GRPC.TLS.CAFile),
            ServerName:   cfg.GRPC.TLS.ServerName,
        })
    }

    // 连接配置
    dialOpts := []grpc.DialOption{
        grpc.WithTransportCredentials(creds),
        grpc.WithDefaultCallOptions(
            grpc.WaitForReady(true),
            grpc.MaxCallRecvMsgSize(100*1024*1024),
            grpc.MaxCallSendMsgSize(100*1024*1024),
        ),
        grpc.WithChainUnaryInterceptor(
            client_interceptors.AuthInterceptor(),
            client_interceptors.RetryInterceptor(cfg.GRPC.Retry),
            client_interceptors.LoggingInterceptor(),
        ),
    }

    // 建立连接
    conn, err := grpc.Dial(cfg.GRPC.Address, dialOpts...)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to NVR service: %w", err)
    }

    return &Client{
        conn: conn,
        nvrClient: nvr_v1.NewNVRServiceClient(conn),
        cameraClient: nvr_v1.NewCameraServiceClient(conn),
        recordingClient: nvr_v1.NewRecordingServiceClient(conn),
    }, nil
}
```

## 六、部署方案

### 6.1 Docker部署

```dockerfile
# deployments/docker/Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o nvr-service ./cmd/server

FROM alpine:latest
RUN apk add --no-cache ffmpeg ffmpeg-libs ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/nvr-service .
COPY --from=builder /app/api ./api
COPY configs/nvr.yaml ./configs/

EXPOSE 9091 8081 8082 9092

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD grpc_health_probe -addr=:9091 || exit 1

CMD ["./nvr-service"]
```

```yaml
# deployments/docker/docker-compose.yml
version: '3.8'

services:
  # 主服务
  main-service:
    build: ../..
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - NVR_SERVICE_ADDRESS=nvr-service:9091
    depends_on:
      - nvr-service
    networks:
      - app-network

  # NVR服务
  nvr-service:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile
    ports:
      - "9091:9091"   # gRPC
      - "8081:8081"   # HTTP
      - "8082:8082"   # WebSocket
      - "9092:9092"   # Metrics
    volumes:
      - ./data/nvr:/app/data
      - ./configs/nvr.yaml:/app/configs/nvr.yaml:ro
      - ./certs:/app/certs:ro
    environment:
      - CONFIG_FILE=/app/configs/nvr.yaml
    networks:
      - app-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "grpc_health_probe", "-addr=:9091"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s

networks:
  app-network:
    driver: bridge
```

### 6.2 Kubernetes部署

```yaml
# deployments/k8s/nvr-service/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nvr-service
  labels:
    app: nvr-service
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nvr-service
  template:
    metadata:
      labels:
        app: nvr-service
    spec:
      containers:
      - name: nvr-service
        image: nvr-service:latest
        ports:
        - containerPort: 9091
          name: grpc
        - containerPort: 8081
          name: http
        - containerPort: 8082
          name: websocket
        - containerPort: 9092
          name: metrics
        volumeMounts:
        - name: storage
          mountPath: /app/data
        - name: config
          mountPath: /app/configs
        - name: certs
          mountPath: /app/certs
          readOnly: true
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
        env:
        - name: CONFIG_FILE
          value: "/app/configs/nvr.yaml"
        livenessProbe:
          exec:
            command:
            - /app/grpc_health_probe
            - -addr=:9091
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command:
            - /app/grpc_health_probe
            - -addr=:9091
          initialDelaySeconds: 10
          periodSeconds: 5
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: nvr-storage
      - name: config
        configMap:
          name: nvr-config
      - name: certs
        secret:
          secretName: nvr-certs
---
apiVersion: v1
kind: Service
metadata:
  name: nvr-service
spec:
  selector:
    app: nvr-service
  ports:
  - name: grpc
    port: 9091
    targetPort: 9091
  - name: http
    port: 8081
    targetPort: 8081
  - name: websocket
    port: 8082
    targetPort: 8082
  - name: metrics
    port: 9092
    targetPort: 9092
  type: ClusterIP
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: nvr-storage
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 500Gi
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: nvr-config
data:
  nvr.yaml: |
    # NVR服务配置内容
```

## 七、监控与运维

### 7.1 健康检查

```go
// gRPC健康检查实现
import "google.golang.org/grpc/health/grpc_health_v1"

type HealthServer struct {
    grpc_health_v1.UnimplementedHealthServer
    checker func() error
}

func (s *HealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
    if err := s.checker(); err != nil {
        return &grpc_health_v1.HealthCheckResponse{
            Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
        }, nil
    }
    return &grpc_health_v1.HealthCheckResponse{
        Status: grpc_health_v1.HealthCheckResponse_SERVING,
    }, nil
}
```

### 7.2 Prometheus指标

```go
// NVR服务特有指标
var (
    activeStreams = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "nvr_active_streams_total",
            Help: "Number of active RTSP streams",
        },
        []string{"nvr_id"},
    )

    recordingDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "nvr_recording_duration_seconds",
            Help: "Recording duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"nvr_id", "camera_id", "type"},
    )

    motionDetectionEvents = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "nvr_motion_detection_events_total",
            Help: "Total number of motion detection events",
        },
        []string{"nvr_id", "camera_id"},
    )

    grpcRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "nvr_grpc_request_duration_seconds",
            Help: "gRPC request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"service", "method", "status"},
    )
)
```

### 7.3 分布式追踪

```go
// OpenTelemetry集成
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/resource"
    tracesdk "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func initTracer(serviceName, jaegerEndpoint string) error {
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint(jaegerEndpoint),
    ))
    if err != nil {
        return err
    }

    tp := tracesdk.NewTracerProvider(
        tracesdk.WithBatcher(exporter),
        tracesdk.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
    )
    otel.SetTracerProvider(tp)
    return nil
}

// gRPC拦截器添加追踪
func TracingInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        tracer := otel.Tracer("nvr-service")
        ctx, span := tracer.Start(ctx, info.FullMethod)
        defer span.End()

        return handler(ctx, req)
    }
}
```

## 八、相关文档

- [14-NVR模块概述.md](./14-NVR模块概述.md) - NVR模块概述（已更新）
- [15-NVR领域模型设计.md](./15-NVR领域模型设计.md) - 领域模型设计
- [16-NVR架构与实现.md](./16-NVR架构与实现.md) - 架构与实现（已更新）
- [17-NVR配置与运维.md](./17-NVR配置与运维.md) - 配置与运维（已更新）
- [01-系统架构总览.md](./01-系统架构总览.md) - 系统架构（需更新）
- [03-视频录制任务生命周期.md](./03-视频录制任务生命周期.md) - 任务生命周期（需更新）
