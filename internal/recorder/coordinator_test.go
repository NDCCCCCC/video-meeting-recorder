package recorder

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
)

func TestRecorderCoordinator_WatcherChannels(t *testing.T) {
	// Phase 25 智能退出撤回后,WatcherChannels 函数体仅返回空切片(无 taskEndedCh
	// 字段)。保留此测试用于验证接口契约:空切片即可,数量为 0。
	c := NewSimpleRecordingCoordinator(zap.NewNop(), &config.Config{})
	c.processes["1_usb"] = &RecordingProcess{}
	c.processes["1_stream"] = &RecordingProcess{}
	c.processes["2_usb"] = &RecordingProcess{}

	assert.Empty(t, c.WatcherChannels(1))
	assert.Empty(t, c.WatcherChannels(2))
	assert.NotNil(t, c.WatcherChannels(99))
	assert.Empty(t, c.WatcherChannels(99))
}

// TestRecorderCoordinator_IsProcessAlive 验证 Phase 25 撤回后的 ffmpeg 进程活跃
// 判断逻辑:taskID 命中索引 + ProcessState == nil 返回 true;否则 false。
func TestRecorderCoordinator_IsProcessAlive(t *testing.T) {
	c := NewSimpleRecordingCoordinator(zap.NewNop(), &config.Config{})

	// 1. 完全空状态
	assert.False(t, c.IsProcessAlive(1), "无索引应返回 false")

	// 2. 索引存在但 processes 不存在
	c.taskIDProcessKeyIndex[1] = "1_usb"
	assert.False(t, c.IsProcessAlive(1), "processes 不存在应返回 false")

	// 3. 进程存在但 Cmd 为 nil
	c.processes["1_usb"] = &RecordingProcess{}
	assert.False(t, c.IsProcessAlive(1), "Cmd 为 nil 应返回 false")

	// 4. 进程存在,Cmd 存在,ProcessState == nil (存活)
	c.processes["1_usb"] = &RecordingProcess{Cmd: &exec.Cmd{}}
	assert.True(t, c.IsProcessAlive(1), "Cmd 存在且 ProcessState == nil 应返回 true")
}

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

// TestBuildRecordingCommand_HLSListSizeValidation 验证 PERF-011 修复：
// 当 cfg.FFmpeg.HLSListSize 为 0 或负数时，hlsDeleteThreshold fallback 到 6。
func TestBuildRecordingCommand_HLSListSizeValidation(t *testing.T) {
	cases := []struct {
		name          string
		hlsListSize   int
		wantListSize  int
		wantDeleteThr int
	}{
		{"zero_falls_back_to_6", 0, 6, 7},
		{"negative_falls_back_to_6", -3, 6, 7},
		{"positive_unchanged", 10, 10, 11},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger := zap.NewNop()
			cfg := &config.Config{
				Storage: config.StorageConfig{
					RecordingsPath: t.TempDir(),
					HLSPath:        t.TempDir(),
				},
				FFmpeg: config.FFmpegConfig{
					Path:               "ffmpeg",
					HLSSegmentDuration: 10,
					HLSListSize:        tc.hlsListSize,
					CRF:                23,
					Preset:             "medium",
					MaxVideoBitrate:    "3M",
					VideoBufSize:       "6M",
				},
			}
			c := NewSimpleRecordingCoordinator(logger, cfg)
			input := RecordingInput{
				Type:          InputSourceRTSP,
				RTSPURL:       "rtsp://example.com/stream",
				CameraBackend: "dshow",
			}
			args, err := c.buildRecordingCommand(input, "out.mkv", "hls_dir", time.Minute, 1)
			if err != nil {
				t.Fatalf("buildRecordingCommand: %v", err)
			}
			// 拼接 args，找 hls_list_size=N 与 hls_delete_threshold=N
			joined := ""
			for _, a := range args {
				joined += a + " "
			}
			expectedList := fmt.Sprintf("hls_list_size=%d", tc.wantListSize)
			expectedDel := fmt.Sprintf("hls_delete_threshold=%d", tc.wantDeleteThr)
			if !strings.Contains(joined, expectedList) {
				t.Errorf("args 缺 %q: %s", expectedList, joined)
			}
			if !strings.Contains(joined, expectedDel) {
				t.Errorf("args 缺 %q: %s", expectedDel, joined)
			}
		})
	}
}

// TestRecordingProcess_ReconnectCountAtomic 验证 PERF-010 修复：
// ReconnectCount 改 atomic.Int32，多 goroutine 并发增 1 无 race。
func TestRecordingProcess_ReconnectCountAtomic(t *testing.T) {
	process := &RecordingProcess{
		MaxReconnects:  100,
		ReconnectDelay: 10 * time.Millisecond,
	}

	const goroutines = 50
	done := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func() {
			process.ReconnectCount.Add(1)
		}()
	}
	// 等待完成
	go func() {
		// busy wait 通过原子读
		for process.ReconnectCount.Load() < goroutines {
			time.Sleep(time.Millisecond)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("ReconnectCount 在 2s 内未达到 %d（实际 %d）", goroutines, process.ReconnectCount.Load())
	}
}

// TestBuildRecordingCommand_SilenceDetect 撤回 (Phase 25 智能退出撤回后,不再注入
// -af silencedetect 过滤器。ActivityWatcher 整文件连带 silence_parser.go 删除,
// SilenceDB / SilenceDurationS 字段也从 SmartEndConfig 删除)。

// TestRestartRecording_OnReconnect 撤回 (Phase 25 智能退出撤回后,ActivityWatcher
// 删除连带 OnReconnect 字段删除。重连路径保留 attemptReconnect,仅 watcher 通知
// 钩子不再需要。)。
