package recorder

import (
	"context"
	"testing"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestNewSimpleRecordingCoordinator 测试协调器创建
func TestNewSimpleRecordingCoordinator(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			RecordingsPath: t.TempDir(),
		},
		FFmpeg: config.FFmpegConfig{
			Path: "ffmpeg",
		},
	}

	coordinator := NewSimpleRecordingCoordinator(logger, cfg)

	assert.NotNil(t, coordinator)
	assert.NotNil(t, coordinator.processes)
	assert.NotNil(t, coordinator.cancelFuncs)
}

// TestBuildRecordingInput 测试录制输入构建
func TestBuildRecordingInput(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			RecordingsPath: t.TempDir(),
		},
		FFmpeg: config.FFmpegConfig{
			Path: "ffmpeg",
		},
	}
	coordinator := NewSimpleRecordingCoordinator(logger, cfg)

	// 测试USB输入
	task := &models.VideoRecordingTask{
		RTSPStreamURL: "",
	}
	inputConfig := &models.InputConfig{
		ConfigType:      models.ConfigTypeUSB,
		CameraBackend:   "dshow",
		USBCameraDevice: "Integrated Camera",
		USBAudioDevice:  "Microphone",
	}

	input := coordinator.buildRecordingInput(task, inputConfig)

	assert.Equal(t, InputSourceUSB, input.Type)
	assert.Equal(t, "Integrated Camera", input.CameraDevice)
	assert.Equal(t, "Microphone", input.AudioDevice)
	assert.True(t, input.hasAudio)
}

// TestBuildRecordingInput_RTSP 测试RTSP输入构建
func TestBuildRecordingInput_RTSP(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			RecordingsPath: t.TempDir(),
		},
		FFmpeg: config.FFmpegConfig{
			Path: "ffmpeg",
		},
	}
	coordinator := NewSimpleRecordingCoordinator(logger, cfg)

	// 测试RTSP输入
	task := &models.VideoRecordingTask{
		RTSPStreamURL: "rtsp://192.168.1.100:554/stream",
	}
	inputConfig := &models.InputConfig{}

	input := coordinator.buildRecordingInput(task, inputConfig)

	assert.Equal(t, InputSourceRTSP, input.Type)
	assert.Equal(t, "rtsp://192.168.1.100:554/stream", input.RTSPURL)
	assert.True(t, input.hasAudio)
}

// TestBuildRecordingInput_Mixed 测试混合输入构建
func TestBuildRecordingInput_Mixed(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			RecordingsPath: t.TempDir(),
		},
		FFmpeg: config.FFmpegConfig{
			Path: "ffmpeg",
		},
	}
	coordinator := NewSimpleRecordingCoordinator(logger, cfg)

	// 测试混合输入
	task := &models.VideoRecordingTask{
		RTSPStreamURL: "rtsp://192.168.1.100:554/stream",
	}
	inputConfig := &models.InputConfig{
		USBAudioDevice: "Microphone",
	}

	input := coordinator.buildRecordingInput(task, inputConfig)

	assert.Equal(t, InputSourceMixed, input.Type)
	assert.Equal(t, "rtsp://192.168.1.100:554/stream", input.RTSPURL)
	assert.Equal(t, "Microphone", input.AudioDevice)
	assert.True(t, input.hasAudio)
}

// TestBuildUSBVideoArgs 测试USB视频参数构建
func TestBuildUSBVideoArgs(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			RecordingsPath: t.TempDir(),
		},
		FFmpeg: config.FFmpegConfig{
			Path: "ffmpeg",
		},
	}
	coordinator := NewSimpleRecordingCoordinator(logger, cfg)

	tests := []struct {
		name    string
		input   RecordingInput
		wantErr bool
	}{
		{
			name: "dshow后端",
			input: RecordingInput{
				CameraBackend: "dshow",
				CameraDevice:  "Integrated Camera",
			},
			wantErr: false,
		},
		{
			name: "v4l2后端",
			input: RecordingInput{
				CameraBackend: "v4l2",
				CameraDevice:  "/dev/video0",
			},
			wantErr: false,
		},
		{
			name: "不支持的后端",
			input: RecordingInput{
				CameraBackend: "unknown",
				CameraDevice:  "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := coordinator.buildUSBVideoArgs(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, args)
				// 验证参数包含必要的元素
				assert.Contains(t, args, "-f")
				assert.Contains(t, args, tt.input.CameraBackend)
				assert.Contains(t, args, "-i")
			}
		})
	}
}

// TestBuildUSBAudioArgs 测试USB音频参数构建
func TestBuildUSBAudioArgs(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			RecordingsPath: t.TempDir(),
		},
		FFmpeg: config.FFmpegConfig{
			Path: "ffmpeg",
		},
	}
	coordinator := NewSimpleRecordingCoordinator(logger, cfg)

	tests := []struct {
		name    string
		input   RecordingInput
		wantErr bool
	}{
		{
			name: "dshow后端",
			input: RecordingInput{
				AudioBackend: "dshow",
				AudioDevice:  "Microphone",
			},
			wantErr: false,
		},
		{
			name: "alsa后端",
			input: RecordingInput{
				AudioBackend: "alsa",
				AudioDevice:  "hw:0,0",
			},
			wantErr: false,
		},
		{
			name: "不支持的后端",
			input: RecordingInput{
				AudioBackend: "unknown",
				AudioDevice:  "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := coordinator.buildUSBAudioArgs(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, args)
				assert.Contains(t, args, "-f")
				assert.Contains(t, args, tt.input.AudioBackend)
				assert.Contains(t, args, "-i")
			}
		})
	}
}

// TestBuildRTSPArgs 测试RTSP参数构建
func TestBuildRTSPArgs(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			RecordingsPath: t.TempDir(),
		},
		FFmpeg: config.FFmpegConfig{
			Path: "ffmpeg",
		},
	}
	coordinator := NewSimpleRecordingCoordinator(logger, cfg)

	t.Run("有效的RTSP URL", func(t *testing.T) {
		input := RecordingInput{
			RTSPURL: "rtsp://192.168.1.100:554/stream",
		}
		args, err := coordinator.buildRTSPArgs(input)
		assert.NoError(t, err)
		assert.Equal(t, []string{"-rtsp_transport", "tcp", "-i", "rtsp://192.168.1.100:554/stream"}, args)
	})

	t.Run("空的RTSP URL", func(t *testing.T) {
		input := RecordingInput{
			RTSPURL: "",
		}
		_, err := coordinator.buildRTSPArgs(input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "RTSP URL不能为空")
	})
}

// TestGetOutputPath 测试输出路径生成
func TestGetOutputPath(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			RecordingsPath: t.TempDir(),
		},
		FFmpeg: config.FFmpegConfig{
			Path: "ffmpeg",
		},
	}
	coordinator := NewSimpleRecordingCoordinator(logger, cfg)

	task := &models.VideoRecordingTask{
		Base:             models.Base{ID: 123},
		Name:             "TestTask",
		ConferenceNumber: "123456",
	}
	outputPath := coordinator.getOutputPath(task, "mp4")

	assert.Contains(t, outputPath, "task_123")
	assert.Contains(t, outputPath, ".mp4")
	assert.Contains(t, outputPath, "TestTask")
}

// TestRecordingStatus 测试录制状态管理
func TestRecordingStatus(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			RecordingsPath: tempDir,
		},
		FFmpeg: config.FFmpegConfig{
			Path: "ffmpeg",
		},
	}
	coordinator := NewSimpleRecordingCoordinator(logger, cfg)

	// 初始状态：任务不存在
	status, err := coordinator.GetRecordingStatus(999)
	assert.Error(t, err)
	assert.Equal(t, "", status)

	// 添加模拟进程（不实际启动FFmpeg）
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	coordinator.processes["1_usb"] = &RecordingProcess{
		TaskID:     1,
		Cmd:        nil,
		StartTime:  time.Now(),
		OutputPath: tempDir + "/test.mp4",
		Status:     "running",
		CancelFunc: cancel,
		ConfigType: "usb",
	}

	status, err = coordinator.GetRecordingStatus(1)
	assert.NoError(t, err)
	assert.Equal(t, "running", status)
}

// TestHealthCheck 测试健康检查
func TestHealthCheck(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			RecordingsPath: tempDir,
		},
		FFmpeg: config.FFmpegConfig{
			Path: "ffmpeg",
		},
	}
	coordinator := NewSimpleRecordingCoordinator(logger, cfg)

	// 空协调器应该通过健康检查
	err := coordinator.HealthCheck()
	assert.NoError(t, err)
}

func TestAttemptReconnectReturnsImmediatelyWhenContextCanceled(t *testing.T) {
	coordinator := &SimpleRecordingCoordinator{logger: zap.NewNop()}
	process := &RecordingProcess{ReconnectDelay: 10 * time.Second, MaxReconnects: 1}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	coordinator.attemptReconnect(ctx, "test", process)
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("canceled reconnect blocked for %v", elapsed)
	}
}
