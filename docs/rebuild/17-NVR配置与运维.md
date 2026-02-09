# NVR配置与运维

## 一、配置管理

### 1.1 NVR服务配置文件

NVR微服务使用独立的配置文件 `configs/nvr.yaml`：

```yaml
# configs/nvr.yaml - NVR微服务配置
server:
  name: "nvr-service"
  version: "1.0.0"

  # gRPC服务器配置
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
      client_ca_required: true  # mTLS双向认证

  # HTTP服务器（可选，用于健康检查）
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

# 数据库配置（NVR服务独立数据库）
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

### 1.2 主服务配置

主服务需要添加NVR服务客户端配置：

```yaml
# configs/config.yaml - 主服务配置
# ... 其他配置 ...

# NVR服务客户端配置
nvr_service:
  # gRPC客户端配置
  grpc:
    address: "localhost:9091"  # NVR服务地址
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

### 1.3 环境变量

NVR服务支持的环境变量：

| 环境变量 | 描述 | 默认值 |
|---------|------|--------|
| `NVR_SERVER_GRPC_PORT` | gRPC端口 | `9091` |
| `NVR_SERVER_HTTP_PORT` | HTTP端口 | `8081` |
| `NVR_SERVER_WS_PORT` | WebSocket端口 | `8082` |
| `NVR_STORAGE_PATH` | 存储路径 | `./data/recordings` |
| `NVR_DB_PATH` | 数据库路径 | `./data/nvr.db` |
| `NVR_MAX_STREAMS` | 最大并发流数 | `4` |
| `NVR_LOG_LEVEL` | 日志级别 | `info` |
| `NVR_SERVICE_DISCOVERY` | 启用服务发现 | `false` |
| `NVR_TLS_ENABLED` | 启用TLS | `true` |

主服务客户端环境变量：

| 环境变量 | 描述 | 默认值 |
|---------|------|--------|
| `NVR_SERVICE_ADDRESS` | NVR服务地址 | `localhost:9091` |
| `NVR_SERVICE_TLS_ENABLED` | 启用TLS | `true` |

### 1.4 配置验证

启动时NVR服务会自动验证配置：

```go
// internal/config/config.go
func (c *Config) Validate() error {
    // 验证服务器配置
    if c.Server.GRPC.Port < 1 || c.Server.GRPC.Port > 65535 {
        return fmt.Errorf("invalid gRPC port: %d", c.Server.GRPC.Port)
    }

    // 验证TLS配置
    if c.Server.GRPC.TLS.Enabled {
        if c.Server.GRPC.TLS.CertFile == "" {
            return fmt.Errorf("TLS enabled but cert_file not specified")
        }
        if c.Server.GRPC.TLS.KeyFile == "" {
            return fmt.Errorf("TLS enabled but key_file not specified")
        }
    }

    // 验证NVR配置
    if c.NVR.MaxConcurrentStreams < 1 || c.NVR.MaxConcurrentStreams > 16 {
        return fmt.Errorf("max_concurrent_streams must be between 1-16")
    }

    if c.NVR.Storage.BasePath == "" {
        return fmt.Errorf("storage.base_path cannot be empty")
    }

    if c.NVR.Storage.DiskManagement.MinFreeSpace < 1073741824 {
        return fmt.Errorf("min_free_space cannot be less than 1GB")
    }

    return nil
}
```

## 二、部署指南

### 2.1 系统要求

#### 硬件要求

| 组件 | 最低配置 | 推荐配置 |
|------|---------|---------|
| CPU | 4核心 | 8核心+ |
| 内存 | 8GB | 16GB+ |
| 存储 | 100GB SSD | 500GB+ NVMe SSD |
| 网络 | 100Mbps | 1Gbps |

#### 软件要求

| 软件 | 版本要求 |
|------|---------|
| 操作系统 | Windows 10+, Linux (Ubuntu 20.04+), macOS 11+ |
| FFmpeg | 4.0+ |
| Go | 1.21+ |
| SQLite | 3.35+ |
| Protocol Buffers | 3.0+ |

### 2.2 本地开发部署

```bash
# 克隆NVR服务仓库
git clone https://github.com/yourorg/nvr-service.git
cd nvr-service

# 安装依赖
go mod download

# 生成gRPC代码
make generate

# 创建配置文件
cp configs/nvr.yaml.example configs/nvr.yaml

# 编辑配置
vim configs/nvr.yaml

# 运行服务
go run cmd/server/main.go --config configs/nvr.yaml
```

### 2.3 Docker部署

#### Dockerfile

```dockerfile
# deployments/docker/Dockerfile
FROM golang:1.21-alpine AS builder

# 安装Protocol Buffers编译器
RUN apk add --no-cache protobuf-dev make git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make generate
RUN go build -o nvr-service ./cmd/server

FROM alpine:latest
RUN apk add --no-cache \
    ffmpeg \
    ffmpeg-libs \
    ca-certificates \
    tzdata

WORKDIR /app
COPY --from=builder /app/nvr-service .
COPY --from=builder /app/api ./api
COPY configs/nvr.yaml ./configs/

# 创建必要目录
RUN mkdir -p data/recordings logs certs

EXPOSE 9091 8081 8082 9092

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD grpc_health_probe -addr=:9091 || exit 1

CMD ["./nvr-service"]
```

#### docker-compose.yml

```yaml
# deployments/docker/docker-compose.yml
version: '3.8'

services:
  # 主服务
  main-service:
    build:
      context: ../../
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - NVR_SERVICE_ADDRESS=nvr-service:9091
      - NVR_SERVICE_TLS_ENABLED=true
    depends_on:
      - nvr-service
    networks:
      - app-network
    volumes:
      - ./certs:/app/certs:ro

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
      - ./logs/nvr:/app/logs
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

#### 部署命令

```bash
# 构建镜像
docker-compose build

# 生成证书（用于mTLS）
mkdir -p certs
openssl req -x509 -newkey rsa:4096 -keyout certs/ca.key -out certs/ca.crt -days 365 -nodes
openssl req -newkey rsa:4096 -keyout certs/server.key -out certs/server.csr -subj "/CN=nvr-service"
openssl x509 -req -in certs/server.csr -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/server.crt -days 365
openssl req -newkey rsa:4096 -keyout certs/client.key -out certs/client.csr -subj "/CN=main-service"
openssl x509 -req -in certs/client.csr -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/client.crt -days 365

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f nvr-service

# 停止服务
docker-compose down
```

### 2.4 Kubernetes部署

#### Deployment

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
        env:
        - name: CONFIG_FILE
          value: "/app/configs/nvr.yaml"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
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
    server:
      grpc:
        host: "0.0.0.0"
        port: 9091
      http:
        host: "0.0.0.0"
        port: 8081
    nvr:
      storage:
        base_path: "/app/data/recordings"
---
apiVersion: v1
kind: Secret
metadata:
  name: nvr-certs
type: Opaque
data:
  # Base64编码的证书内容
  # ca.crt: <base64>
  # server.crt: <base64>
  # server.key: <base64>
```

#### 部署命令

```bash
# 创建命名空间
kubectl create namespace nvr-system

# 部署NVR服务
kubectl apply -f deployments/k8s/nvr-service/ -n nvr-system

# 查看状态
kubectl get pods -n nvr-system
kubectl get svc -n nvr-system

# 查看日志
kubectl logs -f deployment/nvr-service -n nvr-system
```

## 三、监控与运维

### 3.1 健康检查

#### gRPC健康检查

```bash
# 使用grpc_health_probe工具
grpc_health_probe -addr=localhost:9091

# 响应
status: SERVING

# 检查特定服务
grpc_health_probe -addr=localhost:9091 -service nvr.v1.NVRService
```

#### HTTP健康检查（备用）

```bash
# 基本健康检查
curl http://localhost:8081/health

# 响应示例
{
  "status": "healthy",
  "timestamp": "2026-02-09T10:30:00Z",
  "components": {
    "grpc_server": "ok",
    "database": "ok",
    "storage": "ok",
    "rtsp_adapters": "ok"
  },
  "metrics": {
    "active_streams": 2,
    "total_recordings": 145,
    "disk_usage": {
      "total": 107374182400,
      "used": 53687091200,
      "available": 53687091200,
      "percentage": 50
    }
  }
}
```

### 3.2 Prometheus指标

#### NVR服务特有指标

```bash
curl http://localhost:9092/metrics
```

#### 关键指标

| 指标名称 | 类型 | 标签 | 描述 |
|---------|------|------|------|
| `nvr_active_streams_total` | Gauge | nvr_id | 当前活动流数量 |
| `nvr_total_recordings` | Counter | - | 总录像数 |
| `nvr_recording_duration_seconds` | Histogram | nvr_id,camera_id,type | 录制时长分布 |
| `nvr_storage_bytes` | Gauge | - | 存储空间使用 |
| `nvr_motion_detections_total` | Counter | nvr_id,camera_id | 运动检测触发次数 |
| `nvr_stream_errors_total` | Counter | nvr_id,camera_id | 流错误总数 |
| `nvr_onvif_requests_duration_seconds` | Histogram | method,status | ONVIF请求时长 |
| `nvr_grpc_request_duration_seconds` | Histogram | service,method,status | gRPC请求时长 |
| `nvr_grpc_requests_total` | Counter | service,method,status | gRPC请求总数 |

### 3.3 日志管理

#### 结构化日志示例

```json
{
  "level": "info",
  "timestamp": "2026-02-09T10:30:00Z",
  "service": "nvr-service",
  "module": "recording",
  "message": "Recording started",
  "trace_id": "abc123",
  "nvr_id": "nvr-001",
  "camera_id": "cam-001",
  "recording_id": "rec-12345",
  "stream_url": "rtsp://192.168.1.100:554/stream",
  "caller": "recording_service.go:245"
}
```

### 3.4 分布式追踪

```yaml
# configs/nvr.yaml
monitoring:
  tracing:
    enabled: true
    exporter: "jaeger"  # jaeger, zipkin, otel
    endpoint: "http://jaeger:14268/api/traces"
    sample_rate: 0.1  # 10%采样
```

## 四、故障排除

### 4.1 常见问题

#### gRPC连接失败

**症状**: 主服务无法连接到NVR服务

**可能原因**:
1. NVR服务未启动
2. 网络不可达
3. TLS证书问题
4. 防火墙阻止

**解决方案**:

```bash
# 1. 检查NVR服务状态
systemctl status nvr-service

# 2. 测试网络连通性
telnet localhost 9091

# 3. 检查TLS证书
openssl s_client -connect localhost:9091 -showcerts

# 4. 查看NVR服务日志
tail -f logs/nvr.log

# 5. 测试gRPC连接
grpcurl -plaintext localhost:9091 list
```

#### 录像文件损坏

**症状**: 录像文件无法播放

**解决方案**:

```bash
# 1. 检查录像文件完整性
ffmpeg -v error -i recording.mp4 -f null -

# 2. 检查磁盘空间
df -h /data/nvr

# 3. 检查FFmpeg进程
ps aux | grep ffmpeg

# 4. 查看错误日志
tail -f logs/nvr.log | grep "error"

# 5. 尝试修复录像
ffmpeg -i corrupt.mp4 -c copy repaired.mp4
```

### 4.2 性能优化

#### gRPC性能优化

```yaml
# configs/nvr.yaml
server:
  grpc:
    max_connection_age: 1h           # 连接最大存活时间
    max_connection_idle: 5m           # 空闲连接超时
    max_recv_msg_size: 100MB          # 最大接收消息
    max_send_msg_size: 100MB          # 最大发送消息
    initial_window_size: 1MB
    initial_conn_window_size: 1MB
```

#### 录制性能优化

```yaml
nvr:
  rtsp:
    transport: "tcp"                  # 使用TCP更稳定
    buffer_size: 2097152              # 增加缓冲区到2MB
  recording:
    video_bitrate: "2000k"            # 降低比特率
    preset: "faster"                  # 使用更快的预设
```

## 五、维护操作

### 5.1 日常维护

#### 每日检查脚本

```bash
#!/bin/bash
# nvr-daily-check.sh

echo "=== NVR Service Daily Health Check ==="

# 1. 检查服务状态
if systemctl is-active --quiet nvr-service; then
    echo "[OK] NVR service is running"
else
    echo "[FAIL] NVR service is not running"
    systemctl start nvr-service
fi

# 2. gRPC健康检查
if grpc_health_probe -addr=localhost:9091 > /dev/null 2>&1; then
    echo "[OK] gRPC server is healthy"
else
    echo "[FAIL] gRPC health check failed"
fi

# 3. 检查磁盘空间
DISK_USAGE=$(df /data/nvr | awk 'NR==2 {print $5}' | sed 's/%//')
if [ $DISK_USAGE -gt 80 ]; then
    echo "[WARN] Disk usage is ${DISK_USAGE}%"
fi

# 4. 检查活动流
ACTIVE_STREAMS=$(curl -s http://localhost:9092/metrics | grep nvr_active_streams_total | awk '{print $2}')
echo "[INFO] Active streams: $ACTIVE_STREAMS"
```

### 5.2 备份与恢复

#### 配置备份

```bash
#!/bin/bash
# nvr-backup.sh

BACKUP_DIR="/backup/nvr/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

# 备份配置
cp /etc/nvr/nvr.yaml "$BACKUP_DIR/"

# 备份数据库
cp /var/lib/nvr/nvr.db "$BACKUP_DIR/"

# 备份证书
cp -r /etc/nvr/certs "$BACKUP_DIR/"

# 备份录像清单（通过gRPC）
grpcurl -plaintext localhost:9091 nvr.v1.RecordingService/ListRecordings \
  > "$BACKUP_DIR/recordings.json"

echo "Backup completed: $BACKUP_DIR"
```

### 5.3 升级流程

```bash
#!/bin/bash
# nvr-upgrade.sh

VERSION=$1

echo "Upgrading NVR service to version $VERSION..."

# 1. 备份当前版本
./nvr-backup.sh

# 2. 停止服务
systemctl stop nvr-service

# 3. 备份当前二进制
cp /usr/local/bin/nvr-service /usr/local/bin/nvr-service.old

# 4. 下载新版本
wget https://github.com/yourorg/nvr-service/releases/download/$VERSION/nvr-service-linux-amd64.tar.gz

# 5. 解压并替换
tar -xzf nvr-service-linux-amd64.tar.gz
mv nvr-service /usr/local/bin/nvr-service

# 6. 运行数据库迁移（如果需要）
/usr/local/bin/nvr-service --migrate-db

# 7. 启动服务
systemctl start nvr-service

# 8. 验证升级
sleep 10
if systemctl is-active --quiet nvr-service; then
    echo "Upgrade successful!"
    /usr/local/bin/nvr-service --version
else
    echo "Upgrade failed, rolling back..."
    systemctl stop nvr-service
    mv /usr/local/bin/nvr-service.old /usr/local/bin/nvr-service
    systemctl start nvr-service
fi
```

## 六、安全最佳实践

### 6.1 mTLS双向认证

```yaml
# configs/nvr.yaml
server:
  grpc:
    tls:
      enabled: true
      cert_file: "./certs/server.crt"
      key_file: "./certs/server.key"
      ca_file: "./certs/ca.crt"
      client_ca_required: true  # 要求客户端证书
```

```yaml
# 主服务配置
nvr_service:
  grpc:
    tls:
      enabled: true
      cert_file: "./certs/client.crt"
      key_file: "./certs/client.key"
      ca_file: "./certs/ca.crt"
      server_name: "nvr-service"
```

### 6.2 认证拦截器

NVR服务的gRPC认证拦截器验证每个请求：

```go
// internal/application/grpc/interceptors/auth.go
func AuthInterceptor(validator TokenValidator) grpc.UnaryServerInterceptor {
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
        if err := validator.Validate(token); err != nil {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }

        return handler(ctx, req)
    }
}
```

## 七、相关文档

- [14-NVR模块概述.md](./14-NVR模块概述.md) - NVR模块概述
- [15-NVR领域模型设计.md](./15-NVR领域模型设计.md) - 领域模型设计
- [16-NVR架构与实现.md](./16-NVR架构与实现.md) - 架构与实现
- [18-NVR微服务架构设计.md](./18-NVR微服务架构设计.md) - 微服务架构设计
- [07-RTSP流媒体处理.md](./07-RTSP流媒体处理.md) - RTSP流媒体处理
- [10-配置管理详解.md](./10-配置管理详解.md) - 全局配置管理
- [11-错误处理与日志.md](./11-错误处理与日志.md) - 错误处理和日志
- [12-监控与可观测性.md](./12-监控与可观测性.md) - 监控和可观测性
