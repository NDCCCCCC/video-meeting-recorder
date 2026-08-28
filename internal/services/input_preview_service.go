package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
)

// InputPreviewService 按需抓帧服务（输入配置预览）
type InputPreviewService struct {
	db         *gorm.DB
	logger     *zap.Logger
	ffmpegPath string
	sema       chan struct{}
}

// previewSource 描述一个可预览的媒体源
type previewSource struct {
	kind     string // stream | usb | huawei | lavfi
	protocol string
	url      string
	device   string
	backend  string
	name     string
}

// NewInputPreviewService 创建 InputPreviewService
// ffmpegPath 默认 ./bin/ffmpeg
func NewInputPreviewService(db *gorm.DB, logger *zap.Logger, ffmpegPath string) *InputPreviewService {
	if ffmpegPath == "" {
		ffmpegPath = "./bin/ffmpeg"
	}
	return &InputPreviewService{
		db:         db,
		logger:     logger,
		ffmpegPath: ffmpegPath,
		sema:       make(chan struct{}, 2), // 全局并发上限 2
	}
}

// resolveSource 按优先级解析可预览源：stream > USB > 华为
func (s *InputPreviewService) resolveSource(cfg *models.InputConfig) (previewSource, error) {
	// 优先级：stream > USB > 华为（镜像 buildRecordingInput :543-585）

	// 1) stream
	hasStream := cfg.StreamEnabled && cfg.StreamURL != ""
	if hasStream {
		return previewSource{
			kind:     "stream",
			protocol: cfg.StreamProtocol,
			url:      cfg.StreamURL,
		}, nil
	}

	// 2) USB
	hasUSB := cfg.USBCameraDevice != ""
	if hasUSB {
		backend := cfg.CameraBackend
		if backend == "" {
			backend = "dshow"
		}
		device := cfg.USBCameraDevice
		name := cfg.USBCameraName

		// dshow: 优先用 name，device 作为 fallback（mirror buildUSBVideoArgs :868-872）
		if backend == "dshow" {
			if name != "" {
				device = fmt.Sprintf("video=%s", name)
			} else if !strings.HasPrefix(device, "video=") {
				device = fmt.Sprintf("video=%s", device)
			}
		}
		return previewSource{
			kind:    "usb",
			backend: backend,
			device:  device,
			name:    name,
		}, nil
	}

	// 3) 华为终端
	if cfg.HuaweiEnabled && cfg.Server != "" {
		return previewSource{
			kind: "huawei",
			url:  fmt.Sprintf("rtsp://%s:554/stream", cfg.Server),
		}, nil
	}

	return previewSource{}, fmt.Errorf(
		"输入配置 %d 未配置可预览的源 (stream/usb/huawei): %w",
		cfg.ID, apperrors.ErrInvalidInput,
	)
}

// buildArgs 构建 ffmpeg argv
func (s *InputPreviewService) buildArgs(src previewSource, outputPath string) ([]string, error) {
	var args []string

	switch src.kind {
	case "stream":
		protocol := src.protocol
		if protocol == "" {
			protocol = "rtsp"
		}
		switch protocol {
		case "rtsp":
			args = []string{"-rtsp_transport", "tcp", "-i", src.url}
		case "rtmp", "srt", "hls":
			args = []string{"-i", src.url}
		default:
			return nil, fmt.Errorf("不支持的流媒体协议: %s: %w", protocol, apperrors.ErrInvalidInput)
		}

	case "usb":
		backend := src.backend
		if backend == "" {
			backend = "dshow"
		}
		device := src.device
		// dshow: 如果设备名不以 "video=" 开头则加上（mirror buildUSBVideoArgs :869-872）
		if backend == "dshow" && !strings.HasPrefix(device, "video=") {
			device = "video=" + device
		}
		switch backend {
		case "dshow":
			args = []string{
				"-f", "dshow",
				"-video_size", "1280x720",
				"-framerate", "15",
				"-i", device,
			}
		case "v4l2":
			args = []string{"-f", "v4l2", "-i", src.device}
		case "avfoundation":
			args = []string{"-f", "avfoundation", "-i", src.device}
		default:
			return nil, fmt.Errorf("不支持的摄像头后端: %s: %w", backend, apperrors.ErrInvalidInput)
		}

	case "huawei", "lavfi":
		args = []string{"-i", src.url}

	default:
		return nil, apperrors.ErrInvalidInput
	}

	// 公共输出参数：单帧 + 缩放 640宽 + JPEG quality 5
	args = append(args,
		"-frames:v", "1",
		"-vf", "scale=640:-2",
		"-q:v", "5",
		"-y", outputPath,
	)

	return args, nil
}

// CapturePreview 按 configID 抓取一帧 JPEG
func (s *InputPreviewService) CapturePreview(ctx context.Context, configID uint) ([]byte, error) {
	var cfg models.InputConfig
	if err := s.db.WithContext(ctx).First(&cfg, configID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NotFound("input config", configID)
		}
		return nil, err
	}

	src, err := s.resolveSource(&cfg)
	if err != nil {
		return nil, err
	}

	// 信号量获取（阻塞直到有槽或 ctx 取消）
	select {
	case s.sema <- struct{}{}:
		defer func() { <-s.sema }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return s.capture(ctx, src)
}

// capture 执行一次 ffmpeg 抓帧
func (s *InputPreviewService) capture(ctx context.Context, src previewSource) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tmpFile, err := os.CreateTemp("", "input_preview_*.jpg")
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	args, err := s.buildArgs(src, tmpPath)
	if err != nil {
		return nil, err
	}

	var stderr strings.Builder
	cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("预览抓帧失败: %w: %w, stderr: %s",
			apperrors.ErrFFmpegFailed, err, stderr.String())
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("读取临时文件失败: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("预览抓帧结果为空: %w", apperrors.ErrFFmpegFailed)
	}

	return data, nil
}
