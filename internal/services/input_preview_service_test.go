package services

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
)

// Test 1: Source resolution priority
func TestResolvePreviewSource(t *testing.T) {
	db := openInMemoryDB(t)
	svc := NewInputPreviewService(db, zap.NewNop(), "./bin/ffmpeg")

	tests := []struct {
		name       string
		cfg        models.InputConfig
		wantKind   string
		wantURL    string
		wantDevice string
		wantBack   string
		wantName   string
		wantErr    bool
	}{
		{
			name: "stream优先",
			cfg: models.InputConfig{
				StreamEnabled:   true,
				StreamURL:       "rtsp://192.168.1.100/live",
				StreamProtocol:  "rtsp",
				USBCameraDevice: "video=Cam",
			},
			wantKind: "stream",
		},
		{
			name: "仅USB",
			cfg: models.InputConfig{
				USBCameraDevice: "video=Logitech Camera",
				USBCameraName:   "Logitech Camera",
				CameraBackend:   "dshow",
			},
			wantKind:   "usb",
			wantDevice: "video=Logitech Camera",
			wantBack:   "dshow",
			wantName:   "Logitech Camera",
		},
		{
			name: "华为终端",
			cfg: models.InputConfig{
				HuaweiEnabled: true,
				Server:        "10.0.0.5",
			},
			wantKind: "huawei",
			wantURL:  "rtsp://10.0.0.5:554/stream",
		},
		{
			name:    "三源皆无",
			cfg:     models.InputConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := svc.resolveSource(&tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantKind, src.kind)
			if tt.wantURL != "" {
				assert.Equal(t, tt.wantURL, src.url)
			}
			if tt.wantDevice != "" {
				assert.Equal(t, tt.wantDevice, src.device)
			}
			if tt.wantBack != "" {
				assert.Equal(t, tt.wantBack, src.backend)
			}
			if tt.wantName != "" {
				assert.Equal(t, tt.wantName, src.name)
			}
		})
	}
}

// Test 2: argv construction
func TestBuildPreviewArgs(t *testing.T) {
	db := openInMemoryDB(t)
	svc := NewInputPreviewService(db, zap.NewNop(), "./bin/ffmpeg")

	tests := []struct {
		name     string
		src      previewSource
		wantArgs []string
		wantErr  bool
	}{
		{
			name:     "rtsp",
			src:      previewSource{kind: "stream", protocol: "rtsp", url: "rtsp://x.live"},
			wantArgs: []string{"-rtsp_transport", "tcp", "-i", "rtsp://x.live"},
		},
		{
			name:     "rtmp",
			src:      previewSource{kind: "stream", protocol: "rtmp", url: "rtmp://x.live"},
			wantArgs: []string{"-i", "rtmp://x.live"},
		},
		{
			name:     "srt",
			src:      previewSource{kind: "stream", protocol: "srt", url: "srt://x.live"},
			wantArgs: []string{"-i", "srt://x.live"},
		},
		{
			name:     "hls",
			src:      previewSource{kind: "stream", protocol: "hls", url: "http://x.live/playlist.m3u8"},
			wantArgs: []string{"-i", "http://x.live/playlist.m3u8"},
		},
		{
			name:    "未知协议",
			src:     previewSource{kind: "stream", protocol: "ftp", url: "ftp://x"},
			wantErr: true,
		},
		{
			name:     "dshow带名称",
			src:      previewSource{kind: "usb", backend: "dshow", name: "Logitech Cam", device: "video=Logitech Cam"},
			wantArgs: []string{"-f", "dshow", "-video_size", "1280x720", "-framerate", "15", "-i", "video=Logitech Cam"},
		},
		{
			name:     "dshow仅device",
			src:      previewSource{kind: "usb", backend: "dshow", name: "", device: "Integrated Camera"},
			wantArgs: []string{"-f", "dshow", "-video_size", "1280x720", "-framerate", "15", "-i", "video=Integrated Camera"},
		},
		{
			name:     "v4l2",
			src:      previewSource{kind: "usb", backend: "v4l2", device: "/dev/video0"},
			wantArgs: []string{"-f", "v4l2", "-i", "/dev/video0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := svc.buildArgs(tt.src, "/tmp/out.jpg")
			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantArgs, args[:len(tt.wantArgs)])
			// verify tail
			assert.Contains(t, args, "-frames:v")
			assert.Contains(t, args, "1")
			assert.Contains(t, args, "-vf")
			assert.Contains(t, args, "scale=640:-2")
			assert.Contains(t, args, "-q:v")
			assert.Contains(t, args, "5")
			assert.Contains(t, args, "-y")
			assert.Contains(t, args, "/tmp/out.jpg")
		})
	}
}

// Test 3: config not found
func TestCapturePreview_ConfigNotFound(t *testing.T) {
	db := openInMemoryDB(t)
	svc := NewInputPreviewService(db, zap.NewNop(), "./bin/ffmpeg")

	_, err := svc.CapturePreview(context.Background(), 9999)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))
}

// Test 4: no preview source
func TestCapturePreview_NoSource(t *testing.T) {
	db := openInMemoryDB(t)
	svc := NewInputPreviewService(db, zap.NewNop(), "./bin/ffmpeg")

	cfg := &models.InputConfig{Name: "empty", ConfigType: "stream"}
	require.NoError(t, db.Create(cfg).Error)

	_, err := svc.CapturePreview(context.Background(), cfg.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInvalidInput))
}

// Test 5: lavfi real ffmpeg produces JPEG
func TestCaptureffmpeg_ProducesJPEG(t *testing.T) {
	if _, err := os.Stat("./bin/ffmpeg"); os.IsNotExist(err) {
		t.Skip("./bin/ffmpeg not found, skipping real ffmpeg test")
	}

	db := openInMemoryDB(t)
	svc := NewInputPreviewService(db, zap.NewNop(), "./bin/ffmpeg")

	// Inject lavfi source directly via capture
	src := previewSource{kind: "lavfi", url: "testsrc=duration=1:size=320x240:rate=1"}
	data, err := svc.capture(context.Background(), src)
	require.NoError(t, err)
	require.NotEmpty(t, data)
	// JPEG magic: FF D8
	assert.Equal(t, uint8(0xFF), data[0])
	assert.Equal(t, uint8(0xD8), data[1])
}
